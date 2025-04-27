package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/linera-protocol/examples/shopify/client/docs"
	"github.com/linera-protocol/examples/shopify/client/shopify"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/swaggo/swag"
	"gopkg.in/yaml.v3"
)

// @title Shopify Client API
// @version 1.0
// @description This is a Go HTTP client server for the Shopify application in the Linera Protocol.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.linera.io/support
// @contact.email support@linera.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3007
// @BasePath /
// @schemes http https

// Config holds the application configuration
type Config struct {
	ShopifyURL     string
	WebhookURL     string
	Port           string
	AllowedOrigins map[string]bool
	Environment    string
}

// Logger provides structured logging functionality
type Logger struct {
	*log.Logger
}

// LogLevel defines the severity of log messages
type LogLevel string

const (
	LogLevelInfo  LogLevel = "INFO"
	LogLevelError LogLevel = "ERROR"
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelWarn  LogLevel = "WARN"
)

var (
	config        Config
	logger        *Logger
	shopifyClient *shopify.Client
	chainToToken  = map[string]string{
		"ethereum": "ETH",
		"solana":   "SOL",
	}
)

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
	}
}

// Log a message with the specified level
func (l *Logger) log(level LogLevel, format string, v ...interface{}) {
	l.Printf("[%s] %s", level, fmt.Sprintf(format, v...))
}

// Info logs information messages
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(LogLevelInfo, format, v...)
}

// Error logs error messages
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(LogLevelError, format, v...)
}

// Debug logs debug messages
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(LogLevelDebug, format, v...)
}

// Warn logs warning messages
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(LogLevelWarn, format, v...)
}

// Initialize configuration from environment variables or command-line flags
func initConfig() {
	// Define command line flags
	shopifyURL := flag.String("shopify-url", getEnvOrDefault("SHOPIFY_URL", "http://localhost:8081/"), "Shopify service URL")
	webhookURL := flag.String("webhook-url", getEnvOrDefault("WEBHOOK_URL", ""), "Webhook URL for notifications")
	port := flag.String("port", getEnvOrDefault("PORT", "3007"), "Server port")
	solanaRPCURL := flag.String("solana-url", getEnvOrDefault("SOLANA_RPC", "http://localhost:8899"), "Solana RPC endpoint")
	ethereumRPCURL := flag.String("ethereum-url", getEnvOrDefault("ETHEREUM_RPC", "http://localhost:8545"), "Ethereum RPC endpoint")
	environment := flag.String("env", getEnvOrDefault("ENVIRONMENT", "local"), "Environment (local, dev, prod)")

	flag.Parse()

	// Configure allowed origins based on environment
	allowedOrigins := map[string]bool{
		"http://localhost:5173":           true,
		"https://shopify-app.ngrok.io":    true,
		"http://localhost:3002":           true,
		"https://shopify-solver.ngrok.io": true,
	}

	// Set configuration
	config = Config{
		ShopifyURL:     *shopifyURL,
		WebhookURL:     *webhookURL,
		Port:           *port,
		AllowedOrigins: allowedOrigins,
		Environment:    *environment,
	}

	// Initialize shopify client
	shopify.InitLogger()
	shopifyClient = shopify.NewClient(config.ShopifyURL, "")

	// Initialize RPC endpoints and NFT address
	shopify.InitConfig(*ethereumRPCURL, *solanaRPCURL)

	// Log configuration
	logger.Info("Initialized with:")
	logger.Info("  Shopify URL: %s", config.ShopifyURL)
	logger.Info("  Webhook URL: %s", config.WebhookURL)
	logger.Info("  Port: %s", config.Port)
	logger.Info("  Environment: %s", config.Environment)
}

// Get environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// CORS middleware to handle cross-origin requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if config.AllowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Expose-Headers", "Set-Cookie")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Helper function to extract amount from transaction
func extractAmountFromTx(tx interface{}) (float64, error) {
	logger.Debug("Extracting amount from transaction: %+v", tx)

	switch v := tx.(type) {
	case map[string]interface{}:
		// For Ethereum
		if value, ok := v["value"].(string); ok {
			logger.Debug("Processing Ethereum transaction value: %s", value)
			// Parse decimal string to big.Int
			bigValue := new(big.Int)
			if _, success := bigValue.SetString(value, 10); !success {
				return 0, fmt.Errorf("failed to parse decimal value: %s", value)
			}
			// Convert from wei to ETH (divide by 10^18) and check if result fits uint64
			weiPerEth := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
			ethValue := new(big.Int).Div(bigValue, weiPerEth)
			if !ethValue.IsUint64() {
				return 0, fmt.Errorf("converted ETH value exceeds uint64 range: %s", ethValue.String())
			}
			flval, _ := ethValue.Float64()
			return flval, nil
		}
		// For Solana
		if result, ok := v["result"].(map[string]interface{}); ok {
			logger.Debug("Processing Solana transaction result")
			meta := result
			if meta, ok := meta["meta"].(map[string]interface{}); ok {
				if preBalances, ok := meta["preBalances"].([]interface{}); ok && len(preBalances) > 0 {
					if postBalances, ok := meta["postBalances"].([]interface{}); ok && len(postBalances) > 0 {
						// Get the difference between pre and post balances of sender
						preBalance := uint64(preBalances[0].(float64))
						postBalance := uint64(postBalances[0].(float64))
						if preBalance > postBalance {
							// Convert from lamports to SOL (divide by 10^9)
							lamports := preBalance - postBalance

							// Extract fee
							fee := uint64(0)
							if feeVal, ok := meta["fee"].(float64); ok {
								fee = uint64(feeVal)
							}

							// Subtract fee from total amount
							actualLamports := lamports - fee
							solValue := float64(actualLamports) / 1e9

							if solValue > float64(^uint64(0)) {
								return 0, fmt.Errorf("converted SOL value exceeds uint64 range: %f", solValue)
							}
							return solValue, nil
						}
					}
				}
			}
		}
	}
	logger.Error("Could not extract amount from transaction")
	return 0, fmt.Errorf("could not extract amount from transaction")
}

func getTokenForChain(chain string) (string, error) {
	logger.Debug("Getting token for chain: %s", chain)
	token, ok := chainToToken[chain]
	if !ok {
		logger.Error("Unsupported chain: %s", chain)
		return "", fmt.Errorf("unsupported chain: %s", chain)
	}
	logger.Debug("Found token: %s for chain: %s", token, chain)
	return token, nil
}

// @Summary Post transaction hash
// @Description Process a transaction hash from a blockchain
// @Tags transactions
// @Accept json
// @Produce json
// @Param txHash query string true "Transaction hash"
// @Param chain query string true "Blockchain chain (e.g. 'solana', 'ethereum')"
// @Param destinationAddress query string false "Destination address"
// @Param webhookURL query string false "Webhook URL for notifications"
// @Param sourceOwner query string false "Source owner"
// @Param tokenId query string false "Token ID"
// @Param blobHash query string false "Blob hash"
// @Param targetChainId query string false "Target chain ID"
// @Param targetOwner query string false "Target owner"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /post_tx_hash [post]
func handlePostTxHash(w http.ResponseWriter, r *http.Request) {
	logger.Info("Received POST request to /post_tx_hash")

	if r.Method != http.MethodPost {
		logger.Error("Invalid method %s for /post_tx_hash", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var (
		tx  interface{}
		err error
	)

	// Get parameters from query params
	txHash := r.URL.Query().Get("txHash")
	chain := r.URL.Query().Get("chain")
	destinationAddress := r.URL.Query().Get("destinationAddress")
	webhookURL := r.URL.Query().Get("webhookURL")

	logger.Debug("Request parameters - txHash: %s, chain: %s, destinationAddress: %s, webhookURL: %s",
		txHash, chain, destinationAddress, webhookURL)

	// Get additional transfer parameters
	sourceOwner := r.URL.Query().Get("sourceOwner")
	tokenId := r.URL.Query().Get("tokenId")
	blobHash := r.URL.Query().Get("blobHash")
	targetChainId := r.URL.Query().Get("targetChainId")
	targetOwner := r.URL.Query().Get("targetOwner")

	logger.Debug("Additional parameters - sourceOwner: %s, tokenId: %s, blobHash: %s, targetChainId: %s, targetOwner: %s",
		sourceOwner, tokenId, blobHash, targetChainId, targetOwner)

	// Validate required parameters
	if txHash == "" {
		logger.Error("Missing required parameter: txHash")
		http.Error(w, "txHash parameter is required", http.StatusBadRequest)
		return
	}

	if chain == "" {
		logger.Error("Missing required parameter: chain")
		http.Error(w, "chain parameter is required", http.StatusBadRequest)
		return
	}

	// Send immediate response to client
	response := map[string]interface{}{
		"status":  "processing",
		"message": "Transaction is being processed",
		"txHash":  txHash,
		"chain":   chain,
	}

	logger.Info("Sending initial response to client - status: processing")

	// Return response immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Create a channel for webhook notifications
	webhookChan := make(chan shopify.WebhookNotification, 1)

	// Start processing in a goroutine
	go func() {
		logger.Info("Starting transaction processing in background")

		// Get transaction details based on chain
		switch chain {
		case "solana":
			logger.Debug("Processing Solana transaction: %s", txHash)
			tx, err = shopifyClient.GetSolanaTransaction(shopify.SolanaRPC, txHash)
		case "ethereum":
			logger.Debug("Processing Ethereum transaction: %s", txHash)
			tx, err = shopifyClient.GetEthereumTransaction(shopify.EthereumRPC, txHash)
		default:
			logger.Error("Invalid chain parameter: %s", chain)
			errorNotification := shopify.WebhookNotification{
				Status:    "error",
				TxHash:    txHash,
				Chain:     chain,
				Timestamp: time.Now().Unix(),
				Data:      map[string]interface{}{"error": "Invalid chain parameter. Must be 'solana' or 'ethereum'"},
				Client:    "non-fungible",
			}
			webhookChan <- errorNotification
			return
		}

		if err != nil {
			logger.Error("Error getting transaction details: %v", err)
			errorNotification := shopify.WebhookNotification{
				Status:    "error",
				TxHash:    txHash,
				Chain:     chain,
				Timestamp: time.Now().Unix(),
				Data:      map[string]interface{}{"error": err.Error()},
				Client:    "non-fungible",
			}
			webhookChan <- errorNotification
			return
		}

		// Get the from token based on chain
		fromToken, err := getTokenForChain(chain)
		if err != nil {
			logger.Error("Error getting token for chain: %v", err)
			errorNotification := shopify.WebhookNotification{
				Status:    "error",
				TxHash:    txHash,
				Chain:     chain,
				Timestamp: time.Now().Unix(),
				Data:      map[string]interface{}{"error": err.Error()},
				Client:    "non-fungible",
			}
			webhookChan <- errorNotification
			return
		}

		logger.Debug("Retrieved fromToken: %s for chain: %s", fromToken, chain)

		// Extract amount from transaction
		amount, err := extractAmountFromTx(tx)
		if err != nil {
			logger.Error("Error extracting amount from transaction: %v", err)
			errorNotification := shopify.WebhookNotification{
				Status:    "error",
				TxHash:    txHash,
				Chain:     chain,
				FromToken: fromToken,
				Timestamp: time.Now().Unix(),
				Data:      map[string]interface{}{"error": err.Error()},
				Client:    "non-fungible",
			}
			webhookChan <- errorNotification
			return
		}

		logger.Debug("Extracted amount: %f from transaction", amount)

		// Execute transfer
		transferParams := shopify.TransferParams{
			SourceOwner:   sourceOwner,
			TokenId:       tokenId,
			TargetChainId: targetChainId,
			TargetOwner:   targetOwner,
			ChainOwner:    destinationAddress,
			BuyFromToken:  fromToken,
			Amount:        amount,
			BlobHash:      blobHash,
		}

		logger.Debug("Executing transfer with params: %+v", transferParams)

		transferResp, txhash, err := shopifyClient.ExecuteTransferMutation(transferParams)
		if err != nil {
			logger.Error("Error executing transfer: %v", err)
			errorNotification := shopify.WebhookNotification{
				Status:    "error",
				TxHash:    txHash,
				Chain:     chain,
				FromToken: fromToken,
				Amount:    amount,
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"error":          err.Error(),
					"transferParams": transferParams,
				},
				Client: "non-fungible",
			}
			webhookChan <- errorNotification
			return
		}

		logger.Info("Transfer executed successfully with hash: %s", txhash)

		// Send success notification
		notification := shopify.WebhookNotification{
			Status:    "success",
			TxHash:    txHash,
			Chain:     chain,
			FromToken: fromToken,
			Amount:    amount,
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"transferResult": transferResp.Data,
				"newTxHash":      txhash,
				"transferParams": transferParams,
			},
			Client: "non-fungible",
		}
		webhookChan <- notification

	}()

	// Start a goroutine to handle webhook notifications
	go func() {
		logger.Info("Starting webhook notification handler")

		// Wait for notification from the processing goroutine
		notification := <-webhookChan
		logger.Debug("Received notification for webhook: %+v", notification)

		// Send webhook notification if URL is provided
		if webhookURL != "" {
			if err := sendWebhookNotification(notification, webhookURL); err != nil {
				logger.Error("Error sending webhook notification: %v", err)
			} else {
				logger.Info("Successfully sent webhook notification to: %s", webhookURL)
			}
		}
	}()
}

// @Summary List an NFT
// @Description List a new NFT for sale
// @Tags items
// @Accept json
// @Produce json
// @Param request body object true "NFT details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /list_item [post]
func handleListNFT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request body
	var requestBody struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Price       string `json:"price"`
		ChainId     string `json:"chainId"`
		Minter      string `json:"minter"`
		ChainMinter string `json:"chainMinter"`
		ChainOwner  string `json:"chainOwner"`
		ID          int    `json:"id"`
		Token       string `json:"token"`
		BlobHash    string `json:"blobHash"`
		NftType     string `json:"nftType"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Error parsing request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create params
	params := shopify.ListNFTParams{
		Name:        requestBody.Name,
		Description: requestBody.Description,
		Price:       requestBody.Price,
		ChainId:     requestBody.ChainId,
		Minter:      requestBody.Minter,
		ChainMinter: requestBody.ChainMinter,
		ChainOwner:  requestBody.ChainOwner,
		Token:       requestBody.Token,
		BlobHash:    requestBody.BlobHash,
	}

	// List NFT and get blob hash
	blobHash, err := shopifyClient.ListNFT(params)
	if err != nil {
		http.Error(w, "Error listing NFT: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response with blob hash
	response := map[string]interface{}{
		"status":   "success",
		"message":  "NFT listed successfully",
		"blobHash": blobHash,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary Get all NFTs
// @Description Get all NFTs available in the system
// @Tags items
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /items/all [get]
func handleGetNFTs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nfts, err := shopifyClient.GetAllNFTs()
	if err != nil {
		http.Error(w, "Error getting NFTs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "success",
		"data":   nfts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary Get all balances
// @Description Get all token balances
// @Tags balances
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /balances [get]
func handleGetBalances(w http.ResponseWriter, r *http.Request) {
	logger.Info("Received GET request to /balances")

	if r.Method != http.MethodGet {
		logger.Error("Invalid method %s for /balances", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	balances, err := shopifyClient.GetBalances()
	if err != nil {
		logger.Error("Error getting balances: %v", err)
		http.Error(w, "Error getting balances: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":   "success",
		"balances": balances,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary Withdraw token
// @Description Withdraw tokens from the system
// @Tags balances
// @Accept json
// @Produce json
// @Param request body object true "Withdrawal details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /withdraw/token [post]
func handleWithdrawToken(w http.ResponseWriter, r *http.Request) {
	logger.Info("Received POST request to /withdraw/token")

	if r.Method != http.MethodPost {
		logger.Error("Invalid method %s for /withdraw/token", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var requestBody struct {
		Token  string `json:"token"`
		Amount string `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		logger.Error("Error parsing request body: %v", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required parameters
	if requestBody.Token == "" {
		logger.Error("Missing required parameter: token")
		http.Error(w, "token parameter is required", http.StatusBadRequest)
		return
	}

	if requestBody.Amount == "" {
		logger.Error("Missing required parameter: amount")
		http.Error(w, "amount parameter is required", http.StatusBadRequest)
		return
	}

	logger.Debug("Withdrawing %s %s", requestBody.Amount, requestBody.Token)

	// Execute withdrawal
	txHash, err := shopifyClient.WithdrawToken(requestBody.Token, requestBody.Amount)
	if err != nil {
		logger.Error("Error withdrawing token: %v", err)
		http.Error(w, "Error withdrawing token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"status":          "success",
		"message":         "Token withdrawn successfully",
		"transactionHash": txHash,
		"token":           requestBody.Token,
		"amount":          requestBody.Amount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary Filter items
// @Description Filter NFTs by type (ON_SALE or SOLD)
// @Tags items
// @Produce json
// @Param type query string false "Filter type (ON_SALE or SOLD)" default(ON_SALE)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /filter/items [get]
func handleFilterItems(w http.ResponseWriter, r *http.Request) {
	logger.Info("Received GET request to /filter/items")

	if r.Method != http.MethodGet {
		logger.Error("Invalid method %s for /filter/items", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the filter type from query parameter
	nftTypeStr := r.URL.Query().Get("type")
	if nftTypeStr == "" {
		nftTypeStr = "ON_SALE" // Default to ON_SALE if not specified
	}

	// Convert to uppercase for consistency
	nftTypeStr = strings.ToUpper(nftTypeStr)

	// Map from URL parameter to NFTType
	var nftType shopify.NFTType
	switch nftTypeStr {
	case "ONSALE", "ON_SALE":
		nftType = shopify.NFTTypeOnSale
	case "SOLD":
		nftType = shopify.NFTTypeSold
	default:
		logger.Error("Invalid NFT type: %s", nftTypeStr)
		http.Error(w, "Invalid NFT type. Must be 'ON_SALE' or 'SOLD'", http.StatusBadRequest)
		return
	}

	// Filter the NFTs
	nfts, err := shopifyClient.FilterNFTs(nftType)
	if err != nil {
		logger.Error("Error filtering NFTs: %v", err)
		http.Error(w, "Error filtering NFTs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the filtered NFTs
	response := map[string]interface{}{
		"status": "success",
		"type":   nftType,
		"count":  len(nfts),
		"items":  nfts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Add handler for getting balances
func sendWebhookNotification(notification shopify.WebhookNotification, webhookURL string) error {
	// Marshal notification to JSON
	jsonData, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("error marshaling notification: %v", err)
	}

	// Create POST request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

// customSwaggerHandler creates a custom Swagger UI handler that
// updates the Swagger config based on the current environment
func customSwaggerHandler() http.Handler {
	// Get the default SwaggerUI handler
	defaultHandler := httpSwagger.Handler(
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	)

	// Create a wrapper that modifies the Swagger JSON before serving
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If this is a request for the Swagger JSON/YAML, we need to modify it
		if strings.HasSuffix(r.URL.Path, "swagger.json") || strings.HasSuffix(r.URL.Path, "swagger.yaml") {
			// Create a custom response writer to capture the output
			responseRecorder := httptest.NewRecorder()

			// Call the original handler to get the Swagger JSON
			defaultHandler.ServeHTTP(responseRecorder, r)

			// Get the body from the recorder
			body := responseRecorder.Body.Bytes()

			// Parse the original Swagger document based on content type
			var swaggerDoc map[string]interface{}
			contentType := responseRecorder.Header().Get("Content-Type")

			if strings.Contains(contentType, "application/json") {
				if err := json.Unmarshal(body, &swaggerDoc); err != nil {
					http.Error(w, "Error parsing Swagger JSON", http.StatusInternalServerError)
					return
				}
			} else if strings.Contains(contentType, "application/yaml") || strings.Contains(contentType, "application/x-yaml") {
				if err := yaml.Unmarshal(body, &swaggerDoc); err != nil {
					http.Error(w, "Error parsing Swagger YAML", http.StatusInternalServerError)
					return
				}
			} else {
				// If not JSON or YAML, pass through unchanged
				for k, vs := range responseRecorder.Header() {
					for _, v := range vs {
						w.Header().Add(k, v)
					}
				}
				w.WriteHeader(responseRecorder.Code)
				w.Write(body)
				return
			}

			// Modify the host based on the environment
			switch config.Environment {
			case "local":
				swaggerDoc["host"] = fmt.Sprintf("localhost:%s", config.Port)
			case "dev":
				swaggerDoc["host"] = "dev.shopify-solver.ngrok.io"
			case "prod":
				swaggerDoc["host"] = "shopify-solver.ngrok.io"
			}

			// Set the proper response headers
			for k, vs := range responseRecorder.Header() {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}

			// Write the modified document back to the response
			var responseBody []byte
			var err error

			if strings.Contains(contentType, "application/json") {
				responseBody, err = json.Marshal(swaggerDoc)
			} else {
				responseBody, err = yaml.Marshal(swaggerDoc)
			}

			if err != nil {
				http.Error(w, "Error generating Swagger document", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(responseRecorder.Code)
			w.Write(responseBody)
			return
		}

		// For other requests (like UI files), pass through to the default handler
		defaultHandler.ServeHTTP(w, r)
	})
}

func main() {
	// Initialize the logger
	logger = NewLogger()
	logger.Info("Initializing Shopify client application...")

	// Initialize configuration
	initConfig()

	// Create router with CORS middleware
	router := mux.NewRouter()
	router.Use(corsMiddleware)

	// API routes
	router.HandleFunc("/list_item", handleListNFT).Methods("POST", "OPTIONS")
	router.HandleFunc("/post_tx_hash", handlePostTxHash).Methods("POST", "OPTIONS")
	router.HandleFunc("/items/all", handleGetNFTs).Methods("GET", "OPTIONS")
	router.HandleFunc("/balances", handleGetBalances).Methods("GET", "OPTIONS")
	router.HandleFunc("/withdraw/token", handleWithdrawToken).Methods("POST", "OPTIONS")
	router.HandleFunc("/filter/items", handleFilterItems).Methods("GET", "OPTIONS")

	// Swagger documentation
	router.PathPrefix("/swagger/").Handler(customSwaggerHandler())

	// Start server
	logger.Info("Server starting on :%s", config.Port)

	// Provide environment-specific access URLs
	switch config.Environment {
	case "local":
		logger.Info("API accessible at http://localhost:%s", config.Port)
		logger.Info("Swagger UI available at http://localhost:%s/swagger/index.html", config.Port)
	case "dev":
		logger.Info("API accessible at https://dev.shopify-solver.ngrok.io")
		logger.Info("Swagger UI available at https://dev.shopify-solver.ngrok.io/swagger/index.html")
	case "prod":
		logger.Info("API accessible at https://shopify-solver.ngrok.io")
		logger.Info("Swagger UI available at https://shopify-solver.ngrok.io/swagger/index.html")
	}

	if err := http.ListenAndServe(":"+config.Port, router); err != nil {
		logger.Error("Error starting server: %v", err)
		os.Exit(1)
	}
}

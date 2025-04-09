package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/linera-protocol/examples/universal-solver/client/solver"
)

// WebhookNotification represents a notification to be sent to a webhook
type WebhookNotification struct {
	Status      string      `json:"status"`
	TxHash      string      `json:"txHash"`
	Chain       string      `json:"chain"`
	FromAddress string      `json:"fromAddress"`
	FromToken   string      `json:"fromToken"`
	Amount      float64     `json:"amount"`
	Timestamp   int64       `json:"timestamp"`
	Data        interface{} `json:"data,omitempty"`
	Client      string      `json:"client"`
}

// sendWebhookNotification sends a notification to the specified webhook URL
func sendWebhookNotification(notification WebhookNotification, webhookURL string) error {
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

var (
	solverClient *solver.Client
	SolanaRPC    string
	EthereumRPC  string
	chainToToken = map[string]string{
		"ethereum": "ETH",
		"solana":   "SOL",
	}
)

func init() {
	// Initialize the logger
	logger = NewLogger()
	logger.Info("Initializing application...")
	initFlags()
}

func initFlags() {
	// Define command line flags
	solverURL := flag.String("solver-url", getEnvOrDefault("SOLVER_URL", "http://localhost:8080/"), "Universal Solver service URL")
	nonFungibleURL := flag.String("non-fungible-url", getEnvOrDefault("NON_FUNGIBLE_URL", "http://localhost:8081/"), "Non-Fungible service URL")
	lineraURL := flag.String("linera-url", getEnvOrDefault("LINERA_URL", "http://localhost:8080/"), "Linera service URL")
	solanaRPCURL := flag.String("solana-url", getEnvOrDefault("SOLANA_RPC", "http://localhost:8899"), "Solana RPC endpoint")
	ethereumRPCURL := flag.String("ethereum-url", getEnvOrDefault("ETHEREUM_RPC", "http://localhost:8545"), "Ethereum RPC endpoint")
	nftAddress := flag.String("nft-address", getEnvOrDefault("NFT_ADDRESS", ""), "NFT contract address")
	seedPhrase := flag.String("seed-phrase", "", "Seed phrase for deriving chain keys (required)")

	// Only parse flags if not running tests
	if !testing.Testing() {
		flag.Parse()

		// Validate required seed phrase
		if *seedPhrase == "" {
			fmt.Println("Usage:")
			fmt.Println("  -solver-url string")
			fmt.Println("        Universal Solver service URL (default: http://localhost:8080/)")
			fmt.Println("  -non-fungible-url string")
			fmt.Println("        Non-Fungible service URL (default: http://localhost:8081/)")
			fmt.Println("  -linera-url string")
			fmt.Println("        Linera service URL (default: http://localhost:8080/)")
			fmt.Println("  -solana-url string")
			fmt.Println("        Solana RPC endpoint (default: http://localhost:8899)")
			fmt.Println("  -ethereum-url string")
			fmt.Println("        Ethereum RPC endpoint (default: http://localhost:8545)")
			fmt.Println("  -nft-address string")
			fmt.Println("        NFT contract address")
			fmt.Println("  -seed-phrase string")
			fmt.Println("        Seed phrase for deriving chain keys (required)")
			os.Exit(1)
		}
	}

	// Initialize solver client with provided URLs
	solverClient = solver.NewClient(*solverURL, *nonFungibleURL, *lineraURL)

	// Initialize RPC endpoints and NFT address
	solver.InitConfig(*ethereumRPCURL, *solanaRPCURL, *nftAddress)

	// Initialize keys with seed phrase
	if err := solver.InitKeys(*seedPhrase); err != nil {
		log.Fatalf("Failed to initialize keys: %v", err)
	}

	solver.InitLogger()
	// Log configuration
	log.Printf("Initialized with:")
	log.Printf("  Solver URL: %s", *solverURL)
	log.Printf("  Non-Fungible URL: %s", *nonFungibleURL)
	log.Printf("  Linera URL: %s", *lineraURL)
	log.Printf("  Solana RPC: %s", *solanaRPCURL)
	log.Printf("  Ethereum RPC: %s", *ethereumRPCURL)
	log.Printf("  NFT Address: %s", *nftAddress)
	log.Printf("  Keys: Initialized successfully")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Add CORS middleware
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	// Define routes with CORS middleware
	http.HandleFunc("/post_tx_hash", corsMiddleware(handlePostTxHash))
	http.HandleFunc("/list_nft", corsMiddleware(handleListNFT))
	http.HandleFunc("/list_nft_for_sale", corsMiddleware(handleListNFTForSale))
	http.HandleFunc("/nfts", corsMiddleware(handleGetNFTs))
	http.HandleFunc("/publish_image", corsMiddleware(handleBlobHash))
	http.HandleFunc("/next_nft_id", corsMiddleware(handleNextNFTID))
	http.HandleFunc("/ws", corsMiddleware(handleWebSocket))

	// Start server
	port := getEnvOrDefault("PORT", "3000")
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

// Update the handlePostTxHash function
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
	toToken := r.URL.Query().Get("toToken")
	destinationAddress := r.URL.Query().Get("destinationAddress")
	webhookURL := r.URL.Query().Get("webhookURL")

	logger.Debug("Request parameters - txHash: %s, chain: %s, toToken: %s, destinationAddress: %s, webhookURL: %s",
		txHash, chain, toToken, destinationAddress, webhookURL)

	// Get additional transfer parameters
	sourceOwner := r.URL.Query().Get("sourceOwner")
	tokenId := r.URL.Query().Get("tokenId")
	blobHash := r.URL.Query().Get("blobHash")
	targetChainId := r.URL.Query().Get("targetChainId")
	targetOwner := r.URL.Query().Get("targetOwner")
	nftId := r.URL.Query().Get("nftId")

	logger.Debug("Additional parameters - sourceOwner: %s, tokenId: %s, blobHash: %s, targetChainId: %s, targetOwner: %s, nftId: %s",
		sourceOwner, tokenId, blobHash, targetChainId, targetOwner, nftId)

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
	webhookChan := make(chan WebhookNotification, 1)

	// Start processing in a goroutine
	go func() {
		logger.Info("Starting transaction processing in background")

		// Get transaction details based on chain
		switch chain {
		case "solana":
			logger.Debug("Processing Solana transaction: %s", txHash)
			tx, err = solverClient.GetSolanaTransaction(SolanaRPC, txHash)
		case "ethereum":
			logger.Debug("Processing Ethereum transaction: %s", txHash)
			tx, err = solverClient.GetEthereumTransaction(EthereumRPC, txHash)
		default:
			logger.Error("Invalid chain parameter: %s", chain)
			errorNotification := WebhookNotification{
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
			errorNotification := WebhookNotification{
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
			errorNotification := WebhookNotification{
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
			errorNotification := WebhookNotification{
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

		// Process transfer if toToken and destinationAddress are provided
		if toToken != "" && destinationAddress != "" {
			logger.Info("Processing transfer with toToken: %s and destinationAddress: %s", toToken, destinationAddress)

			// Calculate swap
			swapResult, err := solverClient.CalculateSwap(fromToken, toToken, amount)
			if err != nil {
				logger.Error("Error calculating swap: %v", err)
				errorNotification := WebhookNotification{
					Status:    "error",
					TxHash:    txHash,
					Chain:     chain,
					FromToken: fromToken,
					Amount:    amount,
					Timestamp: time.Now().Unix(),
					Data:      map[string]interface{}{"error": err.Error()},
					Client:    "non-fungible",
				}
				webhookChan <- errorNotification
				return
			}

			logger.Debug("Calculated swap result: %+v", swapResult)

			// Execute transfer
			transferParams := solver.TransferParams{
				SourceOwner:   sourceOwner,
				TokenId:       tokenId,
				TargetChainId: targetChainId,
				TargetOwner:   targetOwner,
				ChainOwner:    destinationAddress,
				BuyFromToken:  fromToken,
				ToToken:       toToken,
				Amount:        fmt.Sprintf("%f", swapResult.ToAmount),
				BlobHash:      blobHash,
				NftId:         nftId,
			}

			logger.Debug("Executing transfer with params: %+v", transferParams)

			transferResp, txhash, err := solverClient.ExecuteTransferMutation(transferParams)
			if err != nil {
				logger.Error("Error executing transfer: %v", err)
				errorNotification := WebhookNotification{
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
			notification := WebhookNotification{
				Status:    "success",
				TxHash:    txHash,
				Chain:     chain,
				FromToken: fromToken,
				Amount:    amount,
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"transferResult":  transferResp.Data,
					"swapCalculation": swapResult,
					"newTxHash":       txhash,
					"transferParams":  transferParams,
				},
				Client: "non-fungible",
			}
			webhookChan <- notification
		}
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

func handleBlobHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		ImageBytes []int  `json:"imageBytes"` // Expect array of integers
		ChainId    string `json:"chainId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Error parsing request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert int array to byte array
	imageBytes := make([]byte, len(requestBody.ImageBytes))
	for i, b := range requestBody.ImageBytes {
		imageBytes[i] = byte(b)
	}

	blobHash, err := solverClient.PublishDataBlob(requestBody.ChainId, imageBytes)
	if err != nil {
		http.Error(w, "Error publishing blob: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response with blob hash
	response := map[string]interface{}{
		"status":   "success",
		"message":  "Blob is published successfully",
		"blobHash": blobHash,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}

// Update handler for listing NFT to return blob hash
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
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Error parsing request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert int array to byte array
	// imageBytes := make([]byte, len(requestBody.ImageBytes))
	// for i, b := range requestBody.ImageBytes {
	// 	imageBytes[i] = byte(b)
	// }

	// Create params
	params := solver.ListNFTParams{
		Name:        requestBody.Name,
		Description: requestBody.Description,
		Price:       requestBody.Price,
		ChainId:     requestBody.ChainId,
		Minter:      requestBody.Minter,
		ChainMinter: requestBody.ChainMinter,
		ChainOwner:  requestBody.ChainOwner,
		ID:          requestBody.ID,
		Token:       requestBody.Token,
		BlobHash:    requestBody.BlobHash,
	}

	// List NFT and get blob hash
	blobHash, err := solverClient.ListNFT(params)
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

// Update the handleListNFTForSale function
func handleListNFTForSale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request body
	var requestBody struct {
		Owner      string `json:"owner"`
		ChainId    string `json:"chainId"`
		TokenId    string `json:"tokenId"`
		Price      string `json:"price"`
		NftId      string `json:"nftId"`
		ChainOwner string `json:"chainOwner"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Error parsing request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Call the ListNftForSale function with all parameters
	data, err := solverClient.ListNftForSale(
		requestBody.Owner,
		requestBody.ChainId,
		requestBody.TokenId,
		requestBody.Price,
		requestBody.NftId,
		requestBody.ChainOwner,
	)
	if err != nil {
		http.Error(w, "Error listing NFT for sale: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"status": "success",
		"data":   data,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Add handler for getting all NFTs
func handleGetNFTs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nfts, err := solverClient.GetAllNFTs()
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

// Add the handler function
func handleNextNFTID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current token ID
	currentID, err := solverClient.GetCurrentTokenID()
	if err != nil {
		http.Error(w, "Error getting next NFT ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// The next ID will be current + 1
	nextID := currentID + 1

	// Return success response
	response := map[string]interface{}{
		"status": "success",
		"data": map[string]uint64{
			"currentId": currentID,
			"nextId":    nextID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	solverClient.HandleWebSocket(w, r)
}

// Add after the imports
// Logger represents a custom logger with levels and formatting
type Logger struct {
	*log.Logger
}

// LogLevel represents different logging levels
type LogLevel string

const (
	INFO  LogLevel = "INFO"
	ERROR LogLevel = "ERROR"
	DEBUG LogLevel = "DEBUG"
	WARN  LogLevel = "WARN"
)

var logger *Logger

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds),
	}
}

// log formats and writes the log message with the specified level
func (l *Logger) log(level LogLevel, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.Printf("[%s] %s", level, msg)
}

// Info logs an info level message
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(INFO, format, v...)
}

// Error logs an error level message
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, format, v...)
}

// Debug logs a debug level message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, format, v...)
}

// Warn logs a warning level message
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(WARN, format, v...)
}

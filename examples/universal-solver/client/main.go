package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/linera-protocol/examples/universal-solver/client/solver"
)

var (
	solverClient *solver.Client
	githubClient *solver.GithubAuthConfig
	SolanaRPC    string
	EthereumRPC  string
	chainToToken = map[string]string{
		"ethereum": "ETH",
		"solana":   "SOL",
	}
	lineraPath string // Variable to hold the Linera execution path
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development/testing
	},
	HandshakeTimeout: 10 * time.Second,
}

func init() {
	initFlags()
	// Initialize the solver logger
	solver.InitLogger()
	// Initialize Linera configuration
	if err := solver.InitLineraConfig(); err != nil {
		log.Fatalf("Failed to initialize Linera configuration: %v", err)
	}
}

func initFlags() {
	// Define command line flags
	solverURL := flag.String("solver-url", getEnvOrDefault("SOLVER_URL", "http://localhost:8080/"), "Universal Solver service URL")
	solanaRPCURL := flag.String("solana-url", getEnvOrDefault("SOLANA_RPC", "http://localhost:8899"), "Solana RPC endpoint")
	ethereumRPCURL := flag.String("ethereum-url", getEnvOrDefault("ETHEREUM_RPC", "http://localhost:8545"), "Ethereum RPC endpoint")
	dojimaRPCURL := flag.String("dojima-url", getEnvOrDefault("DOJIMA_RPC", "http://localhost:8545"), "Dojima RPC endpoint")
	seedPhrase := flag.String("seed-phrase", "", "Seed phrase for deriving chain keys (required)")
	githubToken := flag.String("github-token", getEnvOrDefault("GITHUB_TOKEN", ""), "GitHub token for accessing example repositories")
	lineraPath := flag.String("linera-path", getEnvOrDefault("LINERA_PATH", ""), "Path to the Linera executable") // New flag for Linera path

	// Only parse flags if not running tests
	if !testing.Testing() {
		flag.Parse()

		// Validate required seed phrase
		if *seedPhrase == "" {
			fmt.Println("Usage:")
			fmt.Println("  -solver-url string")
			fmt.Println("        Universal Solver service URL (default: http://localhost:8080/)")
			fmt.Println("  -solana-url string")
			fmt.Println("        Solana RPC endpoint (default: http://localhost:8899)")
			fmt.Println("  -ethereum-url string")
			fmt.Println("        Ethereum RPC endpoint (default: http://localhost:8545)")
			fmt.Println("  -seed-phrase string")
			fmt.Println("        Seed phrase for deriving chain keys (required)")
			fmt.Println("  -github-token string")
			fmt.Println("        GitHub token for accessing example repositories")
			fmt.Println("  -linera-path string")
			fmt.Println("        Path to the Linera executable (required)")
			os.Exit(1)
		}

		// Validate Linera path
		if *lineraPath == "" {
			log.Fatal("Linera path must be provided")
		}

		// Validate GitHub token
		if *githubToken == "" {
			solver.Logger.Printf("Warning: GitHub token not provided. Example repositories may be inaccessible.")
		}
	}

	// Initialize solver client with provided URL and Linera path
	solverClient = solver.NewClient(*solverURL, *lineraPath)

	githubClient = solver.NewGithubClient(
		os.Getenv("GITHUB_CLIENT_ID"),
		os.Getenv("GITHUB_CLIENT_SECRET"),
		os.Getenv("GITHUB_REDIRECT_URI"),
		*githubToken,
	)

	// Initialize RPC endpoints
	solver.InitRPCEndpoints(*ethereumRPCURL, *solanaRPCURL, *dojimaRPCURL)

	// Initialize keys with seed phrase
	if err := solver.InitKeys(*seedPhrase); err != nil {
		log.Fatalf("Failed to initialize keys: %v", err)
	}

	// Log configuration (without exposing sensitive data)
	solver.Logger.Printf("Initialized with:")
	solver.Logger.Printf("  Solver URL: %s", *solverURL)
	solver.Logger.Printf("  Solana RPC: %s", *solanaRPCURL)
	solver.Logger.Printf("  Ethereum RPC: %s", *ethereumRPCURL)
	solver.Logger.Printf("  Linera Path: %s", *lineraPath)
	solver.Logger.Printf("  Keys: Initialized successfully")
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
		origin := r.Header.Get("Origin")
		allowedOrigins := map[string]bool{
			"http://localhost:3002":         true,
			"https://uni-solver.ngrok.io":   true,
			"https://market-place.ngrok.io": true,
		}

		if allowedOrigins[origin] {
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

		next(w, r)
	}
}

// LoggingMiddleware adds request logging to handlers
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		solver.Logger.Printf("Started %s %s", r.Method, r.URL.Path)

		// Create a custom response writer to capture status code
		rw := &responseWriter{w, http.StatusOK}
		next(rw, r)

		solver.Logger.Printf("Completed %s %s with status %d in %v",
			r.Method, r.URL.Path, rw.status, time.Since(start))
	}
}

// Custom response writer to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	// Define routes with CORS and logging middleware
	http.HandleFunc("/post_tx_hash", corsMiddleware(loggingMiddleware(handlePostTxHash)))
	http.HandleFunc("/faucet", corsMiddleware(loggingMiddleware(handleFaucet)))
	http.HandleFunc("/get_pool_address", corsMiddleware(loggingMiddleware(handleGetPoolAddress)))
	http.HandleFunc("/fetch_balance", corsMiddleware(loggingMiddleware(handleFetchBalance)))
	http.HandleFunc("/quote_swap", corsMiddleware(loggingMiddleware(handleQuoteSwap)))
	http.HandleFunc("/deploy_bytecode", corsMiddleware(loggingMiddleware(handleDeployBytecode)))
	http.HandleFunc("/create_application", corsMiddleware(loggingMiddleware(handleCreateApplication)))
	http.HandleFunc("/auth/github", corsMiddleware(loggingMiddleware(handleGithubAuth)))
	http.HandleFunc("/auth/github/callback", corsMiddleware(loggingMiddleware(handleGithubCallback)))
	http.HandleFunc("/repos", corsMiddleware(loggingMiddleware(handleListRepos)))
	http.HandleFunc("/repo/files", corsMiddleware(loggingMiddleware(handleFetchRepoFiles)))
	http.HandleFunc("/repo/all-files", corsMiddleware(loggingMiddleware(handleFetchAllFiles)))
	http.HandleFunc("/service/start", corsMiddleware(loggingMiddleware(handleStartService)))
	http.HandleFunc("/service/stop", corsMiddleware(loggingMiddleware(handleStopService)))
	http.HandleFunc("/service/status", corsMiddleware(loggingMiddleware(handleServiceStatus)))
	http.HandleFunc("/example_repos", corsMiddleware(loggingMiddleware(handleExampleRepos)))
	http.HandleFunc("/ws", handleWebSocket) // WebSocket endpoint

	// Start server
	port := getEnvOrDefault("PORT", "3001")
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func handlePostTxHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters from query params
	txHash := r.URL.Query().Get("txHash")
	chain := r.URL.Query().Get("chain")
	toToken := r.URL.Query().Get("toToken")
	destinationAddress := r.URL.Query().Get("destinationAddress")

	// Validate required parameters
	if txHash == "" {
		http.Error(w, "txHash parameter is required", http.StatusBadRequest)
		return
	}

	if chain == "" {
		http.Error(w, "chain parameter is required", http.StatusBadRequest)
		return
	}

	var (
		tx  interface{}
		err error
	)

	// Get transaction details based on chain
	switch chain {
	case "solana":
		tx, err = solverClient.GetSolanaTransaction(SolanaRPC, txHash)
	case "ethereum":
		tx, err = solverClient.GetEthereumTransaction(EthereumRPC, txHash)
	default:
		http.Error(w, "Invalid chain parameter. Must be 'solana' or 'ethereum'", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "Error getting transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "success",
		"chain":  chain,
		"data":   tx,
	}

	// If toToken and destinationAddress are provided, execute swap
	if toToken != "" && destinationAddress != "" {
		// Get the from token based on chain
		fromToken, err := getTokenForChain(chain)
		if err != nil {
			http.Error(w, "Error getting token for chain: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Extract amount from transaction
		amount, err := extractAmountFromTx(tx)
		if err != nil {
			http.Error(w, "Error extracting amount from transaction: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Execute swap with correct fromToken
		swapResponse, err := solverClient.ExecuteSwap(fromToken, toToken, float64(amount), destinationAddress)
		if err != nil {
			http.Error(w, "Error executing swap: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response["swap_result"] = swapResponse
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to extract amount from transaction
func extractAmountFromTx(tx interface{}) (uint64, error) {
	switch v := tx.(type) {
	case map[string]interface{}:
		// For Ethereum
		if value, ok := v["value"].(string); ok {
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
			return ethValue.Uint64(), nil
		}
		// For Solana
		if result, ok := v["result"].(map[string]interface{}); ok {
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
							solValue := float64(lamports) / 1e9
							if solValue > float64(^uint64(0)) {
								return 0, fmt.Errorf("converted SOL value exceeds uint64 range: %f", solValue)
							}
							return uint64(solValue), nil
						}
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("could not extract amount from transaction")
}

func getTokenForChain(chain string) (string, error) {
	token, ok := chainToToken[chain]
	if !ok {
		return "", fmt.Errorf("unsupported chain: %s", chain)
	}
	return token, nil
}

// Update handleFaucet to accept amount parameter
func handleFaucet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters
	chain := r.URL.Query().Get("chain")
	if chain == "" {
		http.Error(w, "chain parameter is required", http.StatusBadRequest)
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "address parameter is required", http.StatusBadRequest)
		return
	}

	// Get optional amount parameter
	amount := r.URL.Query().Get("amount")
	var amountFloat float64
	var err error
	if amount != "" {
		amountFloat, err = strconv.ParseFloat(amount, 64)
		if err != nil {
			http.Error(w, "Invalid amount value", http.StatusBadRequest)
			return
		}
	}

	var result map[string]interface{}

	switch chain {
	case "solana":
		if amount == "" {
			result, err = solverClient.RequestSolanaAirdrop(address)
		} else {
			result, err = solverClient.RequestSolanaAirdropWithAmount(address, amountFloat)
		}
	case "ethereum", "dojima":
		if amount == "" {
			result, err = solverClient.RequestEthereumFaucet(address, chain)
		} else {
			result, err = solverClient.RequestEthereumFaucetWithAmount(address, amountFloat, chain)
		}
	default:
		http.Error(w, "Invalid chain parameter. Must be 'solana' or 'ethereum'", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Error requesting faucet: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"chain":  chain,
		"data":   result,
	})
}

// Add new handler function
func handleGetPoolAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get chain parameter
	chain := r.URL.Query().Get("chain")
	if chain == "" {
		http.Error(w, "chain parameter is required", http.StatusBadRequest)
		return
	}

	// Get pool address for the chain
	poolAddress, err := solverClient.GetPool(chain)
	if err != nil {
		if err.Error() == fmt.Sprintf("pool not found for token: %s", chain) {
			http.Error(w, fmt.Sprintf("No pool found for chain: %s", chain), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Error fetching pool: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"chain":  chain,
		"data": map[string]interface{}{
			"address": poolAddress,
		},
	})
}

// -seed-phrase "indoor dish desk flag debris potato excuse depart ticket judge file exit" -solver-url https://linera-api.ngrok.io/chains/e476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65/applications/babb7d531b6ea2cb67be108f5a52e76e0a9e49fa1621cb1520bf0429c587f4f1477d9afa86a278c26978f4e8c5c9013c305f90ecb2c7b3deb53ef7c82d962f2fe476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65210000000000000000000000  -solana-url  "https://sol-test.dojima.network"  -ethereum-url  "https://eth-test.dojima.network"

func handleFetchBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters
	chain := r.URL.Query().Get("chain")
	if chain == "" {
		http.Error(w, "chain parameter is required", http.StatusBadRequest)
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "address parameter is required", http.StatusBadRequest)
		return
	}

	// Get balance based on chain
	var balance *solver.Balance
	var err error

	switch chain {
	case "solana":
		balance, err = solverClient.GetSolanaBalance(address)
	case "ethereum", "dojima":
		balance, err = solverClient.GetEthereumBalance(address, chain)
	default:
		http.Error(w, "Invalid chain parameter. Must be 'solana' or 'ethereum'", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching balance: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"chain":  chain,
		"data":   balance,
	})
}

func handleQuoteSwap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters
	fromChain := r.URL.Query().Get("fromChain")
	if fromChain == "" {
		http.Error(w, "fromChain parameter is required", http.StatusBadRequest)
		return
	}

	toChain := r.URL.Query().Get("toChain")
	if toChain == "" {
		http.Error(w, "toChain parameter is required", http.StatusBadRequest)
		return
	}

	fromAmount := r.URL.Query().Get("fromAmount")
	if fromAmount == "" {
		http.Error(w, "fromAmount parameter is required", http.StatusBadRequest)
		return
	}

	// Convert amount to float64
	amount, err := strconv.ParseFloat(fromAmount, 64)
	if err != nil {
		http.Error(w, "Invalid fromAmount value", http.StatusBadRequest)
		return
	}

	// Get quote using calculate swap
	quote, err := solverClient.CalculateSwap(fromChain, toChain, amount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error calculating swap: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"fromChain":    fromChain,
			"toChain":      toChain,
			"fromAmount":   amount,
			"toAmount":     quote.ToAmount,
			"exchangeRate": quote.ExchangeRate,
		},
	})
}

// StreamingRequest represents a streaming request with size information

func handleDeployBytecode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create buffered reader for the request body
	bodyReader := bufio.NewReaderSize(r.Body, 1024*1024) // 1MB buffer

	// Read the contract size header
	contractSizeStr, err := bodyReader.ReadString('|')
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading contract size: %v", err), http.StatusBadRequest)
		return
	}
	contractSize, err := strconv.ParseInt(strings.TrimSuffix(contractSizeStr, "|"), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid contract size: %v", err), http.StatusBadRequest)
		return
	}

	// Read the service size header
	serviceSizeStr, err := bodyReader.ReadString('|')
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading service size: %v", err), http.StatusBadRequest)
		return
	}
	serviceSize, err := strconv.ParseInt(strings.TrimSuffix(serviceSizeStr, "|"), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid service size: %v", err), http.StatusBadRequest)
		return
	}

	solver.Logger.Printf("Receiving WASM files - Contract: %d bytes, Service: %d bytes",
		contractSize, serviceSize)

	// Create temporary files with buffered writers
	contractFile, err := os.CreateTemp("/Users/luffybhaagi/RustroverProjects/linera-protocol-jvff/examples/universal-solver", "contract.wasm")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating temp file: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(contractFile.Name())
	defer contractFile.Close()

	serviceFile, err := os.CreateTemp("/Users/luffybhaagi/RustroverProjects/linera-protocol-jvff/examples/universal-solver", "service.wasm")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating temp file: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(serviceFile.Name())
	defer serviceFile.Close()

	// Create buffered writers
	contractWriter := bufio.NewWriterSize(contractFile, 1024*1024) // 1MB buffer
	serviceWriter := bufio.NewWriterSize(serviceFile, 1024*1024)   // 1MB buffer

	// Copy contract WASM with progress tracking
	contractWritten, err := io.CopyN(contractWriter, bodyReader, contractSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error writing contract WASM: %v", err), http.StatusInternalServerError)
		return
	}
	if err := contractWriter.Flush(); err != nil {
		http.Error(w, fmt.Sprintf("Error flushing contract WASM: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy service WASM with progress tracking
	serviceWritten, err := io.CopyN(serviceWriter, bodyReader, serviceSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error writing service WASM: %v", err), http.StatusInternalServerError)
		return
	}
	if err := serviceWriter.Flush(); err != nil {
		http.Error(w, fmt.Sprintf("Error flushing service WASM: %v", err), http.StatusInternalServerError)
		return
	}

	// Verify sizes
	if contractWritten != contractSize || serviceWritten != serviceSize {
		http.Error(w, "WASM file size mismatch", http.StatusBadRequest)
		return
	}

	solver.Logger.Printf("Successfully received WASM files - Contract: %d bytes, Service: %d bytes",
		contractWritten, serviceWritten)

	// Execute the publish bytecode command
	bytecodeID, err := solverClient.PublishBytecodeFromFiles(contractFile.Name(), serviceFile.Name())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error publishing bytecode: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"bytecodeId":   bytecodeID,
			"contractSize": contractWritten,
			"serviceSize":  serviceWritten,
		},
	})
}

func handleCreateApplication(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	solver.Logger.Printf("Received request to create application")

	if r.Method != http.MethodPost {
		solver.Logger.Printf("Invalid method %s for create application", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req struct {
		BytecodeID string `json:"bytecodeId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		solver.Logger.Printf("Error parsing request body: %v", err)
		http.Error(w, fmt.Sprintf("Error parsing request body: %v", err), http.StatusBadRequest)
		return
	}

	// Create application
	response, err := solverClient.CreateApplication(req.BytecodeID)
	if err != nil {
		solver.Logger.Printf("Error creating application: %v", err)
		http.Error(w, fmt.Sprintf("Error creating application: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"application_id": response.ApplicationID,
			"chain_id":       response.ChainID,
			"url":            response.URL,
		},
	})

	solver.Logger.Printf("Successfully created application (took %v)", time.Since(start))
}

func handleGithubAuth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	solver.Logger.Printf("Received GitHub auth request")

	if r.Method != http.MethodGet {
		solver.Logger.Printf("Invalid method %s for GitHub auth", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate random state for CSRF protection
	state := solver.GenerateRandomState()
	solver.Logger.Printf("Generated state token: %s", state)

	// Get the host from the request
	host := r.Host

	// Store state in cookie with correct domain
	stateCookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		Domain:   host,
		MaxAge:   3600,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
	http.SetCookie(w, stateCookie)

	// Redirect to GitHub OAuth with public_repo scope only
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=public_repo",
		githubClient.ClientID,
		githubClient.RedirectURI,
		state,
	)

	solver.Logger.Printf("Redirecting to GitHub OAuth (took %v)", time.Since(start))
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func handleGithubCallback(w http.ResponseWriter, r *http.Request) {
	solver.Logger.Printf("Received GitHub callback request")

	if r.Method != http.MethodGet {
		solver.Logger.Printf("Invalid method %s for GitHub callback", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		solver.Logger.Printf("Invalid state cookie: %v", err)
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != stateCookie.Value {
		solver.Logger.Printf("State mismatch: expected %s, got %s", stateCookie.Value, r.URL.Query().Get("state"))
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	// Exchange code for access token
	code := r.URL.Query().Get("code")
	token, err := githubClient.ExchangeCodeForToken(code)
	if err != nil {
		solver.Logger.Printf("Error exchanging code for token: %v", err)
		http.Error(w, fmt.Sprintf("Error exchanging code: %v", err), http.StatusInternalServerError)
		return
	}
	solver.Logger.Printf("Successfully exchanged code for token")

	// Get the host from the request
	host := r.Host

	// Set token in cookie with correct settings for cross-domain access
	cookie := &http.Cookie{
		Name:     "github_token",
		Value:    token,
		Path:     "/",
		Domain:   host, // Use the full ngrok domain
		MaxAge:   3600 * 24,
		HttpOnly: false,                 // Allow JavaScript access
		Secure:   true,                  // Required for HTTPS
		SameSite: http.SameSiteNoneMode, // Required for cross-site access
	}
	http.SetCookie(w, cookie)

	// Set state cookie with same settings
	stateCookie = &http.Cookie{
		Name:     "oauth_state",
		Value:    stateCookie.Value,
		Path:     "/",
		Domain:   host, // Use the full ngrok domain
		MaxAge:   3600,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
	http.SetCookie(w, stateCookie)

	// Redirect to frontend with success parameter
	frontendURL := "http://localhost:3002"
	if strings.Contains(r.Host, "uni-solver.ngrok.io") {
		frontendURL = "https://market-place.ngrok.io"
	}
	http.Redirect(w, r, frontendURL+"/?auth=success", http.StatusTemporaryRedirect)
}

func handleListRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get token from cookie
	tokenCookie, err := r.Cookie("github_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch repositories
	repos, err := solver.FetchGithubRepos(tokenCookie.Value)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching repos: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"repositories": repos,
		},
	})
}

func handleFetchRepoFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get token from cookie
	tokenCookie, err := r.Cookie("github_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get query parameters
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	path := r.URL.Query().Get("path")

	if owner == "" || repo == "" {
		http.Error(w, "owner and repo are required", http.StatusBadRequest)
		return
	}

	// Fetch repository contents
	contents, err := githubClient.FetchRepoContents(tokenCookie.Value, owner, repo, path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching repo contents: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"contents": contents,
		},
	})
}

func handleFetchAllFiles(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	solver.Logger.Printf("Received request to fetch all files")

	if r.Method != http.MethodGet {
		solver.Logger.Printf("Invalid method %s for fetch all files", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")

	if owner == "" || repo == "" {
		solver.Logger.Printf("Missing owner or repo parameters")
		http.Error(w, "owner and repo are required", http.StatusBadRequest)
		return
	}

	// Get token from cookie if not an example repo
	var token string
	if !githubClient.IsExampleRepo(owner, repo) {
		tokenCookie, err := r.Cookie("github_token")
		if err != nil {
			solver.Logger.Printf("No auth token available for non-example repo")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token = tokenCookie.Value
	}

	// Fetch all repository files recursively
	files, err := githubClient.FetchRepoFilesRecursively(token, owner, repo)
	if err != nil {
		solver.Logger.Printf("Error fetching repo files: %v", err)
		http.Error(w, fmt.Sprintf("Error fetching repo files: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"files": files,
		},
	}
	json.NewEncoder(w).Encode(response)

	solver.Logger.Printf("Successfully fetched %d files from %s/%s (took %v)",
		len(files), owner, repo, time.Since(start))
}

func handleStartService(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	solver.Logger.Printf("Received request to start Linera service")

	if r.Method != http.MethodPost {
		solver.Logger.Printf("Invalid method %s for start service", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req struct {
		Port int `json:"port"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		solver.Logger.Printf("Error parsing request body: %v", err)
		http.Error(w, fmt.Sprintf("Error parsing request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate port
	if req.Port <= 0 || req.Port > 65535 {
		solver.Logger.Printf("Invalid port number: %d", req.Port)
		http.Error(w, "Invalid port number", http.StatusBadRequest)
		return
	}

	// Start the service
	if err := solverClient.StartLineraService(req.Port); err != nil {
		solver.Logger.Printf("Error starting service: %v", err)
		http.Error(w, fmt.Sprintf("Error starting service: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"message": fmt.Sprintf("Linera service started on port %d", req.Port),
			"port":    req.Port,
		},
	}
	json.NewEncoder(w).Encode(response)

	solver.Logger.Printf("Successfully started Linera service on port %d (took %v)", req.Port, time.Since(start))
}

func handleStopService(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	solver.Logger.Printf("Received request to stop Linera service")

	if r.Method != http.MethodPost {
		solver.Logger.Printf("Invalid method %s for stop service", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Stop the service
	if err := solverClient.StopLineraService(); err != nil {
		solver.Logger.Printf("Error stopping service: %v", err)
		http.Error(w, fmt.Sprintf("Error stopping service: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"message": "Linera service stopped successfully",
		},
	}
	json.NewEncoder(w).Encode(response)

	solver.Logger.Printf("Successfully stopped Linera service (took %v)", time.Since(start))
}

func handleServiceStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	solver.Logger.Printf("Received request for service status")

	if r.Method != http.MethodGet {
		solver.Logger.Printf("Invalid method %s for service status", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get service status
	status := solverClient.GetServiceStatus()

	// Return response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status": "success",
		"data":   status,
	}
	json.NewEncoder(w).Encode(response)

	solver.Logger.Printf("Successfully returned service status (took %v)", time.Since(start))
}

func handleExampleRepos(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	solver.Logger.Printf("Received request for example repositories")

	if r.Method != http.MethodGet {
		solver.Logger.Printf("Invalid method %s for example repos", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Fetch example repositories
	repos, err := githubClient.FetchExampleRepos()
	if err != nil {
		solver.Logger.Printf("Error fetching example repos: %v", err)
		http.Error(w, fmt.Sprintf("Error fetching example repos: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"repositories": repos,
		},
	}
	json.NewEncoder(w).Encode(response)

	solver.Logger.Printf("Successfully returned %d example repos (took %v)", len(repos), time.Since(start))
}

// WebSocket handler
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("WebSocket connection attempt from %s", r.RemoteAddr)
	solverClient.HandleWebSocket(w, r)
}

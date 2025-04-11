package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	chainToToken = map[string]string{
		"ethereum": "ETH",
		"solana":   "SOL",
	}
	// RPC endpoints
	EthereumRPC string
	SolanaRPC   string
	CrowdSolver string // Variable to hold the crowd solver URL
	// HTTP client for making requests
	httpClient = &http.Client{}
)

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

func init() {
	// Initialize the logger
	logger = NewLogger()
	logger.Info("Initializing application...")
	initFlags()
}

func initFlags() {
	// Define command line flags
	solanaRPCURL := flag.String("solana-url", getEnvOrDefault("SOLANA_RPC", "http://localhost:8899"), "Solana RPC endpoint")
	ethereumRPCURL := flag.String("ethereum-url", getEnvOrDefault("ETHEREUM_RPC", "http://localhost:8545"), "Ethereum RPC endpoint")
	crowdSolverURL := flag.String("crowd-solver", getEnvOrDefault("CROWD_SOLVER_URL", "http://localhost:8080"), "Crowd solver URL")

	// Parse flags
	flag.Parse()

	// Initialize RPC endpoints and crowd solver URL
	SolanaRPC = *solanaRPCURL
	EthereumRPC = *ethereumRPCURL
	CrowdSolver = *crowdSolverURL

	// Log configuration
	logger.Info("Configuration:")
	logger.Info("  Solana RPC: %s", SolanaRPC)
	logger.Info("  Ethereum RPC: %s", EthereumRPC)
	logger.Info("  Crowd Solver URL: %s", CrowdSolver)
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
			"http://localhost:5173":         true,
			"https://market-place.ngrok.io": true,
			"http://localhost:3002":         true,
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
		logger.Info("Request started - Method: %s, Path: %s, RemoteAddr: %s",
			r.Method, r.URL.Path, r.RemoteAddr)

		// Create a custom response writer to capture status code
		rw := &responseWriter{w, http.StatusOK}
		next(rw, r)

		duration := time.Since(start)
		logger.Info("Request completed - Method: %s, Path: %s, Status: %d, Duration: %v",
			r.Method, r.URL.Path, rw.status, duration)
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

// ChainAddress represents a chain and address pair
type ChainAddress struct {
	Chain     string `json:"chain"`
	Address   string `json:"address"`
	TwitterId string `json:"twitter_id"`
}

// Global variable to store chain addresses
var chainAddresses []ChainAddress

// ChainAddressBalance represents a chain address with its balance
type ChainAddressBalance struct {
	Address string `json:"address"`
	Chain   string `json:"chain"`
}

// GraphQLResponse represents the response structure from the GraphQL query
type GraphQLResponse struct {
	Data struct {
		GetChainAddresses []ChainAddressBalance `json:"getChainAddresses"`
	} `json:"data"`
}

// ChainPledge represents a pledge with deposit address and amount
type ChainPledge struct {
	DepositAddress string `json:"depositAddress"`
	Amount         string `json:"amount"`
}

// ChainPledgesResponse represents the response structure from the GraphQL query
type ChainPledgesResponse struct {
	Data struct {
		GetChainPledges []ChainPledge `json:"getChainPledges"`
	} `json:"data"`
}

// ChainTotalPledge represents total pledges for a chain
type ChainTotalPledge struct {
	Chain  string `json:"chain"`
	Amount string `json:"amount"`
}

// TotalPledgesResponse represents the response structure from the GraphQL query
type TotalPledgesResponse struct {
	Data struct {
		GetTotalChainPledges []ChainTotalPledge `json:"getTotalChainPledges"`
	} `json:"data"`
}

// CollectResponse represents the response structure from the GraphQL mutation
type CollectResponse struct {
	Data struct {
		Collect bool `json:"collect"`
	} `json:"data"`
}

// TotalPledgeInUsdResponse represents the response structure from the GraphQL query
type TotalPledgeInUsdResponse struct {
	Data struct {
		TotalPledgeInUsd string `json:"totalPledgeInUsd"`
	} `json:"data"`
}

// NewCrowdAppResponse represents the response structure from the GraphQL mutation
type NewCrowdAppResponse struct {
	Data struct {
		NewCrowdApp bool `json:"newCrowdApp"`
	} `json:"data"`
}

// GetCrowdAppResponse represents the response structure from the GraphQL query
type GetCrowdAppResponse struct {
	Data struct {
		GetCrowdApp struct {
			Id                string             `json:"id"`
			Title             string             `json:"title"`
			Description       string             `json:"description"`
			ProfileHash       string             `json:"profileHash"`
			ProfileScreenshot []int              `json:"profileScreenshot"`
			Status            string             `json:"status"`
			ChainAddresses    []ChainAddress     `json:"chainAddresses"`
			TotalChainPledges []ChainTotalPledge `json:"totalChainPledges"`
			IndividualPledges []ChainPledge      `json:"individualPledges"`
		} `json:"getCrowdApp"`
	} `json:"data"`
}

// GetAllCrowdAppsResponse represents the response structure from the GraphQL query
type GetAllCrowdAppsResponse struct {
	Data struct {
		GetAllCrowdApps []struct {
			ID                string             `json:"id"`
			Title             string             `json:"title"`
			Description       string             `json:"description"`
			ProfileHash       string             `json:"profileHash"`
			ProfileScreenshot []int              `json:"profileScreenshot"`
			Status            string             `json:"status"`
			ChainAddresses    []ChainAddress     `json:"chainAddresses"`
			TotalChainPledges []ChainTotalPledge `json:"totalChainPledges"`
			IndividualPledges []ChainPledge      `json:"individualPledges"`
		} `json:"getAllCrowdApps"`
	} `json:"data"`
}

// EndpointInfo represents information about an available endpoint
type EndpointInfo struct {
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters,omitempty"`
}

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

func handleAddChainAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body as an array of ChainAddress
	var chainAddressArray []ChainAddress
	if err := json.NewDecoder(r.Body).Decode(&chainAddressArray); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(chainAddressArray) == 0 {
		http.Error(w, "Request body must contain at least one chain address", http.StatusBadRequest)
		return
	}

	successfulAdditions := []ChainAddress{}
	errors := []string{}

	// Process each chain address
	for _, chainAddr := range chainAddressArray {
		// Validate chain
		if chainAddr.Chain == "" {
			errors = append(errors, "Chain is required")
			continue
		}

		// Validate address
		if chainAddr.Address == "" {
			errors = append(errors, fmt.Sprintf("Address is required for chain %s", chainAddr.Chain))
			continue
		}

		if chainAddr.TwitterId == "" {
			errors = append(errors, fmt.Sprintf("twitter id is required for chain %s", chainAddr.Chain))
			continue
		}

		// Validate chain is supported
		if _, ok := chainToToken[chainAddr.Chain]; !ok {
			errors = append(errors, fmt.Sprintf("Unsupported chain: %s", chainAddr.Chain))
			continue
		}

		// Build GraphQL mutation
		mutation := fmt.Sprintf(`{"query":"mutation{addChain(twitterId:\"%s\",chainName:\"%s\",chainAddress:\"%s\")}"}`, chainAddr.TwitterId, chainToToken[chainAddr.Chain], chainAddr.Address)

		// Create request
		req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(mutation)))
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error creating request for %s: %s", chainAddr.Chain, err.Error()))
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		// Send request
		resp, err := httpClient.Do(req)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error sending request for %s: %s", chainAddr.Chain, err.Error()))
			continue
		}
		resp.Body.Close()

		// Add to the array
		chainAddresses = append(chainAddresses, chainAddr)
		successfulAdditions = append(successfulAdditions, chainAddr)
	}

	// Prepare response
	response := map[string]interface{}{
		"status":      "success",
		"message":     fmt.Sprintf("Processed %d chain addresses", len(chainAddressArray)),
		"successful":  successfulAdditions,
		"errors":      errors,
		"total_added": len(successfulAdditions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetChainAddresses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build GraphQL query
	query := `{"query":"query chainAddresses { getChainAddresses { address chain } }"}`

	// Create request
	req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(query)))
	if err != nil {
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "Error sending request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var graphqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		http.Error(w, "Error parsing response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"status":  "success",
		"message": "Chain addresses retrieved successfully",
		"data":    graphqlResp.Data.GetChainAddresses,
		"count":   len(graphqlResp.Data.GetChainAddresses),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetChainPledges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build GraphQL query
	query := `{"query":"query chainPledges { getChainPledges { depositAddress amount } }"}`

	// Create request
	req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(query)))
	if err != nil {
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "Error sending request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var graphqlResp ChainPledgesResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		http.Error(w, "Error parsing response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"status":  "success",
		"message": "Chain pledges retrieved successfully",
		"data":    graphqlResp.Data.GetChainPledges,
		"count":   len(graphqlResp.Data.GetChainPledges),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleTotalPledges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build GraphQL query
	query := `{"query":"query totalPledges { getTotalChainPledges { chain amount } }"}`

	// Create request
	req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(query)))
	if err != nil {
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "Error sending request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var graphqlResp TotalPledgesResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		http.Error(w, "Error parsing response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, pledge := range graphqlResp.Data.GetTotalChainPledges {
		amount := 0.0
		fmt.Sscanf(pledge.Amount, "%f", &amount)
	}

	// Prepare response
	response := map[string]interface{}{
		"status":  "success",
		"message": "Total chain pledges retrieved successfully",
		"data":    graphqlResp.Data.GetTotalChainPledges,
		"count":   len(graphqlResp.Data.GetTotalChainPledges),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleCollect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build GraphQL mutation
	mutation := `{"query":"mutation collect { collect }"}`

	// Create request
	req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(mutation)))
	if err != nil {
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "Error sending request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var graphqlResp CollectResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		http.Error(w, "Error parsing response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"status":    "success",
		"message":   "Collection process completed",
		"collected": graphqlResp.Data.Collect,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleTotalPledgeInUsd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract Twitter ID from query parameters
	twitterID := r.URL.Query().Get("twitterId")
	if twitterID == "" {
		http.Error(w, "Twitter ID is required", http.StatusBadRequest)
		return
	}

	// Build GraphQL query
	query := fmt.Sprintf(`{"query":"query totalPledgeInUsd { totalPledgeInUsd(twitterId:\"%s\") }"}`, twitterID)

	// Create request
	req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(query)))
	if err != nil {
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "Error sending request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var graphqlResp TotalPledgeInUsdResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		http.Error(w, "Error parsing response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert string amount to float for formatting
	amount := 0.0
	fmt.Sscanf(graphqlResp.Data.TotalPledgeInUsd, "%f", &amount)

	// Prepare response
	response := map[string]interface{}{
		"status":   "success",
		"message":  "Total pledge in USD retrieved successfully",
		"amount":   fmt.Sprintf("%f", amount),
		"currency": "USD",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleNewCrowdApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var requestBody struct {
		Args struct {
			Deadline int64  `json:"deadline"`
			Target   string `json:"target"`
		} `json:"args"`
		TwitterID         string `json:"twitterId"`
		ProfileScreenshot string `json:"profileScreenshot"`
		Title             string `json:"title"`
		Description       string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Build GraphQL mutation
	mutation := fmt.Sprintf(`{"query":"mutation CrowdApp { newCrowdApp(args:{deadline:4102473600000000, target:\"%s\"}, twitterId:\"%s\", profileScreenshot:\"%s\", title:\"%s\", description:\"%s\") }"}`,
		requestBody.Args.Target,
		requestBody.TwitterID,
		requestBody.ProfileScreenshot,
		requestBody.Title,
		requestBody.Description)

	// Create request
	req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(mutation)))
	if err != nil {
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "Error sending request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var graphqlResp struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		http.Error(w, "Error parsing response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"status":  "success",
		"message": "New crowd app created successfully",
		"data":    graphqlResp.Data,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetCrowdApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get twitterId from query parameters
	twitterID := r.URL.Query().Get("twitterId")
	if twitterID == "" {
		http.Error(w, "twitterId parameter is required", http.StatusBadRequest)
		return
	}

	query := fmt.Sprintf(`{"query":"query getCrowdApp { getCrowdApp(twitterId:\"%s\") { id title description status profileHash profileScreenshot chainAddresses { chain address } totalChainPledges { amount chain } individualPledges { depositAddress amount } } }"}`, twitterID)

	// Create request
	req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(query)))
	if err != nil {
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "Error sending request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var graphqlResp GetCrowdAppResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		http.Error(w, "Error parsing response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"status":  "success",
		"message": "Crowd app data retrieved successfully",
		"data":    graphqlResp.Data.GetCrowdApp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetAllCrowdApps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := `{"query":"query getApps { getAllCrowdApps { id title description profileHash profileScreenshot status chainAddresses { chain address } totalChainPledges { amount chain } individualPledges { depositAddress amount } } }"}`

	// Create request
	req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(query)))
	if err != nil {
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, "Error sending request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var graphqlResp GetAllCrowdAppsResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		http.Error(w, "Error parsing response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"status":  "success",
		"message": "All crowd apps retrieved successfully",
		"data":    graphqlResp.Data.GetAllCrowdApps,
		"count":   len(graphqlResp.Data.GetAllCrowdApps),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleAvailableEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Define all available endpoints
	endpoints := []EndpointInfo{
		{
			Path:        "/post_tx_hash",
			Method:      "POST",
			Description: "Process a transaction hash and fund the crowd app",
			Parameters:  []string{"txHash", "chain", "webhook"},
		},
		{
			Path:        "/add_chain_address",
			Method:      "POST",
			Description: "Add a new chain address for a crowd app",
			Parameters:  []string{"chain", "address"},
		},
		{
			Path:        "/chain_addresses",
			Method:      "GET",
			Description: "Get all chain addresses",
		},
		{
			Path:        "/chain_pledges",
			Method:      "GET",
			Description: "Get all chain pledges",
		},
		{
			Path:        "/total_pledges",
			Method:      "GET",
			Description: "Get total pledges across all chains",
		},
		{
			Path:        "/collect",
			Method:      "POST",
			Description: "Collect funds from all pledges",
		},
		{
			Path:        "/pledge_in_usd",
			Method:      "GET",
			Description: "Get total pledge amount in USD",
			Parameters:  []string{"twitterId"},
		},
		{
			Path:        "/crowd_app/new",
			Method:      "POST",
			Description: "Create a new crowd app",
			Parameters:  []string{"args.deadline", "args.target", "twitterId"},
		},
		{
			Path:        "/crowd_app/get",
			Method:      "GET",
			Description: "Get crowd app details by Twitter ID",
			Parameters:  []string{"twitterId"},
		},
		{
			Path:        "/crowd_app/all",
			Method:      "GET",
			Description: "Get all crowd apps",
		},
		{
			Path:        "/available",
			Method:      "GET",
			Description: "List all available endpoints",
		},
	}

	// Prepare response
	response := map[string]interface{}{
		"status":    "success",
		"message":   "Available endpoints retrieved successfully",
		"endpoints": endpoints,
		"count":     len(endpoints),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Define routes with CORS and logging middleware
	http.HandleFunc("/post_tx_hash", corsMiddleware(loggingMiddleware(handlePostTxHash)))
	http.HandleFunc("/add_chain_address", corsMiddleware(loggingMiddleware(handleAddChainAddress)))
	http.HandleFunc("/chain_addresses", corsMiddleware(loggingMiddleware(handleGetChainAddresses)))
	http.HandleFunc("/chain_pledges", corsMiddleware(loggingMiddleware(handleGetChainPledges)))
	http.HandleFunc("/total_pledges", corsMiddleware(loggingMiddleware(handleTotalPledges)))
	http.HandleFunc("/collect", corsMiddleware(loggingMiddleware(handleCollect)))
	http.HandleFunc("/pledge_in_usd", corsMiddleware(loggingMiddleware(handleTotalPledgeInUsd)))
	http.HandleFunc("/crowd_app/new", corsMiddleware(loggingMiddleware(handleNewCrowdApp)))
	http.HandleFunc("/crowd_app/get", corsMiddleware(loggingMiddleware(handleGetCrowdApp)))
	http.HandleFunc("/crowd_app/all", corsMiddleware(loggingMiddleware(handleGetAllCrowdApps)))
	http.HandleFunc("/available", corsMiddleware(loggingMiddleware(handleAvailableEndpoints)))

	// Start server
	port := getEnvOrDefault("PORT", "3003")
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

// makeRPCRequest makes a JSON-RPC request to the specified endpoint
func makeRPCRequest(endpoint string, requestBody interface{}) (interface{}, error) {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetSolanaTransaction fetches transaction details from Solana
func GetSolanaTransaction(txHash string) (interface{}, error) {
	// Prepare the JSON-RPC request
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTransaction",
		"params": []interface{}{
			txHash,
			map[string]interface{}{
				"encoding":                       "json",
				"maxSupportedTransactionVersion": 0,
			},
		},
	}

	// Make the request with retries
	var response interface{}
	var err error
	for i := 0; i < 20; i++ {
		logger.Debug("Retrying %v time", i)
		response, err = makeRPCRequest(SolanaRPC, requestBody)
		if responseMap, ok := response.(map[string]interface{}); ok {
			if responseMap["result"] == nil {
				time.Sleep(5 * time.Second)
				continue // Retry if result is nil
			}
		}

		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get Solana transaction after 20 retries: %w", err)
	}

	return response, nil
}

// GetEthereumTransaction fetches transaction details from Ethereum
func GetEthereumTransaction(txHash string) (interface{}, error) {
	client, err := ethclient.Dial(EthereumRPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}
	defer client.Close()

	hash := common.HexToHash(txHash)
	tx, isPending, err := client.TransactionByHash(context.Background(), hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get Ethereum transaction: %w", err)
	}

	// Get the sender address
	from, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender address: %w", err)
	}

	// Convert transaction to map for consistent response format
	return map[string]interface{}{
		"hash":      tx.Hash().Hex(),
		"value":     tx.Value().String(),
		"gas":       tx.Gas(),
		"gasPrice":  tx.GasPrice().String(),
		"nonce":     tx.Nonce(),
		"isPending": isPending,
		"from":      from.Hex(), // Add the from address
	}, nil
}

// Helper function to get token for chain
func getTokenForChain(chain string) (string, error) {
	token, ok := chainToToken[chain]
	if !ok {
		return "", fmt.Errorf("unsupported chain: %s", chain)
	}
	return token, nil
}

// Helper function to extract amount from transaction
func extractAmountFromTx(tx interface{}) (float64, error) {
	switch v := tx.(type) {
	case map[string]interface{}:
		// For Ethereum
		if value, ok := v["value"].(string); ok {
			// Parse decimal string to big.Int
			bigValue := new(big.Int)
			if _, success := bigValue.SetString(value, 10); !success {
				return 0, fmt.Errorf("failed to parse decimal value: %s", value)
			}

			// Convert from wei to ETH by dividing by 10^18
			weiPerEth := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

			// Convert to float64 before division to preserve decimal places
			fValue, _ := new(big.Float).SetInt(bigValue).Float64()
			fWeiPerEth, _ := new(big.Float).SetInt(weiPerEth).Float64()

			ethValue := fValue / fWeiPerEth
			return ethValue, nil
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
	return 0, fmt.Errorf("could not extract amount from transaction")
}

// Helper function to extract from address from transaction
func extractFromAddress(tx interface{}, chain string) (string, error) {
	switch chain {
	case "ethereum":
		if txMap, ok := tx.(map[string]interface{}); ok {
			if from, ok := txMap["from"].(string); ok {
				return from, nil
			}
		}
	case "solana":
		if txMap, ok := tx.(map[string]interface{}); ok {
			if result, ok := txMap["result"].(map[string]interface{}); ok {
				if transaction, ok := result["transaction"].(map[string]interface{}); ok {
					if message, ok := transaction["message"].(map[string]interface{}); ok {
						if accountKeys, ok := message["accountKeys"].([]interface{}); ok && len(accountKeys) > 1 {
							// The first account key is typically the sender
							if from, ok := accountKeys[0].(string); ok {
								return from, nil
							}
						}
					}
				}
			}
		}
	}
	return "", fmt.Errorf("could not extract from address from transaction")
}

// sendWebhookNotification sends a notification to the specified webhook URL
func sendWebhookNotification(webhookURL string, notification WebhookNotification) error {
	if webhookURL == "" {
		return nil // No webhook URL provided, skip notification
	}

	// Marshal notification to JSON
	jsonData, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("error marshaling webhook notification: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	logger.Info("Webhook notification sent successfully to %s", webhookURL)
	return nil
}

func handlePostTxHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logger.Error("Invalid method %s for /post_tx_hash", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters from query params
	txHash := r.URL.Query().Get("txHash")
	chain := r.URL.Query().Get("chain")
	twitterId := r.URL.Query().Get("twitterId")
	webhookURL := r.URL.Query().Get("webhook") // Get webhook URL from query params

	logger.Debug("Processing transaction - Hash: %s, Chain: %s", txHash, chain)

	// Validate required parameters
	if txHash == "" {
		logger.Error("Missing txHash parameter")
		http.Error(w, "txHash parameter is required", http.StatusBadRequest)
		return
	}

	if chain == "" {
		logger.Error("Missing chain parameter")
		http.Error(w, "chain parameter is required", http.StatusBadRequest)
		return
	}

	if twitterId == "" {
		logger.Error("Missing twitterId parameter")
		http.Error(w, "twitterId parameter is required", http.StatusBadRequest)
		return
	}

	// Send immediate response to client
	response := map[string]interface{}{
		"status":  "processing",
		"message": "Transaction is successfully processed for completion",
		"txHash":  txHash,
		"chain":   chain,
	}

	// Return response immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Create a channel for webhook notifications
	webhookChan := make(chan WebhookNotification, 1)

	// Start processing in a goroutine
	go func() {
		var (
			tx  interface{}
			err error
		)

		// Get transaction details based on chain
		switch chain {
		case "solana":
			logger.Debug("Fetching Solana transaction: %s", txHash)
			tx, err = GetSolanaTransaction(txHash)
		case "ethereum":
			logger.Debug("Fetching Ethereum transaction: %s", txHash)
			tx, err = GetEthereumTransaction(txHash)
		default:
			logger.Error("Invalid chain parameter: %s", chain)
			errorNotification := WebhookNotification{
				Status:      "error",
				TxHash:      txHash,
				Chain:       chain,
				FromAddress: "",
				FromToken:   "",
				Amount:      0,
				Timestamp:   time.Now().Unix(),
				Data:        map[string]interface{}{"error": "Invalid chain parameter. Must be 'solana' or 'ethereum'"},
				Client:      "crowd-funding",
			}
			webhookChan <- errorNotification
			return
		}

		if err != nil {
			logger.Error("Error getting transaction: %v", err)

			// Create error webhook notification
			errorNotification := WebhookNotification{
				Status:      "error",
				TxHash:      txHash,
				Chain:       chain,
				FromAddress: "",
				FromToken:   "",
				Amount:      0,
				Timestamp:   time.Now().Unix(),
				Data:        map[string]interface{}{"error": err.Error()},
				Client:      "crowd-funding",
			}
			webhookChan <- errorNotification
			return
		}

		// Extract from address
		fromAddress, err := extractFromAddress(tx, chain)
		if err != nil {
			logger.Error("Error extracting from address: %v", err)

			// Create error webhook notification
			errorNotification := WebhookNotification{
				Status:      "error",
				TxHash:      txHash,
				Chain:       chain,
				FromAddress: "",
				FromToken:   "",
				Amount:      0,
				Timestamp:   time.Now().Unix(),
				Data:        map[string]interface{}{"error": err.Error()},
				Client:      "crowd-funding",
			}
			webhookChan <- errorNotification
			return
		}

		// Get the from token based on chain
		fromToken, err := getTokenForChain(chain)
		if err != nil {
			logger.Error("Error getting token for chain: %v", err)

			// Create error webhook notification
			errorNotification := WebhookNotification{
				Status:      "error",
				TxHash:      txHash,
				Chain:       chain,
				FromAddress: fromAddress,
				FromToken:   "",
				Amount:      0,
				Timestamp:   time.Now().Unix(),
				Data:        map[string]interface{}{"error": err.Error()},
				Client:      "crowd-funding",
			}
			webhookChan <- errorNotification
			return
		}

		// Extract amount from transaction
		amount, err := extractAmountFromTx(tx)
		if err != nil {
			logger.Error("Error extracting amount from transaction: %v", err)

			// Create error webhook notification
			errorNotification := WebhookNotification{
				Status:      "error",
				TxHash:      txHash,
				Chain:       chain,
				FromAddress: fromAddress,
				FromToken:   fromToken,
				Amount:      0,
				Timestamp:   time.Now().Unix(),
				Data:        map[string]interface{}{"error": err.Error()},
				Client:      "crowd-funding",
			}
			webhookChan <- errorNotification
			return
		}

		// Convert amount to string
		amountStr := fmt.Sprintf("%f", amount)

		logger.Info("Transaction processed successfully - Hash: %s, Chain: %s, From: %s, Amount: %s %s",
			txHash, chain, fromAddress, amountStr, fromToken)

		// Build GraphQL mutation
		mutation := fmt.Sprintf(`{"query":"mutation calFund{fund(twitterId:\"%s\",chainName:\"%s\",depositAddress:\"%s\",amount:\"%s\")}"}`,
			twitterId, fromToken, fromAddress, amountStr)

		logger.Debug("Sending GraphQL mutation: %s", mutation)

		// Create request
		req, err := http.NewRequest("POST", CrowdSolver, bytes.NewBuffer([]byte(mutation)))
		if err != nil {
			logger.Error("Error creating GraphQL request: %v", err)

			// Create error webhook notification
			errorNotification := WebhookNotification{
				Status:      "error",
				TxHash:      txHash,
				Chain:       chain,
				FromAddress: fromAddress,
				FromToken:   fromToken,
				Amount:      amount,
				Timestamp:   time.Now().Unix(),
				Data:        map[string]interface{}{"error": err.Error()},
				Client:      "crowd-funding",
			}
			webhookChan <- errorNotification
			return
		}

		req.Header.Set("Content-Type", "application/json")

		// Send request
		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Error("Error sending GraphQL request: %v", err)

			// Create error webhook notification
			errorNotification := WebhookNotification{
				Status:      "error",
				TxHash:      txHash,
				Chain:       chain,
				FromAddress: fromAddress,
				FromToken:   fromToken,
				Amount:      amount,
				Timestamp:   time.Now().Unix(),
				Data:        map[string]interface{}{"error": err.Error()},
				Client:      "crowd-funding",
			}
			webhookChan <- errorNotification
			return
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			logger.Error("Error response from GraphQL endpoint: %d", resp.StatusCode)

			// Create error webhook notification
			errorNotification := WebhookNotification{
				Status:      "error",
				TxHash:      txHash,
				Chain:       chain,
				FromAddress: fromAddress,
				FromToken:   fromToken,
				Amount:      amount,
				Timestamp:   time.Now().Unix(),
				Data:        map[string]interface{}{"error": fmt.Sprintf("GraphQL endpoint returned status: %d", resp.StatusCode)},
				Client:      "crowd-funding",
			}
			webhookChan <- errorNotification
			return
		}

		logger.Info("Successfully processed fund request - Chain: %s, From: %s, Amount: %s %s",
			chain, fromAddress, amountStr, fromToken)

		// Create webhook notification
		notification := WebhookNotification{
			Status:      "success",
			TxHash:      txHash,
			Chain:       chain,
			FromAddress: fromAddress,
			FromToken:   fromToken,
			Amount:      amount,
			Timestamp:   time.Now().Unix(),
			Data:        tx,
			Client:      "crowd-funding", // Add client identifier
		}
		webhookChan <- notification
	}()

	// Start a goroutine to handle webhook notifications
	go func() {
		// Wait for notification from the processing goroutine
		notification := <-webhookChan

		// Send webhook notification if URL is provided
		if webhookURL != "" {
			if err := sendWebhookNotification(webhookURL, notification); err != nil {
				logger.Error("Error sending webhook notification: %v", err)
			}
		}
	}()
}

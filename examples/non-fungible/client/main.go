package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"testing"

	"github.com/linera-protocol/examples/universal-solver/client/solver"
)

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
	http.HandleFunc("/nfts", corsMiddleware(handleGetNFTs))

	// Start server
	port := getEnvOrDefault("PORT", "3000")
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

// Update the handlePostTxHash function
func handlePostTxHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	// Get additional transfer parameters
	sourceOwner := r.URL.Query().Get("sourceOwner")
	tokenId := r.URL.Query().Get("tokenId")

	if err != nil {
		http.Error(w, "Invalid tokenId", http.StatusBadRequest)
		return
	}
	targetChainId := r.URL.Query().Get("targetChainId")
	targetOwner := r.URL.Query().Get("targetOwner")

	// Validate required parameters
	if txHash == "" {
		http.Error(w, "txHash parameter is required", http.StatusBadRequest)
		return
	}

	if chain == "" {
		http.Error(w, "chain parameter is required", http.StatusBadRequest)
		return
	}

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

	// If toToken and destinationAddress are provided, execute transfer
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

		// First calculate the swap
		swapResult, err := solverClient.CalculateSwap(fromToken, toToken, amount)
		if err != nil {
			http.Error(w, "Error calculating swap: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Use provided parameters or defaults
		transferParams := solver.TransferParams{
			SourceOwner:   sourceOwner,
			TokenId:       tokenId,
			TargetChainId: targetChainId,
			TargetOwner:   targetOwner,
			ChainOwner:    destinationAddress,
			BuyFromToken:  fromToken,
			ToToken:       toToken,
			Amount:        fmt.Sprintf("%f", swapResult.ToAmount), // Use calculated amount
		}

		// Execute transfer mutation with swap result
		transferResp, err := solverClient.ExecuteTransferMutation(transferParams)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response["transfer_result"] = transferResp.Data
		response["swap_calculation"] = swapResult
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
							return solValue, nil
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

// Add handler for listing NFT
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
		ImageBytes  []int  `json:"imageBytes"` // Expect array of integers
		ChainId     string `json:"chainId"`
		Minter      string `json:"minter"`
		ChainMinter string `json:"chainMinter"`
		ChainOwner  string `json:"chainOwner"`
		ID          int    `json:"id"`
		Token       string `json:"token"`
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

	// Create params
	params := solver.ListNFTParams{
		Name:        requestBody.Name,
		Description: requestBody.Description,
		Price:       requestBody.Price,
		ImageBytes:  imageBytes,
		ChainId:     requestBody.ChainId,
		Minter:      requestBody.Minter,
		ChainMinter: requestBody.ChainMinter,
		ChainOwner:  requestBody.ChainOwner,
		ID:          requestBody.ID,
		Token:       requestBody.Token,
	}

	// List NFT
	if err := solverClient.ListNFT(params); err != nil {
		http.Error(w, "Error listing NFT: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]string{
		"status":  "success",
		"message": "NFT listed successfully",
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

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
	solanaRPCURL := flag.String("solana-url", getEnvOrDefault("SOLANA_RPC", "http://localhost:8899"), "Solana RPC endpoint")
	ethereumRPCURL := flag.String("ethereum-url", getEnvOrDefault("ETHEREUM_RPC", "http://localhost:8545"), "Ethereum RPC endpoint")
	seedPhrase := flag.String("seed-phrase", "", "Seed phrase for deriving chain keys (required)")

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
			os.Exit(1)
		}
	}

	// Initialize solver client with provided URL
	solverClient = solver.NewClient(*solverURL)

	// Initialize RPC endpoints
	solver.InitRPCEndpoints(*ethereumRPCURL, *solanaRPCURL)

	// Initialize keys with seed phrase
	if err := solver.InitKeys(*seedPhrase); err != nil {
		log.Fatalf("Failed to initialize keys: %v", err)
	}

	// Log configuration (without exposing seed phrase)
	log.Printf("Initialized with:")
	log.Printf("  Solver URL: %s", *solverURL)
	log.Printf("  Solana RPC: %s", *solanaRPCURL)
	log.Printf("  Ethereum RPC: %s", *ethereumRPCURL)
	log.Printf("  Keys: Initialized successfully")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Define routes
	http.HandleFunc("/post_tx_hash", handlePostTxHash)

	// Start server
	port := getEnvOrDefault("PORT", "3000")
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

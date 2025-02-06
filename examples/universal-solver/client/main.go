package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/linera-protocol/examples/universal-solver/client/solver"
)

var solverClient *solver.Client

func main() {
	// Initialize solver client
	solverClient = solver.NewClient("http://localhost:8080/graphql")

	// Define routes
	http.HandleFunc("/post_tx_hash", handlePostTxHash)

	// Start server
	log.Printf("Server starting on :3000")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func handlePostTxHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get txHash from query params
	txHash := r.URL.Query().Get("txHash")
	if txHash == "" {
		http.Error(w, "txHash parameter is required", http.StatusBadRequest)
		return
	}

	// Get transaction details
	tx, err := solverClient.GetTransactionByHash(txHash)
	if err != nil {
		http.Error(w, "Error getting transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   tx,
	})
}

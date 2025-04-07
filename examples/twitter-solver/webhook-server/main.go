package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// WebhookNotification represents a notification to be sent to a webhook
type WebhookNotification struct {
	Status      string      `json:"status"`
	TxHash      string      `json:"txHash"`
	Chain       string      `json:"chain"`
	FromAddress string      `json:"fromAddress"`
	FromToken   string      `json:"fromToken"`
	Amount      string      `json:"amount"`
	Timestamp   string      `json:"timestamp"`
	Data        interface{} `json:"data,omitempty"`
	Client      string      `json:"client,omitempty"` // Which client sent this notification
}

// WebhookServer represents the webhook server
type WebhookServer struct {
	webhooks []WebhookNotification
	mu       sync.RWMutex
	port     int
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(port int) *WebhookServer {
	return &WebhookServer{
		webhooks: make([]WebhookNotification, 0),
		port:     port,
	}
}

// AddWebhook adds a webhook to the server
func (s *WebhookServer) AddWebhook(webhook WebhookNotification) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add timestamp if not provided
	if webhook.Timestamp == "" {
		webhook.Timestamp = fmt.Sprintf("%d", time.Now().Unix())
	}

	// Add to webhooks array
	s.webhooks = append([]WebhookNotification{webhook}, s.webhooks...)

	// Keep only the last 100 webhooks
	if len(s.webhooks) > 100 {
		s.webhooks = s.webhooks[:100]
	}

	log.Printf("Received webhook from %s: %s", webhook.Client, webhook.TxHash)
}

// GetWebhooks returns all webhooks
func (s *WebhookServer) GetWebhooks() []WebhookNotification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.webhooks
}

// ClearWebhooks clears all webhooks
func (s *WebhookServer) ClearWebhooks() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhooks = make([]WebhookNotification, 0)
	log.Println("All webhooks cleared")
}

// StartServer starts the webhook server
func (s *WebhookServer) StartServer() error {
	// Create a new HTTP server
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/webhooks", s.handleGetWebhooks)
	mux.HandleFunc("/clear", s.handleClearWebhooks)
	mux.HandleFunc("/health", s.handleHealth)

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	// Start server
	log.Printf("Webhook server starting on port %d", s.port)
	return server.ListenAndServe()
}

// handleWebhook handles incoming webhook requests
func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var webhook WebhookNotification
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Add client information if not provided
	if webhook.Client == "" {
		webhook.Client = r.Header.Get("X-Client")
		if webhook.Client == "" {
			webhook.Client = "unknown"
		}
	}

	// Add webhook to server
	s.AddWebhook(webhook)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Webhook received",
	})
}

// handleGetWebhooks handles requests to get all webhooks
func (s *WebhookServer) handleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get webhooks
	webhooks := s.GetWebhooks()

	// Return webhooks
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"webhooks": webhooks,
	})
}

// handleClearWebhooks handles requests to clear all webhooks
func (s *WebhookServer) handleClearWebhooks(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clear webhooks
	s.ClearWebhooks()

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "All webhooks cleared",
	})
}

// handleHealth handles health check requests
func (s *WebhookServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Return health status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

func main() {
	// Parse command line flags
	port := flag.Int("port", 3005, "Port to listen on")
	flag.Parse()

	// Create webhook server
	server := NewWebhookServer(*port)

	// Start server
	if err := server.StartServer(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

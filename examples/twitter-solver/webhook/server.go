package webhook

import (
	"encoding/json"
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

// WebhookServer represents a webhook server that can receive and store notifications
type WebhookServer struct {
	port             int
	webhooks         []WebhookNotification
	webhooksMutex    sync.RWMutex
	maxWebhooks      int
	subscribers      map[string][]chan WebhookNotification
	subscribersMutex sync.RWMutex
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(port int) *WebhookServer {
	return &WebhookServer{
		port:        port,
		webhooks:    make([]WebhookNotification, 0),
		maxWebhooks: 100,
		subscribers: make(map[string][]chan WebhookNotification),
	}
}

// Start starts the webhook server
func (s *WebhookServer) Start() error {
	// Set up routes
	http.HandleFunc("/webhook", s.handleWebhook)
	http.HandleFunc("/webhooks", s.handleGetWebhooks)
	http.HandleFunc("/subscribe", s.handleSubscribe)
	http.HandleFunc("/health", s.handleHealth)

	// Start the server
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Webhook server starting on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// handleWebhook handles incoming webhook notifications
func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the webhook notification
	var notification WebhookNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Add timestamp if not provided
	if notification.Timestamp == "" {
		notification.Timestamp = fmt.Sprintf("%d", time.Now().Unix())
	}

	// Add client information if available
	if client := r.URL.Query().Get("client"); client != "" {
		notification.Client = client
	}

	// Store the webhook notification
	s.addWebhook(notification)

	// Notify subscribers
	s.notifySubscribers(notification)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Webhook received",
	})
}

// handleGetWebhooks returns all stored webhook notifications
func (s *WebhookServer) handleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client filter if provided
	client := r.URL.Query().Get("client")

	// Get webhooks
	webhooks := s.getWebhooks(client)

	// Return webhooks
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"webhooks": webhooks,
		"count":    len(webhooks),
	})
}

// handleSubscribe sets up a server-sent events connection for real-time webhook notifications
func (s *WebhookServer) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client filter if provided
	client := r.URL.Query().Get("client")

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this subscriber
	notificationChan := make(chan WebhookNotification, 10)

	// Register the subscriber
	s.subscribersMutex.Lock()
	s.subscribers[client] = append(s.subscribers[client], notificationChan)
	s.subscribersMutex.Unlock()

	// Clean up when the connection closes
	defer func() {
		s.subscribersMutex.Lock()
		defer s.subscribersMutex.Unlock()

		// Remove this channel from subscribers
		if channels, ok := s.subscribers[client]; ok {
			for i, ch := range channels {
				if ch == notificationChan {
					s.subscribers[client] = append(channels[:i], channels[i+1:]...)
					break
				}
			}
		}

		close(notificationChan)
	}()

	// Keep the connection open and send notifications
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial message
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"message\":\"Connected to webhook server\"}\n\n")
	flusher.Flush()

	// Listen for notifications
	for {
		select {
		case notification := <-notificationChan:
			// Send the notification as SSE
			data, _ := json.Marshal(notification)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

// handleHealth returns a health check response
func (s *WebhookServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// addWebhook adds a webhook notification to the store
func (s *WebhookServer) addWebhook(notification WebhookNotification) {
	s.webhooksMutex.Lock()
	defer s.webhooksMutex.Unlock()

	// Add to the beginning of the array
	s.webhooks = append([]WebhookNotification{notification}, s.webhooks...)

	// Keep only the last maxWebhooks
	if len(s.webhooks) > s.maxWebhooks {
		s.webhooks = s.webhooks[:s.maxWebhooks]
	}

	log.Printf("Webhook added: %+v", notification)
}

// getWebhooks returns all webhook notifications, optionally filtered by client
func (s *WebhookServer) getWebhooks(client string) []WebhookNotification {
	s.webhooksMutex.RLock()
	defer s.webhooksMutex.RUnlock()

	if client == "" {
		// Return all webhooks
		return s.webhooks
	}

	// Filter by client
	filtered := make([]WebhookNotification, 0)
	for _, webhook := range s.webhooks {
		if webhook.Client == client {
			filtered = append(filtered, webhook)
		}
	}

	return filtered
}

// notifySubscribers notifies all subscribers of a new webhook notification
func (s *WebhookServer) notifySubscribers(notification WebhookNotification) {
	s.subscribersMutex.RLock()
	defer s.subscribersMutex.RUnlock()

	// Notify all subscribers for this client
	if channels, ok := s.subscribers[notification.Client]; ok {
		for _, ch := range channels {
			select {
			case ch <- notification:
				// Notification sent
			default:
				// Channel is full, skip this notification
			}
		}
	}

	// Also notify subscribers for all clients
	if channels, ok := s.subscribers[""]; ok {
		for _, ch := range channels {
			select {
			case ch <- notification:
				// Notification sent
			default:
				// Channel is full, skip this notification
			}
		}
	}
}

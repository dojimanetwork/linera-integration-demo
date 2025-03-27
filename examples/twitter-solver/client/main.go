package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/linera-protocol/examples/twitter-solver/client/solver"
)

var (
	twitterClient *solver.Client
	solverURL     string
	lineraPath    string
	// Store code verifiers temporarily (in a real app, use a proper session store)
	codeVerifiers = make(map[string]string)
	// Track processed states to prevent duplicate processing
	processedStates = make(map[string]bool)
	verifierMutex   sync.RWMutex
	logger          = solver.NewLogger()
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 10 * time.Second,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

func init() {
	initFlags()
	solver.InitLogger()
	if err := solver.InitLineraConfig(); err != nil {
		logger.Error("Failed to initialize Linera config: %v", err)
		os.Exit(1)
	}
	twitterClient = solver.NewClient()
}

func initFlags() {
	//flag.StringVar(&solverURL, "solver-url", getEnvOrDefault("TWITTER_SOLVER_URL", "http://localhost:8080"), "Twitter solver URL")
	//flag.StringVar(&lineraPath, "linera-path", getEnvOrDefault("LINERA_PATH", ""), "Path to Linera executable")
	//flag.Parse()
	//
	//if lineraPath == "" {
	//	logger.Error("Linera path must be provided")
	//	os.Exit(1)
	//}
	logger.Info("Twitter solver URL: %s", solverURL)
	logger.Info("Linera path: %s", lineraPath)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
		}
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)
		next.ServeHTTP(rw, r)
		logger.Info("%s %s %d %v", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

func handlePostTweet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req solver.PostTweetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := twitterClient.PostTweet(req.Content, req.Author); err != nil {
		http.Error(w, fmt.Sprintf("Failed to post tweet: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleDeleteTweet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req solver.DeleteTweetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := twitterClient.DeleteTweet(req.Index); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete tweet: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleGetTweets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tweets, err := twitterClient.GetTweets()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get tweets: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(solver.GetTweetsResponse{Tweets: tweets})
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			logger.Error("Failed to read message: %v", err)
			break
		}

		// Echo the message back for now
		if err := conn.WriteMessage(messageType, message); err != nil {
			logger.Error("Failed to write message: %v", err)
			break
		}
	}
}

func handleTwitterAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authURL, codeVerifier, state, err := twitterClient.GetAuthURL()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate auth URL: %v", err), http.StatusInternalServerError)
		return
	}

	// Store code verifier with state as key
	verifierMutex.Lock()
	codeVerifiers[state] = codeVerifier
	verifierMutex.Unlock()

	logger.Info("Stored code verifier for state: %s", state)

	// Clean up code verifier after 5 minutes
	go func() {
		time.Sleep(5 * time.Minute)
		verifierMutex.Lock()
		delete(codeVerifiers, state)
		verifierMutex.Unlock()
		logger.Info("Cleaned up code verifier for state: %s", state)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"auth_url":      authURL,
		"code_verifier": codeVerifier,
	})
}

func handleTwitterCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		logger.Error("Missing code or state in callback")
		http.Error(w, "Missing code or state", http.StatusBadRequest)
		return
	}

	logger.Info("Received callback with state: %s, code length: %d", state, len(code))

	// Check if we've already processed this state
	verifierMutex.RLock()
	alreadyProcessed := processedStates[state]
	verifierMutex.RUnlock()

	if alreadyProcessed {
		logger.Info("State %s already processed, returning success", state)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Already authenticated with Twitter",
			"user": map[string]string{
				"username": "Twitter User",
				"name":     "Twitter User",
			},
		})
		return
	}

	// Validate code format
	if len(code) < 10 {
		logger.Error("Invalid authorization code format: too short")
		http.Error(w, "Invalid authorization code format", http.StatusBadRequest)
		return
	}

	// Retrieve and validate code verifier
	verifierMutex.RLock()
	codeVerifier, exists := codeVerifiers[state]
	verifierMutex.RUnlock()

	if !exists {
		logger.Error("No code verifier found for state: %s", state)
		http.Error(w, "Invalid or expired state", http.StatusBadRequest)
		return
	}

	logger.Info("Found code verifier for state: %s", state)

	// Exchange code for token
	if err := twitterClient.ExchangeCodeForToken(code, codeVerifier); err != nil {
		logger.Error("Failed to exchange code for token: %v", err)
		http.Error(w, fmt.Sprintf("Failed to exchange code for token: %v", err), http.StatusInternalServerError)
		return
	}

	// Mark state as processed and clean up
	verifierMutex.Lock()
	processedStates[state] = true
	delete(codeVerifiers, state)
	verifierMutex.Unlock()

	logger.Info("Successfully exchanged code for token and cleaned up state: %s", state)

	// Return success response with user information
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Successfully authenticated with Twitter",
		"user": map[string]string{
			"username": "Twitter User",
			"name":     "Twitter User",
		},
	})
}

func handleLatestTweet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	latestTweet, err := twitterClient.GetLatestTweet()
	if err != nil {
		logger.Error("Failed to fetch latest tweet: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch latest tweet: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"tweet":  latestTweet,
	})
}

func handleTakeScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		TweetURL string `json:"tweet_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.TweetURL == "" {
		http.Error(w, "Tweet URL is required", http.StatusBadRequest)
		return
	}

	screenshotURL, err := twitterClient.TakeScreenshot(request.TweetURL)
	if err != nil {
		logger.Error("Failed to take screenshot: %v", err)
		http.Error(w, fmt.Sprintf("Failed to take screenshot: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":         "success",
		"screenshot_url": screenshotURL,
	})
}

func main() {
	http.HandleFunc("/post_tweet", handlePostTweet)
	http.HandleFunc("/delete_tweet", handleDeleteTweet)
	http.HandleFunc("/get_tweets", handleGetTweets)
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/twitter/auth", handleTwitterAuth)
	http.HandleFunc("/twitter/callback", handleTwitterCallback)
	http.HandleFunc("/latest_tweet", handleLatestTweet)
	http.HandleFunc("/take_screenshot", handleTakeScreenshot)

	// Serve screenshots directory
	http.Handle("/screenshots/", http.StripPrefix("/screenshots/", http.FileServer(http.Dir("screenshots"))))

	handler := corsMiddleware(loggingMiddleware(http.DefaultServeMux))
	logger.Info("Starting server on :8080")
	if err := http.ListenAndServe(":3005", handler); err != nil {
		logger.Error("Failed to start server: %v", err)
		os.Exit(1)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/linera-protocol/examples/twitter-solver/client/solver"
)

const twitterAPIURL = "https://api.twitter.com/2"

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

	// Create screenshots directory if it doesn't exist
	if err := os.MkdirAll("screenshots", 0755); err != nil {
		logger.Error("Failed to create screenshots directory: %v", err)
		os.Exit(1)
	}
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
		// Get the origin from the request
		origin := r.Header.Get("Origin")

		// Define allowed origins
		allowedOrigins := map[string]bool{
			"http://localhost:5173":           true,
			"https://market-place.ngrok.io":   true,
			"http://localhost:3002":           true,
			"https://twitter-solver.ngrok.io": true,
		}

		// Set CORS headers based on origin
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Expose-Headers", "Set-Cookie")
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
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
		time.Sleep(60 * time.Minute)
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

	// Get user information from Twitter
	userInfo, err := twitterClient.GetUserInfo()
	if err != nil {
		logger.Error("Failed to get user information: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get user information: %v", err), http.StatusInternalServerError)
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
			"id":       userInfo.ID,
			"username": userInfo.Username,
			"name":     userInfo.Name,
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

func handleTwitterProfileScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user info from Twitter API using the Twitter client
	userInfo, err := twitterClient.GetUserInfo()
	if err != nil {
		logger.Error("Failed to get user info from Twitter: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	if userInfo == nil {
		logger.Error("No tweets found for user")
		http.Error(w, "No tweets found for user", http.StatusInternalServerError)
		return
	}

	// Use the author ID from the latest tweet to get the username
	username := userInfo.Username

	// Construct Twitter profile URL
	profileURL := fmt.Sprintf("https://x.com/%s", username)

	// Urlbox API configuration
	urlboxAPIKey := os.Getenv("URLBOX_API_KEY")
	urlboxAPISecret := os.Getenv("URLBOX_API_SECRET")

	if urlboxAPIKey == "" || urlboxAPISecret == "" {
		logger.Error("Urlbox API credentials not configured")
		http.Error(w, "Screenshot service not configured", http.StatusInternalServerError)
		return
	}

	urlboxURL := "https://api.urlbox.io/v1/render/sync"

	// Create request body
	requestBody := map[string]interface{}{
		"url":       profileURL,
		"dark_mode": "true",
		"format":    "png",
		"delay":     "1000",
		"cookie":    []string{fmt.Sprintf("auth_token=%s", "ee09f38b122e71bcd40f1887f8fdf1d3bf0f022f")},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("Failed to marshal request body: %v", err)
		http.Error(w, "Failed to generate screenshot", http.StatusInternalServerError)
		return
	}

	// Create a new HTTP client
	client := &http.Client{}

	// Make request to Urlbox API
	req, err := http.NewRequest("POST", urlboxURL, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Failed to create request to Urlbox: %v", err)
		http.Error(w, "Failed to generate screenshot", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", urlboxAPISecret))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to get screenshot from Urlbox: %v", err)
		http.Error(w, "Failed to generate screenshot", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Urlbox API returned non-200 status: %d", resp.StatusCode)
		http.Error(w, "Failed to generate screenshot", http.StatusInternalServerError)
		return
	}

	var renderResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&renderResponse); err != nil {
		logger.Error("Failed to decode render response: %v", err)
		http.Error(w, "Failed to process render response", http.StatusInternalServerError)
		return
	}

	imageURL, ok := renderResponse["renderUrl"].(string)
	if !ok {
		logger.Error("Render URL not found in response")
		http.Error(w, "Failed to retrieve render URL", http.StatusInternalServerError)
		return
	}

	// Download image data from the render URL
	resp, err = http.Get(imageURL)
	if err != nil {
		logger.Error("Failed to download image from render URL: %v", err)
		http.Error(w, "Failed to download image", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Render URL returned non-200 status: %d", resp.StatusCode)
		http.Error(w, "Failed to download image", http.StatusInternalServerError)
		return
	}

	// Read the image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read screenshot data: %v", err)
		http.Error(w, "Failed to process screenshot", http.StatusInternalServerError)
		return
	}

	// Generate a unique filename
	filename := fmt.Sprintf("profile_%s_%d.png", username, time.Now().Unix())
	filepath := filepath.Join("screenshots", filename)

	// Save the screenshot
	if err := os.WriteFile(filepath, imageData, 0644); err != nil {
		logger.Error("Failed to save screenshot: %v", err)
		http.Error(w, "Failed to save screenshot", http.StatusInternalServerError)
		return
	}

	// Return the URL to access the screenshot
	screenshotURL := fmt.Sprintf("/screenshots/%s", filename)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":         "success",
		"screenshot_url": screenshotURL,
	})
}

func handleUserDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user details from Twitter
	userDetails, err := twitterClient.UserDetails()
	if err != nil {
		logger.Error("Failed to get user details: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get user details: %v", err), http.StatusInternalServerError)
		return
	}

	// Return user details as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   userDetails,
	})
}

func handleUserLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id parameter is required", http.StatusBadRequest)
		return
	}

	// Lookup user details
	userDetails, err := twitterClient.LookUpById(userID)
	if err != nil {
		logger.Error("Failed to lookup user: %v", err)
		http.Error(w, fmt.Sprintf("Failed to lookup user: %v", err), http.StatusInternalServerError)
		return
	}

	// Return user details
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   userDetails,
	})
}

type Session struct {
	UserID    string
	ExpiresAt time.Time
}

func getSession(sessionID string) (*Session, error) {
	// TODO: Implement Redis session retrieval
	// For now, return a mock session
	return &Session{
		UserID:    "mock_user_id",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

// customFileServer creates a file server with CORS headers
func customFileServer(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Set content type for images
		if strings.HasSuffix(r.URL.Path, ".png") {
			w.Header().Set("Content-Type", "image/png")
		} else if strings.HasSuffix(r.URL.Path, ".jpg") || strings.HasSuffix(r.URL.Path, ".jpeg") {
			w.Header().Set("Content-Type", "image/jpeg")
		}

		// Serve the file
		http.ServeFile(w, r, filepath.Join(dir, r.URL.Path))
	})
}

func main() {
	// Create screenshots directory if it doesn't exist
	if err := os.MkdirAll("screenshots", 0755); err != nil {
		logger.Error("Failed to create screenshots directory: %v", err)
		os.Exit(1)
	}

	// Create a new mux for routing
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/post_tweet", handlePostTweet)
	mux.HandleFunc("/delete_tweet", handleDeleteTweet)
	mux.HandleFunc("/get_tweets", handleGetTweets)
	mux.HandleFunc("/ws", handleWebSocket)
	mux.HandleFunc("/twitter/auth", handleTwitterAuth)
	mux.HandleFunc("/twitter/callback", handleTwitterCallback)
	mux.HandleFunc("/latest_tweet", handleLatestTweet)
	mux.HandleFunc("/take_screenshot", handleTakeScreenshot)
	mux.HandleFunc("/twitter_profile_screenshot", handleTwitterProfileScreenshot)
	mux.HandleFunc("/user_details", handleUserDetails)
	mux.HandleFunc("/user_lookup", handleUserLookup)

	// Serve screenshots directory with custom file server
	mux.Handle("/screenshots/", http.StripPrefix("/screenshots/", customFileServer("screenshots")))

	// Apply middleware to the mux
	handler := corsMiddleware(loggingMiddleware(mux))

	logger.Info("Starting server on :3005")
	if err := http.ListenAndServe(":3005", handler); err != nil {
		logger.Error("Failed to start server: %v", err)
		os.Exit(1)
	}
}

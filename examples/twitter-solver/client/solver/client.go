package solver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Client represents the Twitter solver client
type Client struct {
	httpClient    *http.Client
	logger        *Logger
	twitterClient *TwitterClient
}

// NewClient creates a new Twitter solver client
func NewClient() *Client {
	// Load Twitter configuration
	twitterConfig, err := LoadTwitterConfig()
	if err != nil {
		logger.Error("Failed to load Twitter config: %v", err)
		return &Client{
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
			logger: NewLogger(),
		}
	}

	// Initialize Twitter client
	twitterClient, err := NewTwitterClient(*twitterConfig)
	if err != nil {
		logger.Error("Failed to initialize Twitter client: %v", err)
		return &Client{
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
			logger: NewLogger(),
		}
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:        NewLogger(),
		twitterClient: twitterClient,
	}
}

// InitLogger initializes the logger
func InitLogger() {
	logger = NewLogger()
}

// InitLineraConfig initializes the Linera configuration
func InitLineraConfig() error {
	// Add any Linera-specific initialization here
	return nil
}

// GetAuthURL generates the authorization URL with PKCE
func (c *Client) GetAuthURL() (string, string, string, error) {
	if c.twitterClient == nil {
		return "", "", "", fmt.Errorf("Twitter client not initialized")
	}
	return c.twitterClient.GetAuthURL()
}

// ExchangeCodeForToken exchanges the authorization code for an access token
func (c *Client) ExchangeCodeForToken(code, codeVerifier string) error {
	if c.twitterClient == nil {
		return fmt.Errorf("Twitter client not initialized")
	}
	return c.twitterClient.ExchangeCodeForToken(code, codeVerifier)
}

// PostTweet sends a request to post a new tweet
func (c *Client) PostTweet(content string, author string) error {
	if c.twitterClient == nil {
		return fmt.Errorf("Twitter client not initialized")
	}
	return c.twitterClient.PostTweet(content)
}

// DeleteTweet sends a request to delete a tweet
func (c *Client) DeleteTweet(index int) error {
	// Implementation will be added later
	return nil
}

// GetTweets retrieves all tweets
func (c *Client) GetTweets() ([]Tweet, error) {
	// Implementation will be added later
	return nil, nil
}

func (c *Client) GetUserInfo() (*UserInfo, error) {
	info, err := c.twitterClient.GetAuthenticatedUserInfo()
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (c *Client) GetTcToken() (*oauth2.Token, error) {
	return c.twitterClient.GetTcToken()
}

func (c *Client) UserDetails() (*UserDetails, error) {
	return c.twitterClient.GetUserDetails()
}

// GetLatestTweet fetches the latest tweet from Twitter with retry logic
func (c *Client) GetLatestTweet() (*TwitterTweet, error) {
	if c.twitterClient == nil {
		return nil, fmt.Errorf("Twitter client not initialized")
	}

	maxRetries := 3
	baseDelay := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		latestTweet, err := c.twitterClient.GetLatestTweet()
		if err == nil {
			return latestTweet, nil
		}

		// Check if it's a rate limit error
		if strings.Contains(err.Error(), "429") {
			delay := baseDelay * time.Duration(attempt)
			c.logger.Info("Rate limit hit, retrying in %v (attempt %d/%d)", delay, attempt, maxRetries)
			time.Sleep(delay)
			continue
		}

		// For other errors, return immediately
		if strings.Contains(err.Error(), "no tweets found") {
			return nil, fmt.Errorf("no tweets found")
		}
		return nil, fmt.Errorf("failed to get latest tweet: %v", err)
	}

	return nil, fmt.Errorf("max retries exceeded for getting latest tweet")
}

// TakeScreenshot takes a screenshot of a tweet URL using TweetPik API
func (c *Client) TakeScreenshot(tweetURL string) (string, error) {
	if c.twitterClient == nil {
		return "", fmt.Errorf("Twitter client not initialized")
	}

	// Extract tweet ID from URL
	tweetID := extractTweetID(tweetURL)
	if tweetID == "" {
		return "", fmt.Errorf("invalid tweet URL")
	}

	// Create screenshots directory if it doesn't exist
	if err := os.MkdirAll("screenshots", 0755); err != nil {
		return "", fmt.Errorf("failed to create screenshots directory: %v", err)
	}

	// Check if screenshot already exists
	screenshotPath := fmt.Sprintf("screenshots/%s.png", tweetID)
	if _, err := os.Stat(screenshotPath); err == nil {
		// Screenshot exists, return the URL
		c.logger.Info("Screenshot already exists: %s", screenshotPath)
		return fmt.Sprintf("/screenshots/%s.png", tweetID), nil
	}

	// Create request body
	reqBody := map[string]interface{}{
		"url":             tweetURL,
		"dimension":       "instagramFeed",
		"displayMetrics":  true,
		"displayVerified": "auto",
		"displayEmbeds":   true,
		"contentWidth":    90,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", "https://tweetpik.com/api/v2/images", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "563c27cc-f658-4b62-bd7b-874b20f8ff18")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("TweetPik API error: %s", string(body))
	}

	// Parse response
	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	// Download the image
	imgResp, err := http.Get(result.URL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %v", err)
	}
	defer imgResp.Body.Close()

	// Save the image
	file, err := os.Create(screenshotPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, imgResp.Body); err != nil {
		return "", fmt.Errorf("failed to save image: %v", err)
	}

	// Log success and show screenshot URL
	screenshotURL := fmt.Sprintf("/screenshots/%s.png", tweetID)
	c.logger.Info("Screenshot created successfully: %s", screenshotURL)

	// Return the URL to access the screenshot
	return screenshotURL, nil
}

// extractTweetID extracts the tweet ID from a Twitter URL
func extractTweetID(url string) string {
	// Handle different Twitter URL formats
	patterns := []string{
		`twitter\.com/\w+/status/(\d+)`,
		`x\.com/\w+/status/(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// Tweet represents a tweet in the system
type Tweet struct {
	Content   string `json:"content"`
	Author    string `json:"author"`
	Timestamp int64  `json:"timestamp"`
}

// TwitterTweet represents a tweet from Twitter's API
type TwitterTweet struct {
	ID               string   `json:"id"`
	Text             string   `json:"text"`
	CreatedAt        string   `json:"created_at"`
	AuthorID         string   `json:"author_id"`
	DisplayTextRange []int    `json:"display_text_range"`
	EditHistoryIDs   []string `json:"edit_history_tweet_ids"`
	Entities         struct {
		Mentions []struct {
			Start    int    `json:"start"`
			End      int    `json:"end"`
			Username string `json:"username"`
			ID       string `json:"id"`
		} `json:"mentions"`
		Hashtags []struct {
			Start int    `json:"start"`
			End   int    `json:"end"`
			Tag   string `json:"tag"`
		} `json:"hashtags"`
		Annotations []struct {
			Start          int     `json:"start"`
			End            int     `json:"end"`
			Probability    float64 `json:"probability"`
			Type           string  `json:"type"`
			NormalizedText string  `json:"normalized_text"`
		} `json:"annotations"`
	} `json:"entities"`
}

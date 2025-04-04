package solver

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"golang.org/x/oauth2"
)

const (
	twitterAuthURL  = "https://twitter.com/i/oauth2/authorize"
	twitterTokenURL = "https://api.twitter.com/2/oauth2/token"
	twitterAPIURL   = "https://api.twitter.com/2"
)

// TwitterConfig holds Twitter OAuth credentials
type TwitterConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	// OAuth 1.0a credentials
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
}

// TwitterClient handles Twitter API interactions
type TwitterClient struct {
	config *oauth2.Config
	token  *oauth2.Token
	// OAuth 1.0a client for tweet operations
	oauth1Client *http.Client
}

// NewTwitterClient creates a new Twitter client with the given credentials
func NewTwitterClient(config TwitterConfig) (*TwitterClient, error) {
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURI,
		Scopes:       config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  twitterAuthURL,
			TokenURL: twitterTokenURL,
		},
	}

	// Initialize OAuth 1.0a client for tweet operations
	oauth1Config := oauth1.NewConfig(config.ConsumerKey, config.ConsumerSecret)
	oauth1Token := oauth1.NewToken(config.AccessToken, config.AccessSecret)
	oauth1Client := oauth1Config.Client(oauth1.NoContext, oauth1Token)

	return &TwitterClient{
		config:       oauthConfig,
		oauth1Client: oauth1Client,
	}, nil
}

// GeneratePKCE generates a PKCE code verifier and challenge
func GeneratePKCE() (string, string, error) {
	// Generate random code verifier
	verifier := make([]byte, 32)
	if _, err := rand.Read(verifier); err != nil {
		return "", "", fmt.Errorf("failed to generate code verifier: %v", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifier)

	// Generate code challenge
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return codeVerifier, codeChallenge, nil
}

// GetAuthURL generates the authorization URL with PKCE
func (tc *TwitterClient) GetAuthURL() (string, string, string, error) {
	codeVerifier, codeChallenge, err := GeneratePKCE()
	if err != nil {
		return "", "", "", err
	}

	// Generate a random state
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate state: %v", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	logger.Debug("Generated PKCE - Verifier: %s, Challenge: %s, State: %s", codeVerifier, codeChallenge, state)

	authURL := tc.config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	return authURL, codeVerifier, state, nil
}

// ExchangeCodeForToken exchanges the authorization code for an access token
func (tc *TwitterClient) ExchangeCodeForToken(code, codeVerifier string) error {
	logger.Info("Starting token exchange with code length: %d", len(code))

	// Validate code format
	if len(code) < 10 {
		logger.Error("Invalid authorization code format: too short")
		return fmt.Errorf("invalid authorization code format")
	}

	// Create a custom HTTP client that includes the client credentials in the token exchange
	client := &http.Client{}

	// Prepare the token exchange request
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("code_verifier", codeVerifier)
	data.Set("redirect_uri", tc.config.RedirectURL)

	logger.Debug("Token exchange request data: %v", data)

	req, err := http.NewRequest("POST", twitterTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		logger.Error("Failed to create token exchange request: %v", err)
		return fmt.Errorf("failed to create token exchange request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(tc.config.ClientID+":"+tc.config.ClientSecret)))

	logger.Debug("Token exchange request headers: %v", req.Header)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to exchange code for token: %v", err)
		return fmt.Errorf("failed to exchange code for token: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read token response: %v", err)
		return fmt.Errorf("failed to read token response: %v", err)
	}

	logger.Debug("Token exchange response status: %d, body: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errorResp); err != nil {
			logger.Error("Failed to parse error response: %v", err)
			return fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
		}
		logger.Error("Token exchange failed: %s - %s", errorResp.Error, errorResp.ErrorDescription)
		return fmt.Errorf("token exchange failed: %s - %s", errorResp.Error, errorResp.ErrorDescription)
	}

	// Parse the token response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token,omitempty"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		logger.Error("Failed to parse token response: %v", err)
		return fmt.Errorf("failed to parse token response: %v", err)
	}

	logger.Info("Successfully exchanged code for token. Expires in: %d seconds", tokenResp.ExpiresIn)

	// Create and store the token
	tc.token = &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return nil
}

// PostTweet posts a tweet to Twitter
func (tc *TwitterClient) PostTweet(content string) error {
	if tc.token == nil {
		return fmt.Errorf("not authenticated")
	}

	client := tc.config.Client(oauth2.NoContext, tc.token)

	// Prepare tweet data
	tweetData := map[string]interface{}{
		"text": content,
	}

	jsonData, err := json.Marshal(tweetData)
	if err != nil {
		return fmt.Errorf("failed to marshal tweet data: %v", err)
	}

	// Send tweet
	resp, err := client.Post(twitterAPIURL+"/tweets", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to post tweet: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to post tweet: status %d", resp.StatusCode)
	}

	return nil
}

// GetAuthenticatedUserInfo retrieves the authenticated user's information from Twitter
func (tc *TwitterClient) GetAuthenticatedUserInfo() (*UserInfo, error) {
	client := tc.config.Client(oauth2.NoContext, tc.token)
	userResp, err := client.Get(twitterAPIURL + "/users/me")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer userResp.Body.Close()

	var userData struct {
		Data UserInfo `json:"data"`
	}

	if err := json.NewDecoder(userResp.Body).Decode(&userData); err != nil {
		return nil, fmt.Errorf("failed to parse user data: %v", err)
	}

	return &userData.Data, nil
}

func (tc *TwitterClient) GetTcToken() (*oauth2.Token, error) {
	return tc.token, nil
}

// GetLatestTweet retrieves the latest tweet from the authenticated user
func (tc *TwitterClient) GetLatestTweet() (*TwitterTweet, error) {
	// First, get the authenticated user's ID
	userInfo, err := tc.GetAuthenticatedUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}

	tweetsURL := fmt.Sprintf("%s/users/%s/tweets?tweet.fields=author_id,created_at,display_text_range,edit_history_tweet_ids,entities,id,text,source&user.fields=id,confirmed_email", twitterAPIURL, userInfo.ID)
	req, err := http.NewRequest("GET", tweetsURL, nil)

	// Add required headers
	req.Header.Add("Authorization", "Bearer AAAAAAAAAAAAAAAAAAAAAEiK0AEAAAAAYsG8yyn7gdNdMDIq445ek%2FnXypY%3Djp5rtuWxopwQlhpSB2cNlaxyemQMishgqEAXEiDpWI5AVAN3Ps")
	req.Header.Add("Content-Type", "application/json")

	// Use a new HTTP client for this request
	httpClient := &http.Client{}
	tweetsResp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tweets: %v", err)
	}
	var tweetsData struct {
		Data []TwitterTweet `json:"data"`
	}

	if err := json.NewDecoder(tweetsResp.Body).Decode(&tweetsData); err != nil {
		return nil, fmt.Errorf("failed to parse tweets data: %v", err)
	}

	if len(tweetsData.Data) == 0 {
		return nil, fmt.Errorf("no tweets found")
	}

	return &tweetsData.Data[0], nil
}

// GetTweetByID fetches a tweet by its ID using OAuth 1.0a
func (tc *TwitterClient) GetTweetByID(tweetID string) (*TwitterTweet, error) {
	if tc.oauth1Client == nil {
		return nil, fmt.Errorf("OAuth 1.0a client not initialized")
	}

	url := fmt.Sprintf("%s/tweets/%s?tweet.fields=created_at,entities,author_id,text,display_text_range,edit_history_tweet_ids", twitterAPIURL, tweetID)

	resp, err := tc.oauth1Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received %d status code, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data TwitterTweet `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result.Data, nil
}

// LoadTwitterConfig loads Twitter credentials from environment variables
func LoadTwitterConfig() (*TwitterConfig, error) {
	config := &TwitterConfig{
		ClientID:     os.Getenv("TWITTER_CLIENT_ID"),
		ClientSecret: os.Getenv("TWITTER_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("TWITTER_REDIRECT_URI"),
		Scopes:       []string{"tweet.read", "tweet.write", "users.read"},
		// OAuth 1.0a credentials
		ConsumerKey:    os.Getenv("TWITTER_CONSUMER_KEY"),
		ConsumerSecret: os.Getenv("TWITTER_CONSUMER_SECRET"),
		AccessToken:    os.Getenv("TWITTER_ACCESS_TOKEN"),
		AccessSecret:   os.Getenv("TWITTER_ACCESS_SECRET"),
	}

	// Validate required credentials
	if config.ClientID == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("Twitter client credentials are required")
	}
	if config.RedirectURI == "" {
		return nil, fmt.Errorf("Twitter redirect URI is required")
	}
	if config.ConsumerKey == "" || config.ConsumerSecret == "" {
		return nil, fmt.Errorf("Twitter OAuth 1.0a consumer credentials are required")
	}
	if config.AccessToken == "" || config.AccessSecret == "" {
		return nil, fmt.Errorf("Twitter OAuth 1.0a access credentials are required")
	}

	return config, nil
}

// UserDetails represents detailed user information from Twitter
type UserDetails struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	BannerURL       string `json:"profile_banner_url"`
	FollowersCount  int    `json:"followers_count"`
	FollowingCount  int    `json:"following_count"`
	TweetCount      int    `json:"tweet_count"`
	CreatedAt       string `json:"created_at"`
	Verified        bool   `json:"verified"`
}

// GetUserDetails fetches detailed user information from Twitter
func (tc *TwitterClient) GetUserDetails() (*UserDetails, error) {
	// First get the user ID using GetAuthenticatedUserInfo
	userInfo, err := tc.GetAuthenticatedUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Construct the URL for user lookup
	url := fmt.Sprintf("%s/users/%s?user.fields=created_at,description,profile_image_url,profile_banner_url,public_metrics,verified", twitterAPIURL, userInfo.ID)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tc.token.AccessToken))

	// Make request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("twitter API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Data struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			Username        string `json:"username"`
			Description     string `json:"description"`
			ProfileImageURL string `json:"profile_image_url"`
			BannerURL       string `json:"profile_banner_url"`
			CreatedAt       string `json:"created_at"`
			Verified        bool   `json:"verified"`
			PublicMetrics   struct {
				FollowersCount int `json:"followers_count"`
				FollowingCount int `json:"following_count"`
				TweetCount     int `json:"tweet_count"`
			} `json:"public_metrics"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to UserDetails
	details := &UserDetails{
		ID:              result.Data.ID,
		Name:            result.Data.Name,
		Username:        result.Data.Username,
		Description:     result.Data.Description,
		ProfileImageURL: result.Data.ProfileImageURL,
		BannerURL:       result.Data.BannerURL,
		FollowersCount:  result.Data.PublicMetrics.FollowersCount,
		FollowingCount:  result.Data.PublicMetrics.FollowingCount,
		TweetCount:      result.Data.PublicMetrics.TweetCount,
		CreatedAt:       result.Data.CreatedAt,
		Verified:        result.Data.Verified,
	}

	return details, nil
}

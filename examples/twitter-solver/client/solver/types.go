package solver

import (
	"encoding/json"
)

// Request represents a generic request structure
type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Response represents a generic response structure
type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Error represents an error response
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// PostTweetRequest represents a request to post a new tweet
type PostTweetRequest struct {
	Content string `json:"content"`
	Author  string `json:"author"`
}

// DeleteTweetRequest represents a request to delete a tweet
type DeleteTweetRequest struct {
	Index int `json:"index"`
}

// GetTweetsResponse represents a response containing tweets
type GetTweetsResponse struct {
	Tweets []Tweet `json:"tweets"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// UserDetails represents the response from Twitter's user lookup endpoint
type LookUpUserDetails struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	CreatedAt       string `json:"created_at"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	Protected       bool   `json:"protected"`
	Verified        bool   `json:"verified"`
	VerifiedType    string `json:"verified_type"`
	Location        string `json:"location"`
	URL             string `json:"url"`
	PublicMetrics   struct {
		FollowersCount int `json:"followers_count"`
		FollowingCount int `json:"following_count"`
		TweetCount     int `json:"tweet_count"`
		ListedCount    int `json:"listed_count"`
	} `json:"public_metrics"`
}

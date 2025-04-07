package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

// WebhookClient represents a client for sending webhooks
type WebhookClient struct {
	serverURL string
	client    *http.Client
	clientID  string
}

// NewWebhookClient creates a new webhook client
func NewWebhookClient(serverURL, clientID string) *WebhookClient {
	return &WebhookClient{
		serverURL: serverURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		clientID: clientID,
	}
}

// SendWebhook sends a webhook to the server
func (c *WebhookClient) SendWebhook(webhook WebhookNotification) error {
	// Add client information if not provided
	if webhook.Client == "" {
		webhook.Client = c.clientID
	}

	// Add timestamp if not provided
	if webhook.Timestamp == "" {
		webhook.Timestamp = fmt.Sprintf("%d", time.Now().Unix())
	}

	// Marshal webhook to JSON
	jsonData, err := json.Marshal(webhook)
	if err != nil {
		return fmt.Errorf("error marshaling webhook: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", c.serverURL+"/webhook", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client", c.clientID)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetWebhooks gets all webhooks from the server
func (c *WebhookClient) GetWebhooks() ([]WebhookNotification, error) {
	// Create request
	req, err := http.NewRequest("GET", c.serverURL+"/webhooks", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("X-Client", c.clientID)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get webhooks request failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		Status   string                `json:"status"`
		Webhooks []WebhookNotification `json:"webhooks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return result.Webhooks, nil
}

// ClearWebhooks clears all webhooks from the server
func (c *WebhookClient) ClearWebhooks() error {
	// Create request
	req, err := http.NewRequest("POST", c.serverURL+"/clear", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("X-Client", c.clientID)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clear webhooks request failed with status: %d", resp.StatusCode)
	}

	return nil
}

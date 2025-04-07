package main

import (
	"fmt"
	"log"
	"time"

	"github.com/linera-protocol/webhook-server/client"
)

func main() {
	// Create a new webhook client
	webhookClient := client.NewWebhookClient("http://localhost:3005", "example-client")

	// Example 1: Send a successful transaction webhook
	successNotification := client.WebhookNotification{
		Status:      "success",
		TxHash:      "0x1234567890abcdef",
		Chain:       "ethereum",
		FromAddress: "0xabcdef1234567890",
		FromToken:   "ETH",
		Amount:      "1.5",
		Data: map[string]interface{}{
			"transaction_type": "transfer",
			"to_address":       "0x0987654321fedcba",
		},
	}

	if err := webhookClient.SendWebhook(successNotification); err != nil {
		log.Printf("Error sending success webhook: %v", err)
	} else {
		fmt.Println("Successfully sent success webhook")
	}

	// Wait a bit before sending the next webhook
	time.Sleep(time.Second)

	// Example 2: Send a failed transaction webhook
	failedNotification := client.WebhookNotification{
		Status:      "failed",
		TxHash:      "0xfedcba0987654321",
		Chain:       "ethereum",
		FromAddress: "0xabcdef1234567890",
		FromToken:   "ETH",
		Amount:      "0.5",
		Data: map[string]interface{}{
			"transaction_type": "swap",
			"error":            "insufficient funds",
		},
	}

	if err := webhookClient.SendWebhook(failedNotification); err != nil {
		log.Printf("Error sending failed webhook: %v", err)
	} else {
		fmt.Println("Successfully sent failed webhook")
	}

	// Wait a bit before retrieving webhooks
	time.Sleep(time.Second)

	// Example 3: Retrieve all webhooks
	webhooks, err := webhookClient.GetWebhooks()
	if err != nil {
		log.Printf("Error getting webhooks: %v", err)
	} else {
		fmt.Println("\nRetrieved webhooks:")
		for i, webhook := range webhooks {
			fmt.Printf("\nWebhook #%d:\n", i+1)
			fmt.Printf("  Status:      %s\n", webhook.Status)
			fmt.Printf("  TxHash:      %s\n", webhook.TxHash)
			fmt.Printf("  Chain:       %s\n", webhook.Chain)
			fmt.Printf("  FromAddress: %s\n", webhook.FromAddress)
			fmt.Printf("  FromToken:   %s\n", webhook.FromToken)
			fmt.Printf("  Amount:      %s\n", webhook.Amount)
			fmt.Printf("  Timestamp:   %s\n", webhook.Timestamp)
			fmt.Printf("  Client:      %s\n", webhook.Client)
			fmt.Printf("  Data:        %v\n", webhook.Data)
		}
	}

	// Example 4: Clear all webhooks
	if err := webhookClient.ClearWebhooks(); err != nil {
		log.Printf("Error clearing webhooks: %v", err)
	} else {
		fmt.Println("\nSuccessfully cleared all webhooks")
	}

	// Verify webhooks are cleared
	webhooks, err = webhookClient.GetWebhooks()
	if err != nil {
		log.Printf("Error getting webhooks after clear: %v", err)
	} else if len(webhooks) == 0 {
		fmt.Println("Verified: All webhooks have been cleared")
	} else {
		fmt.Printf("Warning: %d webhooks still exist after clear\n", len(webhooks))
	}
}

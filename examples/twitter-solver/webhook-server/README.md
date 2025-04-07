# Webhook Server

A common webhook server for handling notifications from various clients in the Linera Protocol project.

## Features

- Centralized webhook handling for multiple clients
- Client identification and filtering
- Webhook storage and retrieval
- Simple REST API
- Go client library for easy integration

## Server Setup

1. Install Go 1.21 or later
2. Clone the repository
3. Navigate to the webhook-server directory
4. Run the server:

```bash
go run main.go
```

The server will start on port 3005 by default. You can change the port by setting the `PORT` environment variable.

## API Endpoints

- `POST /webhook` - Send a webhook notification
- `GET /webhooks` - Get all webhooks
- `POST /clear` - Clear all webhooks
- `GET /health` - Health check endpoint

## Client Library Usage

### Installation

```bash
go get github.com/linera-protocol/webhook-server/client
```

### Example Usage

```go
package main

import (
    "fmt"
    "log"
    "github.com/linera-protocol/webhook-server/client"
)

func main() {
    // Create a new webhook client
    webhookClient := client.NewWebhookClient("http://localhost:3005", "my-client")

    // Create a webhook notification
    notification := client.WebhookNotification{
        Status:      "success",
        TxHash:      "0x123...",
        Chain:       "ethereum",
        FromAddress: "0xabc...",
        FromToken:   "ETH",
        Amount:      "1.5",
        Data: map[string]interface{}{
            "custom_field": "value",
        },
    }

    // Send the webhook
    if err := webhookClient.SendWebhook(notification); err != nil {
        log.Fatalf("Error sending webhook: %v", err)
    }

    // Get all webhooks
    webhooks, err := webhookClient.GetWebhooks()
    if err != nil {
        log.Fatalf("Error getting webhooks: %v", err)
    }

    // Print webhooks
    for _, webhook := range webhooks {
        fmt.Printf("Webhook: %+v\n", webhook)
    }

    // Clear all webhooks
    if err := webhookClient.ClearWebhooks(); err != nil {
        log.Fatalf("Error clearing webhooks: %v", err)
    }
}
```

## Webhook Notification Format

```json
{
    "status": "success",
    "txHash": "0x123...",
    "chain": "ethereum",
    "fromAddress": "0xabc...",
    "fromToken": "ETH",
    "amount": "1.5",
    "timestamp": "1234567890",
    "data": {
        "custom_field": "value"
    },
    "client": "my-client"
}
```

## Client Identification

Each client should provide a unique client ID when sending webhooks. This ID is used to identify the source of the webhook and can be used for filtering and organization.

The client ID can be provided in two ways:
1. In the webhook notification's `client` field
2. In the `X-Client` HTTP header

## Error Handling

The client library provides detailed error messages for various failure scenarios:
- Network errors
- Invalid JSON
- Server errors
- Invalid responses

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details. 
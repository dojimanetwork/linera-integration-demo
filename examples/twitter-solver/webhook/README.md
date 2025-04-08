# Linera Protocol Webhook Server

A centralized webhook server for handling notifications across multiple Linera Protocol clients, including the crowd-funding app and Twitter solver frontend.

## Features

- Real-time notifications via Server-Sent Events (SSE)
- CORS support for cross-origin requests
- Client-specific webhook filtering
- Health check endpoint
- Static file serving for screenshots and other assets
- In-memory webhook storage with automatic cleanup

## API Endpoints

### POST /webhook
Receives webhook notifications from clients.

Query Parameters:
- `client` (optional): Identifier for the client sending the webhook (e.g., 'crowd-funding', 'twitter-solver')

Request Body:
```json
{
  "status": "success",
  "txHash": "0x123...",
  "chain": "ethereum",
  "fromAddress": "0xabc...",
  "fromToken": "ETH",
  "amount": "1.5",
  "timestamp": "2024-03-21T12:00:00Z",
  "data": {
    "additional": "information"
  },
  "client": "crowd-funding"
}
```

### GET /webhooks
Retrieves stored webhook notifications.

Query Parameters:
- `client` (optional): Filter webhooks by client identifier

### GET /subscribe
Establishes a Server-Sent Events connection for real-time notifications.

### GET /health
Health check endpoint.

### GET /screenshots/*
Serves static files from the screenshots directory.

## Usage

### Starting the Server

```bash
cd examples/twitter-solver/webhook
go run main.go
```

The server will start on port 3005 by default.

### Sending Webhooks

```bash
curl -X POST http://localhost:3005/webhook?client=crowd-funding \
  -H "Content-Type: application/json" \
  -d '{
    "status": "success",
    "txHash": "0x123...",
    "chain": "ethereum",
    "fromAddress": "0xabc...",
    "fromToken": "ETH",
    "amount": "1.5",
    "timestamp": "2024-03-21T12:00:00Z",
    "data": {},
    "client": "crowd-funding"
  }'
```

### Retrieving Webhooks

```bash
# Get all webhooks
curl http://localhost:3005/webhooks

# Get webhooks for a specific client
curl http://localhost:3005/webhooks?client=crowd-funding
```

### Subscribing to Notifications

```javascript
const eventSource = new EventSource('http://localhost:3005/subscribe');
eventSource.onmessage = (event) => {
  const webhook = JSON.parse(event.data);
  console.log('New webhook:', webhook);
};
```

## Integration

### Crowd-Funding Client

The crowd-funding client sends webhook notifications when transactions are processed:

```go
webhookURL := "http://localhost:3005/webhook?client=crowd-funding"
notification := WebhookNotification{
    Status:      "success",
    TxHash:      txHash,
    Chain:       "ethereum",
    FromAddress: fromAddress,
    FromToken:   "ETH",
    Amount:      amount,
    Timestamp:   time.Now().UTC(),
    Data:        map[string]interface{}{},
    Client:      "crowd-funding",
}
```

### Twitter Frontend

The Twitter frontend can:
1. Send webhook notifications for transactions
2. Subscribe to real-time notifications
3. Display webhook history
4. View transaction screenshots

## Development

### Dependencies

- Go 1.21 or later
- github.com/gorilla/mux v1.8.1

### Building

```bash
go build -o webhook-server
```

### Testing

```bash
go test ./...
```

## License

This project is part of the Linera Protocol and is subject to its licensing terms. 
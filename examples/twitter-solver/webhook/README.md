# Webhook Server

This is a centralized webhook server for handling notifications from various clients in the Twitter Solver project.

## Features

- Receives webhook notifications from multiple clients (universal-solver, crowd-funding, non-fungible)
- Stores webhook notifications in memory with a configurable limit
- Provides REST API endpoints for retrieving webhook notifications
- Supports Server-Sent Events (SSE) for real-time notifications
- Includes health check endpoint
- Supports filtering notifications by client

## API Endpoints

### POST /webhook
Receives webhook notifications from clients.

Request body:
```json
{
  "status": "success",
  "txHash": "0x...",
  "chain": "ethereum",
  "fromAddress": "0x...",
  "fromToken": "ETH",
  "amount": "1.5",
  "timestamp": "2024-03-21T12:00:00Z",
  "data": {
    "additional": "data"
  },
  "client": "universal-solver"
}
```

### GET /webhooks
Retrieves stored webhook notifications.

Query parameters:
- `client` (optional): Filter notifications by client
- `limit` (optional): Limit the number of notifications returned

### GET /webhooks/subscribe
Subscribes to real-time webhook notifications using Server-Sent Events.

Query parameters:
- `client` (optional): Filter notifications by client

### GET /health
Health check endpoint.

## Usage

1. Start the server:
```bash
go run main.go -port 3005
```

2. Send a webhook notification:
```bash
curl -X POST http://localhost:3005/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "status": "success",
    "txHash": "0x...",
    "chain": "ethereum",
    "fromAddress": "0x...",
    "fromToken": "ETH",
    "amount": "1.5",
    "timestamp": "2024-03-21T12:00:00Z",
    "data": {},
    "client": "universal-solver"
  }'
```

3. Retrieve webhook notifications:
```bash
curl http://localhost:3005/webhooks?client=universal-solver
```

4. Subscribe to real-time notifications:
```bash
curl http://localhost:3005/webhooks/subscribe?client=universal-solver
```

## Integration with Clients

### Universal Solver
Update the webhook URL in the configuration to point to this server:
```
http://localhost:3005/webhook
```

### Crowd Funding
Update the webhook URL in the configuration to point to this server:
```
http://localhost:3005/webhook
```

### Non-Fungible Client
Update the webhook URL in the configuration to point to this server:
```
http://localhost:3005/webhook
```

## Development

### Building
```bash
go build -o webhook-server
```

### Running Tests
```bash
go test ./...
```

### Docker
```bash
docker build -t webhook-server .
docker run -p 3005:3005 webhook-server
``` 
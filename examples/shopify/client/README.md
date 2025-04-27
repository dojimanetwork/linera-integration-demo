# Shopify Go HTTP Client

This is a Go-based HTTP client server for the Shopify application in the Linera Protocol. It provides an interface between web applications and the Linera blockchain for Shopify operations.

## Features

- RESTful API for product management and order processing
- WebSocket support for real-time updates
- Webhook notifications for event handling
- CORS support for web applications

## Prerequisites

- Go 1.21 or later
- Connection to a running Linera node
- Shopify application deployed on Linera

## Installation

1. Clone the repository:
```bash
git clone https://github.com/linera-protocol/examples.git
cd examples/shopify/client
```

2. Install dependencies:
```bash
go mod tidy
```

## Usage

### Starting the server

```bash
go run main.go [options]
```

Available options:
- `-shopify-url string`: Shopify service URL (default: "http://localhost:8081/")
- `-linera-url string`: Linera service URL (default: "http://localhost:8080/")
- `-webhook-url string`: Webhook URL for notifications
- `-port string`: Server port (default: "3000")

You can also set these options using environment variables:
- `SHOPIFY_URL`: Shopify service URL
- `LINERA_URL`: Linera service URL
- `WEBHOOK_URL`: Webhook URL for notifications
- `PORT`: Server port

### API Endpoints

#### Product Management
- `GET /products`: List all products
- `GET /product/{id}`: Get product details by ID

#### Order Processing
- `POST /create_order`: Create a new order
  - Body: JSON with orderID, productID, and amount
  - Query param: webhookURL (optional)

#### WebSocket
- `GET /ws`: WebSocket endpoint for real-time updates

## Example Requests

### Creating an Order

```bash
curl -X POST http://localhost:3000/create_order \
  -H "Content-Type: application/json" \
  -d '{"orderId":"order123","productId":"prod_001","amount":19.99}'
```

### Getting Products

```bash
curl -X GET http://localhost:3000/products
```

### Getting Product Details

```bash
curl -X GET http://localhost:3000/product/prod_001
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 
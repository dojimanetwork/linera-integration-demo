# Universal Solver

A Linera application for managing files and pool addresses.

## GraphQL API Reference

### Queries

#### Get File by ID
```graphql
query {
  getFileSolverApp(id: "file_id_here") {
    solverFileId
    owner
    name
    payload
  }
}
```

#### Get Files by Owner
```graphql
query {
  getFilesByOwner(owner: "User:21cf9e655850b761c4577f6d1324b2956c7a3d39f3733ca0e05c4aad381da4a5") {
    solverFileId
    owner
    name
    payload
  }
}
```

#### Get Pool Address
```graphql
query {
  getPoolAddress(chainName: "ethereum") {
    address
  }
}
```

#### Get All Pool Addresses
```graphql
query {
  getAllPoolAddresses {
    chainName
    address
  }
}
```

#### JSON Utilities
```graphql
# Convert JSON to bytes
query {
  jsonToBytes(jsonStr: "{\"key\": \"value\"}")
}

# Convert bytes to JSON
query {
  bytesToJson(bytes: [123, 34, 107, 101, 121, 34, 58, 32, 34, 118, 97, 108, 117, 101, 34, 125])
}
```

### Mutations

#### Add File
```graphql
mutation {
  addFile(
    owner: "User:21cf9e655850b761c4577f6d1324b2956c7a3d39f3733ca0e05c4aad381da4a5"
    name: "example.json"
    blobHash: "hash_here"
  )
}
```

#### Add Pool Address
```graphql
mutation {
  addPoolAddress(
    chainName: "ethereum"
    address: "0x123..."
  )
}
```

#### Remove Pool Address
```graphql
mutation {
  removePoolAddress(
    chainName: "ethereum"
  )
}
```

## HTTP API Reference

The service also provides a REST API for transaction-related operations.

### Endpoints

#### POST /post_tx_hash
Get transaction details by hash.

Request:
```bash
curl -X POST "http://localhost:3000/post_tx_hash?txHash=0x123..."
```

Response:
```json
{
  "status": "success",
  "data": {
    "hash": "0x123...",
    "blockHash": "0x456...",
    "blockNumber": "12345",
    "from": "0x789...",
    "to": "0xabc...",
    "value": "1000000000000000000",
    "gasPrice": "20000000000",
    "gas": "21000",
    "nonce": "5",
    "input": "0x",
    "transactionIndex": "0",
    "v": "0x1b",
    "r": "0xdef...",
    "s": "0x123..."
  }
}
```

## Development

### Building
```bash
cargo build
```

### Testing
```bash
cargo test
```

### Running
```bash
# Start GraphQL server
cargo run --bin solver_service

# Start HTTP server
cd client && go run main.go
``` 
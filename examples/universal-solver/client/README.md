## Running the Client

The client requires a seed phrase to derive chain-specific keys for transaction signing. You can run the client with:

```bash
go run main.go -seed-phrase "your twelve word seed phrase here"
```

Additional optional flags:
- `-solver-url`: Universal Solver service URL (default: http://localhost:8080/)
- `-solana-url`: Solana RPC endpoint (default: http://localhost:8899)
- `-ethereum-url`: Ethereum RPC endpoint (default: http://localhost:8545)

Example:
```bash
go run main.go \
  -seed-phrase "your twelve word seed phrase here" \
  -solver-url "http://custom-solver:8080" \
  -solana-url "http://custom-solana:8899" \
  -ethereum-url "http://custom-ethereum:8545"
```

**Important**: Keep your seed phrase secure and never share it. The seed phrase is used to derive private keys for both Ethereum and Solana chains. 
package solver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gorilla/websocket"
	"github.com/linera-protocol/examples/universal-solver/client/solver/keys"
	"github.com/mr-tron/base58"
)

// Add at the top with other package-level variables
var (
	// RPC endpoints
	EthereumRPC string
	SolanaRPC   string
	DojimaRPC   string
	// Chain keys
	chainKeys *keys.ChainKeys
	// Linera service management
	activeService *LineraService
	serviceMutex  sync.Mutex
	// Linera configuration
	lineraConfig *LineraConfig
	// Linera executable path
	lineraPath string // New variable to hold the Linera executable path
)

// Add a function to initialize RPC URLs
func InitRPCEndpoints(ethereumURL, solanaURL, dojimaURL string) {
	EthereumRPC = ethereumURL
	SolanaRPC = solanaURL
	DojimaRPC = dojimaURL
	Logger.Printf("Initialized RPC endpoints - Ethereum: %s, Solana: %s", ethereumURL, solanaURL)
}

// InitKeys initializes the private keys from a seed phrase
func InitKeys(seedPhrase string) error {
	var err error
	chainKeys, err = keys.DeriveKeysFromSeedPhrase(seedPhrase)
	if err != nil {
		Logger.Printf("Failed to derive keys: %v", err)
		return fmt.Errorf("failed to derive keys: %w", err)
	}
	Logger.Printf("Successfully initialized chain keys")
	return nil
}

type WSMessage struct {
	Type  string      `json:"type"`
	Data  interface{} `json:"data"`
	Error string      `json:"error,omitempty"`
}

type Client struct {
	baseURL string
	http    *http.Client

	// WebSocket related fields
	upgrader    websocket.Upgrader
	clients     map[*websocket.Conn]bool
	clientsLock sync.RWMutex
	broadcast   chan WSMessage
}

func NewClient(baseURL, lineraExecutablePath string) *Client {
	lineraPath = lineraExecutablePath // Set the Linera path

	client := &Client{
		baseURL:   baseURL,
		http:      &http.Client{},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan WSMessage),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development/testing
			},
		},
	}

	// Start broadcast handler
	go client.handleBroadcasts()

	return client
}

// GetSolanaTransaction fetches transaction details from Solana
func (c *Client) GetSolanaTransaction(_, txHash string) (interface{}, error) {
	// Prepare the JSON-RPC request
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTransaction",
		"params": []interface{}{
			txHash,
			map[string]interface{}{
				"encoding":                       "json",
				"maxSupportedTransactionVersion": 0,
			},
		},
	}

	// Make the request with retries
	var response interface{}
	var err error
	for i := 0; i < 20; i++ {
		response, err = c.makeRPCRequest(SolanaRPC, requestBody)
		if responseMap, ok := response.(map[string]interface{}); ok {
			if responseMap["result"] == nil {
				time.Sleep(5 * time.Second)
				continue // Retry if result is nil
			}
		}

		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get Solana transaction after 10 retries: %w", err)
	}

	return response, nil
}

// GetEthereumTransaction fetches transaction details from Ethereum
func (c *Client) GetEthereumTransaction(_, txHash string) (interface{}, error) {
	client, err := ethclient.Dial(EthereumRPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}
	defer client.Close()

	hash := common.HexToHash(txHash)
	tx, isPending, err := client.TransactionByHash(context.Background(), hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get Ethereum transaction: %w", err)
	}

	// Convert transaction to map for consistent response format
	return map[string]interface{}{
		"hash":      tx.Hash().Hex(),
		"value":     tx.Value().String(),
		"gas":       tx.Gas(),
		"gasPrice":  tx.GasPrice().String(),
		"nonce":     tx.Nonce(),
		"isPending": isPending,
	}, nil
}

func (c *Client) makeRPCRequest(endpoint string, requestBody interface{}) (interface{}, error) {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) GetFile(id string) (*SolverFile, error) {
	query := fmt.Sprintf(`{
		"query": "query { getFileSolverApp(id: \"%s\") { solverFileId owner name payload } }"
	}`, id)

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	var result GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &result.Data.GetFileSolverApp, nil
}

func (c *Client) GetTransactionByHash(hash string) (*Transaction, error) {
	query := fmt.Sprintf(`{
		"query": "query { getTransaction(hash: \"%s\") { 
			hash
			blockHash
			blockNumber
			from
			to
			value
			gasPrice
			gas
			nonce
			input
			transactionIndex
			v
			r
			s
	 }}"
	}`, hash)

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			GetTransaction *Transaction `json:"getTransaction"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return result.Data.GetTransaction, nil
}

// CalculateSwap calculates swap details without executing the swap
func (c *Client) CalculateSwap(fromToken, toToken string, amount float64) (*SwapResult, error) {
	// Prepare GraphQL query
	query := fmt.Sprintf(`{
		"query": "query { calculateSwap(fromToken:\"%s\",toToken:\"%s\",amount:%f) { fromToken toToken fromAmount toAmount exchangeRate } }"
	}`, fromToken, toToken, amount)

	// Create request
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		Data struct {
			CalculateSwap struct {
				FromToken    string  `json:"fromToken"`
				ToToken      string  `json:"toToken"`
				FromAmount   float64 `json:"fromAmount"`
				ToAmount     float64 `json:"toAmount"`
				ExchangeRate float64 `json:"exchangeRate"`
			} `json:"calculateSwap"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	return &SwapResult{
		FromToken:    result.Data.CalculateSwap.FromToken,
		ToToken:      result.Data.CalculateSwap.ToToken,
		FromAmount:   result.Data.CalculateSwap.FromAmount,
		ToAmount:     result.Data.CalculateSwap.ToAmount,
		ExchangeRate: result.Data.CalculateSwap.ExchangeRate,
	}, nil
}

// ExecuteSwap performs the swap operation
func (c *Client) ExecuteSwap(fromToken, toToken string, amount float64, destinationAddress string) (*SwapResponse, error) {
	// First calculate the swap
	swapResult, err := c.CalculateSwap(fromToken, toToken, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate swap: %w", err)
	}

	c.broadcast <- WSMessage{
		Type: "swap_calculate",
		Data: map[string]interface{}{
			"status":      "calculating_swap",
			"description": fmt.Sprintf("from %s to %s", fromToken, toToken),
		},
	}

	// Execute the swap mutation
	mutation := fmt.Sprintf(`{"query":"mutation calSwap{swap(fromToken:\"%s\",toToken:\"%s\",amount:\"%v\",destinationAddress:\"%s\")}"}`, fromToken, toToken, amount, destinationAddress)

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer([]byte(mutation)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	var rawResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("error parsing raw response: %w", err)
	}

	// Create properly structured result
	var result struct {
		Data   string `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	// Re-encode and decode to ensure proper type conversion
	jsonData, err := json.Marshal(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("error re-encoding response: %w", err)
	}

	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("error parsing structured response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	swapResponse := &SwapResponse{
		TxHash:             result.Data,
		SwapResult:         *swapResult,
		Status:             "pending",
		DestinationAddress: destinationAddress,
	}

	// Prepare transaction for signing based on chain
	chain := c.determineChain(toToken)
	if err := c.PrepareTransaction(chain, swapResponse); err != nil {
		return nil, fmt.Errorf("failed to prepare transaction: %w", err)
	}

	c.broadcast <- WSMessage{
		Type: "swap_prepare",
		Data: map[string]interface{}{
			"status":      "prepare",
			"description": fmt.Sprintf("preparing transaction for signing"),
		},
	}

	// Sign the prepared transaction
	if err := c.SignTransaction(swapResponse); err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	c.broadcast <- WSMessage{
		Type: "swap_sign",
		Data: map[string]interface{}{
			"status":      "sign",
			"description": fmt.Sprintf("signing transaction for submission"),
		},
	}

	// Submit the signed transaction
	if err := c.SubmitTransaction(swapResponse); err != nil {
		return nil, fmt.Errorf("failed to submit transaction: %w", err)
	}

	c.broadcast <- WSMessage{
		Type: "swap_complete",
		Data: map[string]interface{}{
			"status":      "complete",
			"description": fmt.Sprintf("successfully signed transaction for submission"),
		},
	}

	return swapResponse, nil
}

func (c *Client) determineChain(token string) string {
	switch token {
	case "ETH":
		return "ethereum"
	case "SOL":
		return "solana"
	default:
		return "unknown"
	}
}

// PrepareTransaction prepares a transaction for signing based on chain type
func (c *Client) PrepareTransaction(chain string, swap *SwapResponse) error {
	switch chain {
	case "ethereum":
		return c.prepareEthereumTransaction(swap)
	case "solana":
		return c.prepareSolanaTransaction(swap)
	default:
		return fmt.Errorf("unsupported chain: %s", chain)
	}
}

// GetAllPools fetches all pool addresses
func (c *Client) GetAllPools() ([]Pool, error) {
	query := `{"query":"query pools{getAllPools{chainName poolAddress}}"}`

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			GetAllPools []Pool `json:"getAllPools"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	// Accumulate pools from response
	var pools []Pool
	for _, pool := range result.Data.GetAllPools {
		pools = append(pools, Pool{
			ChainName:   pool.ChainName,
			PoolAddress: pool.PoolAddress,
		})
	}

	return pools, nil
}

// GetAllPoolBalances fetches all pool balances
func (c *Client) GetAllPoolBalances() ([]PoolBalance, error) {
	query := `{"query":"query balances{getAllPoolBalances{poolAddress balance}}"}`

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			GetAllPoolBalances []PoolBalance `json:"getAllPoolBalances"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	return result.Data.GetAllPoolBalances, nil
}

// GetPool fetches pool address for a specific chain
func (c *Client) GetPool(chain string) (string, error) {
	// Reuse existing getPoolAddress method
	return c.getPoolAddress(chain)
}

// getPoolAddress gets the pool address for a given token
func (c *Client) getPoolAddress(token string) (string, error) {
	pools, err := c.GetAllPools()
	if err != nil {
		return "", fmt.Errorf("failed to get pools: %w", err)
	}

	for _, pool := range pools {
		if pool.ChainName == token {
			return pool.PoolAddress, nil
		}
	}

	return "", fmt.Errorf("pool not found for token: %s", token)
}

// Update the prepareEthereumTransaction method
func (c *Client) prepareEthereumTransaction(swap *SwapResponse) error {
	// Get pool address for the token
	fromAddress, err := c.getPoolAddress(swap.SwapResult.ToToken)
	if err != nil {
		return fmt.Errorf("failed to get source pool address: %w", err)
	}

	// Query Ethereum node for current gas price
	client, err := ethclient.Dial(EthereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}
	defer client.Close()

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Get nonce for the from address
	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(fromAddress))
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	// Prepare transaction parameters
	swap.TxToSign = &TransactionPrep{
		Chain: "ethereum",
		RawTx: "", // Will be filled by the signer
		ChainParams: ChainParams{
			FromAddress: fromAddress,
			ToAddress:   swap.DestinationAddress,
			Amount:      fmt.Sprintf("%f", swap.SwapResult.ToAmount),
			GasPrice:    gasPrice.String(),
			GasLimit:    21000, // Standard ETH transfer gas limit
			Nonce:       nonce,
		},
	}
	return nil
}

// Update the prepareSolanaTransaction method
func (c *Client) prepareSolanaTransaction(swap *SwapResponse) error {
	// Get pool address for the token
	fromAddress, err := c.getPoolAddress(swap.SwapResult.ToToken)
	if err != nil {
		return fmt.Errorf("failed to get source pool address: %w", err)
	}

	// Query Solana node for recent blockhash
	client := rpc.New(SolanaRPC)
	resp, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Prepare transaction parameters
	swap.TxToSign = &TransactionPrep{
		Chain: "solana",
		RawTx: "", // Will be filled by the signer
		ChainParams: ChainParams{
			FromAddress:     fromAddress,
			ToAddress:       swap.DestinationAddress,
			Amount:          fmt.Sprintf("%f", swap.SwapResult.ToAmount),
			RecentBlockhash: resp.Value.Blockhash.String(),
			Lamports:        swap.SwapResult.ToAmount,
		},
	}
	return nil
}

// SignTransaction signs the prepared transaction based on chain type
func (c *Client) SignTransaction(swap *SwapResponse) error {
	if swap.TxToSign == nil {
		return fmt.Errorf("no transaction prepared for signing")
	}

	switch swap.TxToSign.Chain {
	case "ethereum":
		return c.signEthereumTransaction(swap)
	case "solana":
		return c.signSolanaTransaction(swap)
	default:
		return fmt.Errorf("unsupported chain for signing: %s", swap.TxToSign.Chain)
	}
}

func (c *Client) signEthereumTransaction(swap *SwapResponse) error {
	// Get derived Ethereum key instead of environment variable
	if chainKeys == nil || chainKeys.EthereumKey == nil {
		return fmt.Errorf("ethereum private key not initialized")
	}

	// Create the transaction object
	tx := types.NewTransaction(
		swap.TxToSign.ChainParams.Nonce,
		common.HexToAddress(swap.TxToSign.ChainParams.ToAddress),
		func() *big.Int {
			// Convert decimal to integer by multiplying by 10^18 (standard ETH decimals)
			amountFloat, _ := strconv.ParseFloat(swap.TxToSign.ChainParams.Amount, 64)
			amountBigFloat := new(big.Float).SetFloat64(amountFloat)
			multiplier := new(big.Float).SetFloat64(1e18)
			result := new(big.Float).Mul(amountBigFloat, multiplier)

			amountBigInt := new(big.Int)
			result.Int(amountBigInt)
			return amountBigInt
		}(),
		swap.TxToSign.ChainParams.GasLimit,
		func() *big.Int {
			gasPrice, _ := new(big.Int).SetString(swap.TxToSign.ChainParams.GasPrice, 10)
			return gasPrice
		}(),
		nil, // data
	)

	// Get the signer
	chainID := big.NewInt(1337) // mainnet, adjust as needed
	signer := types.NewEIP155Signer(chainID)

	// Sign the transaction
	signedTx, err := types.SignTx(tx, signer, chainKeys.EthereumKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Convert to raw bytes
	rawTxBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to encode signed transaction: %w", err)
	}

	// Store the raw signed transaction
	swap.TxToSign.RawTx = hexutil.Encode(rawTxBytes)
	return nil
}

func (c *Client) signSolanaTransaction(swap *SwapResponse) error {
	// Get derived Solana key instead of environment variable
	if chainKeys == nil || chainKeys.SolanaKey == nil {
		return fmt.Errorf("solana private key not initialized")
	}

	from_address, err := solana.PublicKeyFromBase58(swap.TxToSign.ChainParams.FromAddress)
	if err != nil {
		return fmt.Errorf("failed to get from address: %w", err)
	}
	to_address, err := solana.PublicKeyFromBase58(swap.TxToSign.ChainParams.ToAddress)

	if err != nil {
		return fmt.Errorf("failed to get to address: %w", err)
	}

	// Create a new transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				uint64(swap.TxToSign.ChainParams.Lamports),
				from_address,
				to_address,
			).Build(),
		},
		solana.MustHashFromBase58(swap.TxToSign.ChainParams.RecentBlockhash),
	)

	// Sign the transaction
	_, _ = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if chainKeys.SolanaKey.PublicKey().Equals(key) {
				return chainKeys.SolanaKey
			}
			return nil
		},
	)

	// Store the raw signed transaction
	rawTx, err := tx.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to serialize signed transaction: %w", err)
	}
	swap.TxToSign.RawTx = base58.Encode(rawTx)

	return nil
}

// SubmitTransaction submits the signed transaction to the appropriate chain
func (c *Client) SubmitTransaction(swap *SwapResponse) error {
	if swap.TxToSign == nil || swap.TxToSign.RawTx == "" {
		return fmt.Errorf("no signed transaction available")
	}

	switch swap.TxToSign.Chain {
	case "ethereum":
		return c.submitEthereumTransaction(swap)
	case "solana":
		return c.submitSolanaTransaction(swap)
	default:
		return fmt.Errorf("unsupported chain for submission: %s", swap.TxToSign.Chain)
	}
}

func (c *Client) submitEthereumTransaction(swap *SwapResponse) error {
	// Connect to Ethereum node
	client, err := ethclient.Dial(EthereumRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}
	defer client.Close()

	// Decode raw transaction
	rawTxBytes, err := hexutil.Decode(swap.TxToSign.RawTx)
	if err != nil {
		return fmt.Errorf("failed to decode raw transaction: %w", err)
	}

	var tx types.Transaction
	if err := tx.UnmarshalBinary(rawTxBytes); err != nil {
		return fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	// Submit transaction
	if err := client.SendTransaction(context.Background(), &tx); err != nil {
		return fmt.Errorf("failed to submit transaction: %w", err)
	}

	// Update response with transaction hash
	swap.TxHash = tx.Hash().Hex()
	swap.Status = "submitted"

	return nil
}

func (c *Client) submitSolanaTransaction(swap *SwapResponse) error {
	// Create RPC request
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "sendTransaction",
		"params": []interface{}{
			swap.TxToSign.RawTx,
			map[string]interface{}{
				"encoding": "base58",
			},
		},
	}

	// Submit transaction
	response, err := c.makeRPCRequest(SolanaRPC, requestBody)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %w", err)
	}

	// Extract transaction signature
	result, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	if errMsg, hasError := result["error"]; hasError {
		return fmt.Errorf("RPC error: %v", errMsg)
	}

	signature, ok := result["result"].(string)
	if !ok {
		return fmt.Errorf("invalid signature format in response")
	}

	// Update response with transaction signature
	swap.TxHash = signature
	swap.Status = "submitted"

	return nil
}

func (c *Client) RequestSolanaAirdrop(address string) (map[string]interface{}, error) {
	// Create RPC client
	client := rpc.New(SolanaRPC)

	// Parse address
	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Solana address: %w", err)
	}

	c.broadcast <- WSMessage{
		Type: "faucet_start",
		Data: map[string]interface{}{
			"status":      "requested",
			"description": fmt.Sprintf("faucet request is initiated"),
		},
	}

	// Request airdrop (2 SOL)
	sig, err := client.RequestAirdrop(
		context.Background(),
		pubKey,
		2*solana.LAMPORTS_PER_SOL,
		rpc.CommitmentFinalized,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to request airdrop: %w", err)
	}

	c.broadcast <- WSMessage{
		Type: "faucet_complete",
		Data: map[string]interface{}{
			"status":      "completed",
			"description": fmt.Sprintf("successfully faucet is completed"),
		},
	}

	// Wait for confirmation
	// _, err = client.GetConfirmedTransactionWithOpts(context.Background(), sig, &rpc.GetTransactionOpts{
	// 	Commitment: rpc.CommitmentConfirmed,
	// })
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to confirm airdrop: %w", err)
	// }

	return map[string]interface{}{
		"signature": sig.String(),
		"amount":    "2 SOL",
		"address":   address,
	}, nil
}

// RequestEthereumFaucet sends a fixed amount of ETH to the specified address
func (c *Client) RequestEthereumFaucet(address, chain string) (map[string]interface{}, error) {
	// For testnet/local network only
	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid Ethereum address")
	}
	c.broadcast <- WSMessage{
		Type: "faucet_start",
		Data: map[string]interface{}{
			"status":      "requested",
			"description": fmt.Sprintf("faucet request is initiated"),
		},
	}

	// Select appropriate RPC URL
	var rpcURL string
	if chain == "dojima" {
		rpcURL = DojimaRPC
	} else {
		rpcURL = EthereumRPC
	}

	// Connect to the network
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}
	defer client.Close()

	// Get the faucet's private key
	faucetKey := chainKeys.EthereumKey
	if faucetKey == nil {
		return nil, fmt.Errorf("faucet private key not initialized")
	}

	// Get the faucet's address
	faucetAddress := crypto.PubkeyToAddress(faucetKey.PublicKey)

	// Get the latest nonce for the faucet account
	nonce, err := client.PendingNonceAt(context.Background(), faucetAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get the current gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create transaction data
	value := big.NewInt(1000000000000000000) // 1 ETH in wei
	toAddress := common.HexToAddress(address)
	gasLimit := uint64(21000) // Standard gas limit for ETH transfers

	// Create the transaction
	tx := types.NewTransaction(
		nonce,
		toAddress,
		value,
		gasLimit,
		gasPrice,
		nil, // No data for simple transfers
	)

	// Get the chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), faucetKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	c.broadcast <- WSMessage{
		Type: "faucet_complete",
		Data: map[string]interface{}{
			"status":      "completed",
			"description": fmt.Sprintf("successfully faucet is completed"),
		},
	}

	// Return the transaction details
	return map[string]interface{}{
		"txHash":   signedTx.Hash().Hex(),
		"from":     faucetAddress.Hex(),
		"to":       address,
		"value":    value.String(),
		"chain":    chain,
		"gasPrice": gasPrice.String(),
		"gasLimit": gasLimit,
	}, nil
}

// RequestEthereumFaucetWithAmount sends a specified amount of ETH
func (c *Client) RequestEthereumFaucetWithAmount(address string, amount float64, chain string) (map[string]interface{}, error) {
	// For testnet/local network only
	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid Ethereum address")
	}

	// Select appropriate RPC URL
	var rpcURL string
	if chain == "dojima" {
		rpcURL = DojimaRPC
	} else {
		rpcURL = EthereumRPC
	}

	// Connect to the network
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}
	defer client.Close()

	// Get the faucet's private key
	faucetKey := chainKeys.EthereumKey
	if faucetKey == nil {
		return nil, fmt.Errorf("faucet private key not initialized")
	}

	// Get the faucet's address
	faucetAddress := crypto.PubkeyToAddress(faucetKey.PublicKey)

	// Get the latest nonce for the faucet account
	nonce, err := client.PendingNonceAt(context.Background(), faucetAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get the current gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Convert amount to wei
	amountInWei := new(big.Float).Mul(big.NewFloat(amount), big.NewFloat(1e18))
	value := new(big.Int)
	amountInWei.Int(value)

	// Create transaction data
	toAddress := common.HexToAddress(address)
	gasLimit := uint64(21000) // Standard gas limit for ETH transfers

	// Create the transaction
	tx := types.NewTransaction(
		nonce,
		toAddress,
		value,
		gasLimit,
		gasPrice,
		nil, // No data for simple transfers
	)

	// Get the chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), faucetKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Return the transaction details
	return map[string]interface{}{
		"txHash":   signedTx.Hash().Hex(),
		"from":     faucetAddress.Hex(),
		"to":       address,
		"value":    value.String(),
		"chain":    chain,
		"gasPrice": gasPrice.String(),
		"gasLimit": gasLimit,
	}, nil
}

// GetSolanaBalance fetches SOL balance for an address
func (c *Client) GetSolanaBalance(address string) (*Balance, error) {
	// Create RPC client
	client := rpc.New(SolanaRPC)

	// Parse address
	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Solana address: %w", err)
	}
	c.broadcast <- WSMessage{
		Type: "balance_started",
		Data: map[string]interface{}{
			"status":      "balance_started",
			"description": fmt.Sprintf("balance fetch initiated"),
		},
	}
	// Get balance
	balance, err := client.GetBalance(
		context.Background(),
		pubKey,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	// Convert lamports to SOL
	solBalance := float64(balance.Value) / float64(solana.LAMPORTS_PER_SOL)
	c.broadcast <- WSMessage{
		Type: "balance_completed",
		Data: map[string]interface{}{
			"status":      "balance_completed",
			"description": fmt.Sprintf("successfully balance is fetched"),
		},
	}
	return &Balance{
		Address: address,
		Amount:  solBalance,
		Symbol:  "SOL",
	}, nil
}

// GetEthereumBalance fetches ETH balance for an address
func (c *Client) GetEthereumBalance(address, chain string) (*Balance, error) {
	// Validate address
	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid Ethereum address")
	}

	c.broadcast <- WSMessage{
		Type: "balance_started",
		Data: map[string]interface{}{
			"status":      "balance_started",
			"description": fmt.Sprintf("balance fetch initiated"),
		},
	}

	var rpcURL string
	var symbol string
	// Connect to Ethereum node
	if chain == "dojima" {
		rpcURL = DojimaRPC
		symbol = "DOJ"
	} else {
		rpcURL = EthereumRPC
		symbol = "ETH"
	}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}
	defer client.Close()

	// Get balance
	account := common.HexToAddress(address)
	balance, err := client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	// Convert wei to ETH
	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	amount, _ := ethValue.Float64()
	c.broadcast <- WSMessage{
		Type: "balance_completed",
		Data: map[string]interface{}{
			"status":      "balance_completed",
			"description": fmt.Sprintf("successfully balance is fetched"),
		},
	}

	return &Balance{
		Address: address,
		Amount:  amount,
		Symbol:  symbol,
	}, nil
}

// Add new functions with amount parameter
func (c *Client) RequestSolanaAirdropWithAmount(address string, amount float64) (map[string]interface{}, error) {
	// Convert amount to lamports (1 SOL = 1e9 lamports)
	lamports := uint64(amount * 1e9)

	client := rpc.New(SolanaRPC)
	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Solana address: %w", err)
	}

	sig, err := client.RequestAirdrop(
		context.Background(),
		pubKey,
		lamports,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to request airdrop: %w", err)
	}

	return map[string]interface{}{
		"signature": sig.String(),
		"amount":    fmt.Sprintf("%f SOL", amount),
		"address":   address,
	}, nil
}

// ApplicationResponse represents the response from creating an application
type ApplicationResponse struct {
	ApplicationID string `json:"application_id"`
	ChainID       string `json:"chain_id"`
	URL           string `json:"url"`
}

// Helper function to execute command and capture both stdout and stderr
func executeCommand(cmd *exec.Cmd) (string, error) {
	// Create pipes for both stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %v", err)
	}

	// Read both stdout and stderr
	var stdoutBuilder, stderrBuilder strings.Builder
	var wg sync.WaitGroup

	// Read stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			stdoutBuilder.WriteString(line + "\n")
			Logger.Printf("[Command Output] %s", line)
		}
	}()

	// Read stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuilder.WriteString(line + "\n")
			Logger.Printf("[Command Error] %s", line)
		}
	}()

	// Wait for both readers to finish
	wg.Wait()

	// Wait for command to complete
	err = cmd.Wait()
	if err != nil {
		// Include stderr in error message if available
		if errOutput := strings.TrimSpace(stderrBuilder.String()); errOutput != "" {
			return "", fmt.Errorf("command failed: %v\nError output:\n%s", err, errOutput)
		}
		return "", fmt.Errorf("command failed: %v", err)
	}

	return stdoutBuilder.String(), nil
}

// LineraConfig holds common configuration for Linera services
type LineraConfig struct {
	WalletPath  string
	StoragePath string
	Chain1      string
	Owner1      string
	Chain2      string
	Owner2      string
}

// InitLineraConfig initializes the Linera configuration
func InitLineraConfig() error {
	// Create temp directory for Linera files
	// tmpDir, err := os.MkdirTemp("", "linera_*")
	// if err != nil {
	// 	return fmt.Errorf("failed to create temp directory: %v", err)
	// }

	// Initialize config
	lineraConfig = &LineraConfig{
		WalletPath:  os.Getenv("LINERA_WALLET"),
		StoragePath: os.Getenv("LINERA_STORAGE"),
		Chain1:      os.Getenv("CHAIN_1"),
		Owner1:      os.Getenv("OWNER_1"),
		Chain2:      os.Getenv("CHAIN_2"),
		Owner2:      os.Getenv("OWNER_2"),
	}

	// Create wallet file
	// if err := createWalletFile(lineraConfig.WalletPath); err != nil {
	// 	return fmt.Errorf("failed to create wallet file: %v", err)
	// }
	//
	// // Create storage directory
	// if err := os.MkdirAll(lineraConfig.StoragePath, 0755); err != nil {
	// 	return fmt.Errorf("failed to create storage directory: %v", err)
	// }

	// Logger.Printf("Initialized Linera configuration in %s", tmpDir)
	return nil
}

// createWalletFile creates an empty wallet file
func createWalletFile(path string) error {
	// Create an empty JSON wallet file
	walletData := []byte("{}")
	if err := os.WriteFile(path, walletData, 0644); err != nil {
		return fmt.Errorf("failed to write wallet file: %v", err)
	}
	return nil
}

// GetLineraEnv returns the environment variables for Linera commands
func GetLineraEnv() []string {
	if lineraConfig == nil {
		Logger.Printf("Warning: Linera configuration not initialized")
		return nil
	}
	return []string{
		fmt.Sprintf("LINERA_WALLET=%s", lineraConfig.WalletPath),
		fmt.Sprintf("LINERA_STORAGE=%s", lineraConfig.StoragePath),
		fmt.Sprintf("CHAIN_1=%s", lineraConfig.Chain1),
		fmt.Sprintf("OWNER_1=%s", lineraConfig.Owner1),
		fmt.Sprintf("CHAIN_2=%s", lineraConfig.Chain2),
		fmt.Sprintf("OWNER_2=%s", lineraConfig.Owner2),
	}
}

// PublishBytecodeFromFiles publishes bytecode using the Linera executable
func (c *Client) PublishBytecodeFromFiles(contractPath, servicePath string) (string, error) {
	Logger.Printf("Publishing bytecode from files: %s, %s", contractPath, servicePath)

	// Verify files exist
	if _, err := os.Stat(contractPath); err != nil {
		return "", fmt.Errorf("contract file not found: %v", err)
	}
	if _, err := os.Stat(servicePath); err != nil {
		return "", fmt.Errorf("service file not found: %v", err)
	}

	// Prepare and execute command
	cmd := exec.Command(lineraPath, "publish-bytecode", contractPath, servicePath) // Use lineraPath here
	cmd.Env = append(os.Environ(), GetLineraEnv()...)

	// Execute command and capture output
	output, err := executeCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to publish bytecode: %w", err)
	}

	bytecodeID := strings.TrimSpace(output)
	if len(bytecodeID) == 0 {
		return "", fmt.Errorf("failed to extract bytecode ID from output: %s", output)
	}
	Logger.Printf("Successfully published bytecode with ID: %s", bytecodeID)
	return bytecodeID, nil
}

// CreateApplication executes the Linera create-application command with the provided bytecode ID
func (c *Client) CreateApplication(bytecodeID string) (*ApplicationResponse, error) {
	Logger.Printf("Creating application with bytecode ID: %s", bytecodeID)

	bytecodeIDs := strings.Fields(bytecodeID)
	strs := []string{
		"create-application",
	}
	strs = append(strs, bytecodeIDs...)
	cmd := exec.Command(lineraPath, strs...) // Assuming bytecodeIDs has at least one element
	cmd.Env = append(os.Environ(), GetLineraEnv()...)

	// Execute command and capture output
	output, err := executeCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Parse the output to get the application ID
	appID := strings.TrimSpace(output)
	chainID := "e476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65" // Using Chain1 as default

	// Construct the application URL
	appURL := fmt.Sprintf("https://linera-api.ngrok.io/chains/%s/applications/%s", chainID, appID)

	response := &ApplicationResponse{
		ApplicationID: appID,
		ChainID:       chainID,
		URL:           appURL,
	}

	Logger.Printf("Successfully created application with ID: %s, URL: %s", appID, appURL)
	return response, nil
}

// LineraService represents a running Linera service instance
type LineraService struct {
	Process *exec.Cmd
	Port    int
}

// StartLineraService starts a new Linera service on the specified port
func (c *Client) StartLineraService(port int) error {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()

	if activeService != nil {
		return fmt.Errorf("linera service is already running on port %d", activeService.Port)
	}

	Logger.Printf("Starting Linera service on port %d", port)

	// Prepare the command
	cmd := exec.Command(lineraPath, "service", "--port", strconv.Itoa(port))
	cmd.Env = append(os.Environ(), GetLineraEnv()...)

	// Set up logging
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the service
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start linera service: %v", err)
	}

	// Create new service instance
	activeService = &LineraService{
		Process: cmd,
		Port:    port,
	}

	// Handle output in background
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			Logger.Printf("[Linera Service] %s", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			Logger.Printf("[Linera Service Error] %s", scanner.Text())
		}
	}()

	// Wait for service to be ready
	time.Sleep(2 * time.Second)

	Logger.Printf("Linera service started successfully on port %d", port)
	return nil
}

// StopLineraService stops the running Linera service
func (c *Client) StopLineraService() error {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()

	if activeService == nil {
		return fmt.Errorf("no linera service is running")
	}

	Logger.Printf("Stopping Linera service on port %d", activeService.Port)

	// Send interrupt signal
	if err := activeService.Process.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to send interrupt signal: %v", err)
	}

	// Wait for process to exit
	if err := activeService.Process.Wait(); err != nil {
		Logger.Printf("Service exited with error: %v", err)
	}

	activeService = nil
	Logger.Printf("Linera service stopped successfully")
	return nil
}

// GetServiceStatus returns the current status of the Linera service
func (c *Client) GetServiceStatus() map[string]interface{} {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()

	if activeService == nil {
		return map[string]interface{}{
			"status": "stopped",
		}
	}

	return map[string]interface{}{
		"status": "running",
		"port":   activeService.Port,
	}
}

// HandleWebSocket manages WebSocket connections
func (c *Client) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	// Register new client
	c.clientsLock.Lock()
	c.clients[conn] = true
	c.clientsLock.Unlock()

	// Clean up on disconnect
	defer func() {
		c.clientsLock.Lock()
		delete(c.clients, conn)
		c.clientsLock.Unlock()
		conn.Close()
	}()

	// Send initial connection message
	err = conn.WriteJSON(WSMessage{
		Type: "connected",
		Data: "Successfully connected to WebSocket",
	})
	if err != nil {
		fmt.Println("Error sending initial message:", err)
		return
	}

	// Message handling loop
	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Println("WebSocket error:", err)
			break
		}

		// Handle different message types
		switch msg.Type {
		case "ping":
			conn.WriteJSON(WSMessage{
				Type: "pong",
				Data: "pong",
			})
		default:
			fmt.Println("Unknown message type:", msg.Type)
		}
	}
}

// handleBroadcasts sends messages to all connected clients
func (c *Client) handleBroadcasts() {
	for msg := range c.broadcast {
		c.clientsLock.RLock()
		for client := range c.clients {
			err := client.WriteJSON(msg)
			if err != nil {
				fmt.Println("Error broadcasting to client:", err)
				client.Close()
				delete(c.clients, client)
			}
		}
		c.clientsLock.RUnlock()
	}
}

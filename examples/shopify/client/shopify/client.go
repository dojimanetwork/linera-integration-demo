package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// RPC endpoints
	EthereumRPC string
	SolanaRPC   string
)

// Client provides methods to interact with the Shopify and Linera services
type Client struct {
	ShopifyURL string
	LineraURL  string
	HTTPClient *http.Client
}

// Add a function to initialize RPC URLs
func InitConfig(ethereumURL, solanaURL string) {
	EthereumRPC = ethereumURL
	SolanaRPC = solanaURL
}

// NewClient creates a new Shopify client with the provided URLs
func NewClient(shopifyURL, lineraURL string) *Client {
	return &Client{
		ShopifyURL: shopifyURL,
		LineraURL:  lineraURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Add function to mint NFT
func (c *Client) MintNFT(params ListNFTParams, blobHash string, token string) error {
	Logger.Printf("Minting NFT with params: %+v, blobHash: %s", params, blobHash)

	mutation := fmt.Sprintf(`{
		"query": "mutation mint{mint(minter:\"%s\",name:\"%s\",blobHash:\"%s\",token:\"%s\",price:\"%s\",chainMinter:\"%s\",chainOwner:\"%s\",description:\"%s\")}"
	}`, params.Minter, params.Name, blobHash, token, params.Price, params.ChainMinter, params.ChainOwner, params.Description)

	req, err := http.NewRequest("POST", c.ShopifyURL, bytes.NewBuffer([]byte(mutation)))
	if err != nil {
		Logger.Printf("Error creating mint request: %v", err)
		return fmt.Errorf("error creating mint request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		Logger.Printf("Error minting NFT: %v", err)
		return fmt.Errorf("error minting NFT: %w", err)
	}
	defer resp.Body.Close()

	var mintResp MintResponse
	if err := json.NewDecoder(resp.Body).Decode(&mintResp); err != nil {
		Logger.Printf("Error parsing mint response: %v", err)
		return fmt.Errorf("error parsing mint response: %w", err)
	}

	if len(mintResp.Errors) > 0 {
		Logger.Printf("Mint error: %s", mintResp.Errors[0].Message)
		return fmt.Errorf("mint error: %s", mintResp.Errors[0].Message)
	}

	Logger.Printf("Successfully minted NFT with transaction hash: %s", mintResp.Data)
	return nil
}

// Update ListNFT to return the blob hash
func (c *Client) ListNFT(params ListNFTParams) (string, error) {
	// Logger.Printf("Listing NFT with params: %+v", params)

	// First publish the image data blob
	// blobHash, err := c.PublishDataBlob(params.ChainId, params.ImageBytes)
	// if err != nil {
	// 	Logger.Printf("Failed to publish data blob: %v", err)
	// 	return "", fmt.Errorf("failed to publish data blob: %w", err)
	// }

	// Mint the NFT with the blob hash
	if err := c.MintNFT(params, params.BlobHash, params.Token); err != nil {
		Logger.Printf("Failed to mint NFT: %v", err)
		return "", fmt.Errorf("failed to mint NFT: %w", err)
	}

	Logger.Printf("Successfully listed NFT with blob hash: %s", params.BlobHash)
	return params.BlobHash, nil
}

// Update GetNFTDetails function
func (c *Client) GetNFTDetails(id string) (*NFTQueryResponse, error) {
	Logger.Printf("Fetching NFT details for blobHash: %s", id)
	query := `{
		"query": "query nft{nftUsingBlobHash(id:` + id + `){token tokenId price chainOwner chainMinter name owner minter payload}}"
	}`

	req, err := http.NewRequest("POST", c.ShopifyURL, bytes.NewBuffer([]byte(query)))
	if err != nil {
		Logger.Printf("Error creating NFT query request: %v", err)
		return nil, fmt.Errorf("error creating NFT query request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		Logger.Printf("Error executing NFT query: %v", err)
		return nil, fmt.Errorf("error executing NFT query: %w", err)
	}
	defer resp.Body.Close()

	var nftResp NFTQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&nftResp); err != nil {
		Logger.Printf("Error parsing NFT query response: %v", err)
		return nil, fmt.Errorf("error parsing NFT query response: %w", err)
	}
	return &nftResp, nil
}

// Add function to get all NFTs
func (c *Client) GetAllNFTs() (map[string]NFT, error) {
	Logger.Println("Getting all NFTs")

	query := `{
		"query": "query nfts{nfts}"
	}`

	req, err := http.NewRequest("POST", c.ShopifyURL, bytes.NewBuffer([]byte(query)))
	if err != nil {
		Logger.Printf("Error creating NFTs query request: %v", err)
		return nil, fmt.Errorf("error creating NFTs query request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		Logger.Printf("Error executing NFTs query: %v", err)
		return nil, fmt.Errorf("error executing NFTs query: %w", err)
	}
	defer resp.Body.Close()

	var nftsResp NFTsResponse
	if err := json.NewDecoder(resp.Body).Decode(&nftsResp); err != nil {
		Logger.Printf("Error parsing NFTs response: %v", err)
		return nil, fmt.Errorf("error parsing NFTs response: %w", err)
	}

	if len(nftsResp.Errors) > 0 {
		Logger.Printf("NFTs query error: %s", nftsResp.Errors[0].Message)
		return nil, fmt.Errorf("NFTs query error: %s", nftsResp.Errors[0].Message)
	}

	// Logger.Printf("Successfully retrieved NFTs: %+v", nftsResp.Data.NFTs)
	return nftsResp.Data.NFTs, nil
}

func (c *Client) ExecuteTransferMutation(params TransferParams) (*TransferResponse, string, error) {
	Logger.Printf("Executing transfer mutation with params: %+v", params)
	// First get the NFT details to get the ID

	//_, err := c.GetNFTDetails(params.)

	//if err != nil {
	//	Logger.Printf("Failed to get NFT details: %v", err)
	//	return nil, "", fmt.Errorf("failed to get NFT details: %w", err)
	//}
	var hash string

	// Preserve special chars while escaping / and +

	mutation := `{
    "query": "mutation transfer{transfer(sourceOwner:\"` + params.SourceOwner + `\", tokenId:\"` + params.TokenId + `\", targetAccount: { chainId:\"` + params.TargetChainId + `\", owner:\"` + params.TargetOwner + `\"}, chainOwner:\"` + params.ChainOwner + `\", buyFromToken:\"` + params.BuyFromToken + `\", amount:\"` + fmt.Sprintf("%v", params.Amount) + `\")}"
}`
	req, err := http.NewRequest("POST", c.ShopifyURL, bytes.NewBuffer([]byte(mutation)))
	if err != nil {
		Logger.Printf("Error creating transfer request: %v", err)
		return nil, "", fmt.Errorf("error creating transfer request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		Logger.Printf("Error executing transfer: %v", err)
		return nil, "", fmt.Errorf("error executing transfer: %w", err)
	}
	defer resp.Body.Close()

	var transferResp TransferResponse
	if err := json.NewDecoder(resp.Body).Decode(&transferResp); err != nil {
		Logger.Printf("Error parsing transfer response: %v", err)
		return nil, "", fmt.Errorf("error parsing transfer response: %w", err)
	}

	if len(transferResp.Errors) > 0 {
		Logger.Printf("Transfer error: %s", transferResp.Errors[0].Message)
		return nil, "", fmt.Errorf("transfer error: %s", transferResp.Errors[0].Message)
	}

	// Logger.Printf("Successfully executed transfer mutation: %+v", transferResp)
	return &transferResp, hash, nil
}

// GetBalances fetches all balances from the shopify service
func (c *Client) GetBalances() (map[string]interface{}, error) {
	Logger.Println("Getting all balances")

	query := `{
		"query": "query balances{balances}"
	}`

	req, err := http.NewRequest("POST", c.ShopifyURL, bytes.NewBuffer([]byte(query)))
	if err != nil {
		Logger.Printf("Error creating balances query request: %v", err)
		return nil, fmt.Errorf("error creating balances query request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		Logger.Printf("Error executing balances query: %v", err)
		return nil, fmt.Errorf("error executing balances query: %w", err)
	}
	defer resp.Body.Close()

	// Define a structure to parse the response
	var balancesResp struct {
		Data struct {
			Balances map[string]interface{} `json:"balances"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&balancesResp); err != nil {
		Logger.Printf("Error parsing balances response: %v", err)
		return nil, fmt.Errorf("error parsing balances response: %w", err)
	}

	if len(balancesResp.Errors) > 0 {
		Logger.Printf("Balances query error: %s", balancesResp.Errors[0].Message)
		return nil, fmt.Errorf("balances query error: %s", balancesResp.Errors[0].Message)
	}

	Logger.Printf("Successfully retrieved balances")
	return balancesResp.Data.Balances, nil
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

	resp, err := c.HTTPClient.Do(req)
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

// WithdrawToken withdraws tokens from the shopify service
func (c *Client) WithdrawToken(token string, amount string) (string, error) {
	Logger.Printf("Withdrawing %s %s", amount, token)

	mutation := fmt.Sprintf(`{
		"query": "mutation withdrawToken{withdrawToken(token:\"%s\", amount:\"%s\")}"
	}`, token, amount)

	req, err := http.NewRequest("POST", c.ShopifyURL, bytes.NewBuffer([]byte(mutation)))
	if err != nil {
		Logger.Printf("Error creating withdraw token request: %v", err)
		return "", fmt.Errorf("error creating withdraw token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		Logger.Printf("Error executing withdraw token: %v", err)
		return "", fmt.Errorf("error executing withdraw token: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var withdrawResp TransferResponse

	if err := json.NewDecoder(resp.Body).Decode(&withdrawResp); err != nil {
		Logger.Printf("Error parsing withdraw token response: %v", err)
		return "", fmt.Errorf("error parsing withdraw token response: %w", err)
	}

	if len(withdrawResp.Errors) > 0 {
		Logger.Printf("Withdraw token error: %s", withdrawResp.Errors[0].Message)
		return "", fmt.Errorf("withdraw token error: %s", withdrawResp.Errors[0].Message)
	}

	Logger.Printf("Successfully withdrew tokens with transaction hash: %s", withdrawResp.Data)
	return withdrawResp.Data, nil
}

// FilterNFTs filters NFTs by type (ON_SALE, SOLD, etc.)
func (c *Client) FilterNFTs(nftType NFTType) (map[string]NFT, error) {
	Logger.Printf("Filtering NFTs by type: %s", nftType)

	query := fmt.Sprintf(`{
		"query": "query filternfts{filterNfts(nftType:%s)}"
	}`, nftType)

	req, err := http.NewRequest("POST", c.ShopifyURL, bytes.NewBuffer([]byte(query)))
	if err != nil {
		Logger.Printf("Error creating filter NFTs request: %v", err)
		return nil, fmt.Errorf("error creating filter NFTs request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		Logger.Printf("Error executing filter NFTs: %v", err)
		return nil, fmt.Errorf("error executing filter NFTs: %w", err)
	}
	defer resp.Body.Close()

	var filterResp struct {
		Data struct {
			FilterNfts map[string]NFT `json:"filterNfts"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&filterResp); err != nil {
		Logger.Printf("Error parsing filter NFTs response: %v", err)
		return nil, fmt.Errorf("error parsing filter NFTs response: %w", err)
	}

	if len(filterResp.Errors) > 0 {
		Logger.Printf("Filter NFTs error: %s", filterResp.Errors[0].Message)
		return nil, fmt.Errorf("filter NFTs error: %s", filterResp.Errors[0].Message)
	}

	Logger.Printf("Successfully filtered NFTs, found %d items", len(filterResp.Data.FilterNfts))
	return filterResp.Data.FilterNfts, nil
}

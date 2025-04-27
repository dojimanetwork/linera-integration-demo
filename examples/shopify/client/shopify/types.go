package shopify

// WebhookNotification represents a notification to be sent to a webhook
type WebhookNotification struct {
	Status      string      `json:"status"`
	TxHash      string      `json:"txHash"`
	Chain       string      `json:"chain"`
	FromAddress string      `json:"fromAddress"`
	FromToken   string      `json:"fromToken"`
	Amount      float64     `json:"amount"`
	Timestamp   int64       `json:"timestamp"`
	Data        interface{} `json:"data,omitempty"`
	Client      string      `json:"client"`
}

type ListNFTParams struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       string `json:"price"`
	ChainId     string `json:"chainId"`
	Minter      string `json:"minter"`
	ChainMinter string `json:"chainMinter"`
	ChainOwner  string `json:"chainOwner"`
	Token       string `json:"token"`
	BlobHash    string `json:"blobHash"`
}

type MintResponse struct {
	Data   string `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

type TransferParams struct {
	SourceOwner   string  `json:"sourceOwner"`
	TokenId       string  `json:"tokenId"`
	TargetChainId string  `json:"targetChainId"`
	TargetOwner   string  `json:"targetOwner"`
	ChainOwner    string  `json:"chainOwner"`
	BuyFromToken  string  `json:"buyFromToken"`
	Amount        float64 `json:"amount"`
	BlobHash      string  `json:"blobHash"`
}

type TransferResponse struct {
	Data   string `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

type NFTQueryResponse struct {
	Data struct {
		NftUsingBlobHash struct {
			Token       string `json:"token"`
			TokenId     string `json:"tokenId"`
			Price       string `json:"price"`
			ChainOwner  string `json:"chainOwner"`
			ChainMinter string `json:"chainMinter"`
			Name        string `json:"name"`
			Owner       string `json:"owner"`
			ID          int    `json:"id"`
			Minter      string `json:"minter"`
			Payload     []int  `json:"payload"`
		} `json:"nftUsingBlobHash"`
	} `json:"data"`
}

// Add NFT type to represent individual NFT data
type NFT struct {
	TokenId     string `json:"tokenId"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Minter      string `json:"minter"`
	Payload     []int  `json:"payload"`
	Token       string `json:"token"`
	Price       string `json:"price"`
	ChainMinter string `json:"chainMinter"`
	ChainOwner  string `json:"chainOwner"`
	Description string `json:"description"`
	BlobHash    string `json:"blobHash"`
	NftStatus   string `json:"status"`
}

type NFTsResponse struct {
	Data struct {
		NFTs map[string]NFT `json:"nfts"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// NFTType defines the types of NFT status
type NFTType string

const (
	NFTTypeOnSale NFTType = "ON_SALE"
	NFTTypeSold   NFTType = "SOLD"
)

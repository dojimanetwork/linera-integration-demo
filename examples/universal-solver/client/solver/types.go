package solver

type SolverFile struct {
	SolverFileId string `json:"solverFileId"`
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	Payload      []byte `json:"payload"`
}

type GraphQLResponse struct {
	Data struct {
		GetFileSolverApp SolverFile `json:"getFileSolverApp"`
	} `json:"data"`
}

// Transaction represents an Ethereum transaction
type Transaction struct {
	Hash             string `json:"hash"`
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	From             string `json:"from"`
	To               string `json:"to"`
	Value            string `json:"value"`
	GasPrice         string `json:"gasPrice"`
	Gas              string `json:"gas"`
	Nonce            string `json:"nonce"`
	Input            string `json:"input"`
	TransactionIndex string `json:"transactionIndex"`
	V                string `json:"v"`
	R                string `json:"r"`
	S                string `json:"s"`
}

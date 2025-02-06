package solver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(endpoint string) *Client {
	return &Client{
		endpoint:   endpoint,
		httpClient: &http.Client{},
	}
}

func (c *Client) GetFile(id string) (*SolverFile, error) {
	query := fmt.Sprintf(`{
		"query": "query { getFileSolverApp(id: \"%s\") { solverFileId owner name payload } }"
	}`, id)

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/linera-protocol/examples/universal-solver/client/solver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSolverClient is a mock implementation of the solver client
type MockSolverClient struct {
	mock.Mock
}

func (m *MockSolverClient) GetEthereumTransaction(rpcEndpoint, txHash string) (interface{}, error) {
	args := m.Called(rpcEndpoint, txHash)
	return args.Get(0), args.Error(1)
}

func (m *MockSolverClient) GetSolanaTransaction(rpcEndpoint, txHash string) (interface{}, error) {
	args := m.Called(rpcEndpoint, txHash)
	return args.Get(0), args.Error(1)
}

func (m *MockSolverClient) ExecuteSwap(fromToken, toToken string, amount uint64, destinationAddress string) (*solver.SwapResponse, error) {
	args := m.Called(fromToken, toToken, amount, destinationAddress)
	return args.Get(0).(*solver.SwapResponse), args.Error(1)
}

func TestMain(m *testing.M) {
	// Set default values for flags during tests
	solver.InitRPCEndpoints(
		"http://localhost:8545", // Ethereum RPC
		"http://localhost:8899", // Solana RPC
	)
	solverClient = solver.NewClient("http://localhost:8080/chains/e476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65/applications/fe5249547ea2bd2a0754f42dc619007532c6bb1304bc5e7aaafaf646a044fd67cfb392dbb9f8603587a75e842d45992fc8e925f74565bd741f88718ce3a5a89be476187f6ddfeb9d588c7b45d3df334d5501d6499b3f9ad5595cae86cce16a65170000000000000000000000")

	// Run tests
	os.Exit(m.Run())
}

func TestHandlePostTxHash(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		queryParams      map[string]string
		mockSetup        func(*MockSolverClient)
		expectedStatus   int
		expectedResponse map[string]interface{}
		expectedErrorMsg string
		expectedTxPrep   *solver.TransactionPrep
	}{
		{
			name:   "Valid Ethereum Transaction with Swap",
			method: http.MethodPost,
			queryParams: map[string]string{
				"chain":              "ethereum",
				"txHash":             "0x106125634d7a095de31cb4c04a297011bc42c2becce8de788b1c30059192eda6",
				"toToken":            "AVAX",
				"destinationAddress": "0x807cF9A772d5a3f9CeFBc1192e939D62f0D9bD39",
			},
			mockSetup: func(m *MockSolverClient) {
				// Mock Ethereum transaction response
				m.On("GetEthereumTransaction", "http://localhost:8545", "0x106125634d7a095de31cb4c04a297011bc42c2becce8de788b1c30059192eda6").Return(map[string]interface{}{
					"hash":  "0x106125634d7a095de31cb4c04a297011bc42c2becce8de788b1c30059192eda6",
					"value": "0x10000000000000000000",
				}, nil)

				// Mock swap response with ETH as fromToken
				m.On("ExecuteSwap", "ETH", "AVAX", uint64(18446744073709551615), "0x807cF9A772d5a3f9CeFBc1192e939D62f0D9bD39").Return(&solver.SwapResponse{
					TxHash: "0x789def",
					SwapResult: solver.SwapResult{
						FromToken:    "ETH",
						ToToken:      "AVAX",
						FromAmount:   18446744073709551615,
						ToAmount:     18446744073709551615,
						ExchangeRate: 1.0,
					},
					Status: "pending",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"status": "success",
				"chain":  "ethereum",
				"data": map[string]interface{}{
					"hash":  "0x106125634d7a095de31cb4c04a297011bc42c2becce8de788b1c30059192eda6",
					"value": "0x10000000000000000000",
				},
				"swap_result": map[string]interface{}{
					"tx_hash": "0x789def",
					"swap_result": map[string]interface{}{
						"from_token":    "ETH",
						"to_token":      "AVAX",
						"from_amount":   float64(18446744073709551615),
						"to_amount":     float64(18446744073709551615),
						"exchange_rate": 1.0,
					},
					"status": "pending",
				},
			},
			expectedTxPrep: &solver.TransactionPrep{
				Chain: "ethereum",
				ChainParams: solver.ChainParams{
					FromAddress: "ETH",
					ToAddress:   "AVAX",
					Amount:      "18446744073709551615",
					GasPrice:    "20000000000",
					GasLimit:    21000,
				},
			},
		},
		{
			name:   "Valid Solana Transaction with Swap",
			method: http.MethodPost,
			queryParams: map[string]string{
				"chain":              "solana",
				"txHash":             "abc123",
				"toToken":            "ETH",
				"destinationAddress": "0xdef456",
			},
			mockSetup: func(m *MockSolverClient) {
				m.On("GetSolanaTransaction", "http://localhost:8899", "abc123").Return(map[string]interface{}{
					"result": map[string]interface{}{
						"hash":   "abc123",
						"amount": float64(1000000000), // 1 SOL
					},
				}, nil)

				m.On("ExecuteSwap", "solana", "ETH", uint64(1000000000), "0xdef456").Return(&solver.SwapResponse{
					TxHash: "0x456abc",
					SwapResult: solver.SwapResult{
						FromToken:    "SOL",
						ToToken:      "ETH",
						FromAmount:   1000000000,
						ToAmount:     40000000,
						ExchangeRate: 0.04,
					},
					Status: "pending",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"status": "success",
				"chain":  "solana",
				"data": map[string]interface{}{
					"result": map[string]interface{}{
						"hash":   "abc123",
						"amount": float64(1000000000),
					},
				},
				"swap_result": map[string]interface{}{
					"tx_hash": "0x456abc",
					"swap_result": map[string]interface{}{
						"from_token":    "SOL",
						"to_token":      "ETH",
						"from_amount":   float64(1000000000),
						"to_amount":     float64(40000000),
						"exchange_rate": 0.04,
					},
					"status": "pending",
				},
			},
		},
		{
			name:   "Missing Destination Address",
			method: http.MethodPost,
			queryParams: map[string]string{
				"chain":   "ethereum",
				"txHash":  "0x106125634d7a095de31cb4c04a297011bc42c2becce8de788b1c30059192eda6",
				"toToken": "SOL",
			},
			mockSetup: func(m *MockSolverClient) {
				m.On("GetEthereumTransaction", "http://localhost:8545", "0x106125634d7a095de31cb4c04a297011bc42c2becce8de788b1c30059192eda6").Return(map[string]interface{}{
					"hash":  "0x106125634d7a095de31cb4c04a297011bc42c2becce8de788b1c30059192eda6",
					"value": "0x10000000000000000000",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"status": "success",
				"chain":  "ethereum",
				"data": map[string]interface{}{
					"hash":  "0x106125634d7a095de31cb4c04a297011bc42c2becce8de788b1c30059192eda6",
					"value": "0x10000000000000000000",
				},
			},
		},
		{
			name:   "Invalid Method",
			method: http.MethodGet,
			queryParams: map[string]string{
				"chain":  "ethereum",
				"txHash": "0x123",
			},
			mockSetup:        func(m *MockSolverClient) {},
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedErrorMsg: "Method not allowed\n",
		},
		{
			name:   "Missing txHash",
			method: http.MethodPost,
			queryParams: map[string]string{
				"chain": "ethereum",
			},
			mockSetup:        func(m *MockSolverClient) {},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "txHash parameter is required\n",
		},
		{
			name:   "Missing chain",
			method: http.MethodPost,
			queryParams: map[string]string{
				"txHash": "0x123",
			},
			mockSetup:        func(m *MockSolverClient) {},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "chain parameter is required\n",
		},
		{
			name:   "Invalid chain",
			method: http.MethodPost,
			queryParams: map[string]string{
				"chain":  "invalid",
				"txHash": "0x123",
			},
			mockSetup:        func(m *MockSolverClient) {},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorMsg: "Invalid chain parameter. Must be 'solana' or 'ethereum'\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := new(MockSolverClient)
			tt.mockSetup(mockClient)

			// Replace global client with mock
			originalClient := solverClient
			// solverClient = mockClient
			defer func() {
				solverClient = originalClient
			}()

			// Create request
			req := httptest.NewRequest(tt.method, "/post_tx_hash", nil)
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			// Create response recorder
			rr := httptest.NewRecorder()

			// Handle request
			handlePostTxHash(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedErrorMsg != "" {
				assert.Equal(t, tt.expectedErrorMsg, rr.Body.String())
			} else {
				var response map[string]interface{}
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResponse, response)
			}

			// Verify all mocked calls were made
			mockClient.AssertExpectations(t)
		})
	}
}

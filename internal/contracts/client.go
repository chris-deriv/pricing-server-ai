package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"pricingserver/internal/common/logging"
)

// ContractServiceClient handles communication with the Python contracts service
type ContractServiceClient struct {
	baseURL string
	client  *http.Client
}

// NewContractServiceClient creates a new client for the contracts service
func NewContractServiceClient() *ContractServiceClient {
	baseURL := os.Getenv("CONTRACTS_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://contracts-service:8000" // default URL
	}
	logging.DebugLog("Creating new contract service client with base URL: %s", baseURL)
	return &ContractServiceClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// ContractParams represents the parameters needed to create a contract
type ContractParams struct {
	ContractType string                 `json:"contract_type"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// AddContract forwards contract creation to the Python service
func (c *ContractServiceClient) AddContract(contractID string, params ContractParams) error {
	// Ensure contract_id is set in parameters
	if params.Parameters == nil {
		params.Parameters = make(map[string]interface{})
	}
	params.Parameters["contract_id"] = contractID

	jsonBody, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	logging.DebugLog("Sending contract creation request to Python service: %s", string(jsonBody))

	// Send request to Python service
	resp, err := c.client.Post(
		fmt.Sprintf("%s/contracts", c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	logging.DebugLog("Received response from Python service: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("contract service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RemoveContract forwards contract removal to the Python service
func (c *ContractServiceClient) RemoveContract(contractID string) error {
	logging.DebugLog("Removing contract %s from Python service", contractID)
	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/contracts/%s", c.baseURL, contractID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	logging.DebugLog("Received response from Python service: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("contract service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdatePrice forwards price updates to the Python service and returns the response
func (c *ContractServiceClient) UpdatePrice(contractID string, price float64) ([]byte, error) {
	body := map[string]interface{}{
		"price": price,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	logging.DebugLog("Sending price update for contract %s: %s", contractID, string(jsonBody))

	resp, err := c.client.Post(
		fmt.Sprintf("%s/contracts/%s/price-update", c.baseURL, contractID),
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	logging.DebugLog("Received price update response for contract %s: %s", contractID, string(responseBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("contract service returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// GetProduct checks if a contract exists in the Python service
func (c *ContractServiceClient) GetProduct(contractID string) (bool, error) {
	logging.DebugLog("Checking if contract %s exists in Python service", contractID)
	resp, err := c.client.Get(fmt.Sprintf("%s/contracts/%s/price-update", c.baseURL, contractID))
	if err != nil {
		logging.DebugLog("Failed to get product: %v", err)
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	logging.DebugLog("Received response from Python service: %s", string(body))

	return resp.StatusCode == http.StatusOK, nil
}

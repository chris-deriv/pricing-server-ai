package contracts

import (
	"encoding/json"
	"pricingserver/internal/common/logging"
	"time"
)

// ContractProxy implements both Product and MessageSender interfaces
type ContractProxy struct {
	contractID    string
	client        *ContractServiceClient
	priceCallback func(price float64, timestamp time.Time)
	lastResponse  map[string]interface{}
	isActive      bool
	startTime     time.Time
}

// NewContractProxy creates a new proxy for a contract
func NewContractProxy(contractID string, _ interface{}, client *ContractServiceClient) *ContractProxy {
	logging.DebugLog("Creating new contract proxy for contract %s", contractID)
	return &ContractProxy{
		contractID: contractID,
		client:     client,
		isActive:   true,
		startTime:  time.Now(),
	}
}

// SendMessage implements MessageSender interface
func (cp *ContractProxy) SendMessage(message []byte) {
	logging.DebugLog("Contract %s received message: %s", cp.contractID, string(message))
	var update map[string]interface{}
	if err := json.Unmarshal(message, &update); err != nil {
		logging.DebugLog("Failed to unmarshal update message: %v", err)
		return
	}

	// Store the update in lastResponse
	cp.lastResponse = update
	logging.DebugLog("Contract %s stored update in lastResponse", cp.contractID)
}

// Init initializes the proxy (implements Product interface)
func (cp *ContractProxy) Init(params map[string]interface{}) error {
	// No local initialization needed as everything is handled by Python service
	return nil
}

// Start starts the proxy (implements Product interface)
func (cp *ContractProxy) Start() {
	logging.DebugLog("Starting contract proxy for contract %s", cp.contractID)
	cp.startTime = time.Now()
	cp.isActive = true
}

// Stop stops the proxy (implements Product interface)
func (cp *ContractProxy) Stop() {
	logging.DebugLog("Stopping contract proxy for contract %s", cp.contractID)
	cp.isActive = false
}

// HandlePriceUpdate forwards price updates to the Python service and processes the response
func (cp *ContractProxy) HandlePriceUpdate(price float64, timestamp time.Time) {
	logging.DebugLog("Contract %s handling price update: %f at %v", cp.contractID, price, timestamp)

	// Only forward updates if the contract is active
	if !cp.isActive {
		logging.DebugLog("Contract %s is inactive, skipping price update", cp.contractID)
		return
	}

	// Forward to Python service and get response directly
	resp, err := cp.client.UpdatePrice(cp.contractID, price)
	if err != nil {
		logging.DebugLog("Failed to forward price update to Python service: %v", err)
		return
	}

	logging.DebugLog("Contract %s received response from Python service: %s", cp.contractID, string(resp))

	// Parse the response
	var pythonResp map[string]interface{}
	if err := json.Unmarshal(resp, &pythonResp); err != nil {
		logging.DebugLog("Failed to unmarshal Python response: %v", err)
		return
	}

	// Add contract ID and timestamp to response if not present
	if _, ok := pythonResp["contractID"]; !ok {
		pythonResp["contractID"] = cp.contractID
	}
	if _, ok := pythonResp["timestamp"]; !ok {
		pythonResp["timestamp"] = timestamp.Format(time.RFC3339)
	}

	// Store the response
	cp.lastResponse = pythonResp
	logging.DebugLog("Contract %s stored Python response in lastResponse", cp.contractID)

	// Handle different status responses
	status, _ := pythonResp["status"].(string)
	logging.DebugLog("Contract %s status: %s", cp.contractID, status)

	// Create contract update message
	update := map[string]interface{}{
		"type": "ContractUpdate",
		"data": pythonResp,
	}

	// Marshal and send the update
	if updateBytes, err := json.Marshal(update); err == nil {
		if cp.priceCallback != nil {
			cp.priceCallback(price, timestamp)
		}
		cp.SendMessage(updateBytes)
	}

	// Handle status changes
	switch status {
	case "inactive", "expired", "target_hit":
		cp.Stop()
	}
}

// SetUpdateCallback sets the callback for price updates (implements Product interface)
func (cp *ContractProxy) SetUpdateCallback(callback func(price float64, timestamp time.Time)) {
	logging.DebugLog("Setting update callback for contract %s", cp.contractID)
	cp.priceCallback = callback
}

// CheckConditions checks conditions for the proxy (implements Product interface)
func (cp *ContractProxy) CheckConditions() {
	// No local conditions to check as everything is handled by Python service
}

// GetState gets the state of the proxy (implements Product interface)
func (cp *ContractProxy) GetState() map[string]interface{} {
	logging.DebugLog("Getting state for contract %s", cp.contractID)
	if cp.lastResponse != nil {
		logging.DebugLog("Returning lastResponse for contract %s: %+v", cp.contractID, cp.lastResponse)
		return cp.lastResponse
	}
	// Return a basic state if no response is available
	logging.DebugLog("No lastResponse available for contract %s, returning basic state", cp.contractID)
	return map[string]interface{}{
		"contractID": cp.contractID,
		"status":     "active",
		"timestamp":  time.Now().Format(time.RFC3339),
	}
}

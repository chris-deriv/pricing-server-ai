package server

import (
	"encoding/json"
	"fmt"
	"os"
	"pricingserver/internal/common/logging"
	"pricingserver/internal/contracts"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var debugLogging bool

func init() {
	debugLog := os.Getenv("DEBUG")
	if parsed, err := strconv.ParseBool(debugLog); err == nil {
		debugLogging = parsed
	}
}

// Message types
const (
	MessageTypeContractSubmission = "ContractSubmission"
	MessageTypeContractAccepted   = "ContractAccepted"
	MessageTypeContractUpdate     = "ContractUpdate"
	MessageTypeContractQuery      = "ContractQuery"
	MessageTypeError              = "Error"
)

// Error types
const (
	ErrorTypeValidation = "ValidationError"
	ErrorTypeParse      = "ParseError"
)

// Message structure
type Message struct {
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data,omitempty"`
	ContractID string          `json:"contractID,omitempty"`
}

// ContractData represents data required to create a contract
type ContractData struct {
	ProductType    string    `json:"productType"`
	Rungs          []float64 `json:"rungs,omitempty"`
	TargetMovement float64   `json:"targetMovement,omitempty"`
	Duration       int64     `json:"duration"` // milliseconds
	Payoff         float64   `json:"payoff"`
}

// ErrorResponse represents an error message
type ErrorResponse struct {
	Type      string `json:"type"`
	ErrorType string `json:"errorType"`
	Message   string `json:"message"`
}

// Client represents a connected client
type Client struct {
	ID        string
	Conn      *websocket.Conn
	Send      chan []byte
	Contracts map[string]string
	Hub       *Hub
	mu        sync.Mutex
}

// NewClient creates a new client instance
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:        GenerateUniqueID(),
		Hub:       hub,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		Contracts: make(map[string]string),
	}
}

// ReadPump handles incoming messages from the client
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			logging.DebugLog("ReadPump error: %v", err)
			break
		}

		// Try to parse as JSON first
		if !json.Valid(message) {
			logging.DebugLog("Invalid JSON received")
			c.sendError(ErrorTypeParse, "Invalid JSON format")
			continue
		}

		c.handleMessage(message)
	}
}

// handleMessage processes messages from the client
func (c *Client) handleMessage(message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		logging.DebugLog("Failed to unmarshal message: %v", err)
		c.sendError(ErrorTypeParse, "Invalid message format")
		return
	}

	if msg.Type == "" {
		logging.DebugLog("Missing message type")
		c.sendError(ErrorTypeValidation, "Message type is required")
		return
	}

	logging.DebugLog("Received message type: %s", msg.Type)

	switch msg.Type {
	case MessageTypeContractSubmission:
		if msg.Data == nil {
			logging.DebugLog("Missing data field in contract submission")
			c.sendError(ErrorTypeValidation, "Data field is required for contract submission")
			return
		}
		c.handleContractSubmission(msg.Data)
	case MessageTypeContractQuery:
		if msg.ContractID == "" {
			logging.DebugLog("Missing contractID in contract query")
			c.sendError(ErrorTypeValidation, "ContractID is required for contract query")
			return
		}
		logging.DebugLog("Querying contract: %s", msg.ContractID)
		c.handleContractQuery(msg.ContractID)
	default:
		logging.DebugLog("Unknown message type: %s", msg.Type)
		c.sendError(ErrorTypeValidation, fmt.Sprintf("Unknown message type: %s", msg.Type))
	}
}

// handleContractQuery processes contract query requests
func (c *Client) handleContractQuery(contractID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logging.DebugLog("Getting contract state for: %s", contractID)

	// Get contract state from service
	state, err := c.Hub.ContractService.GetContractState(contractID)
	if err != nil {
		logging.DebugLog("Failed to get contract state: %v", err)
		c.sendError(ErrorTypeValidation, fmt.Sprintf("Failed to get contract state: %v", err))
		return
	}

	// If contract exists, send its state
	if state != nil {
		logging.DebugLog("Got contract state: %+v", state)
		update := map[string]interface{}{
			"type":       MessageTypeContractUpdate,
			"contractID": contractID,
			"data":       state,
		}
		logging.DebugLog("Sending contract update: %+v", update)
		c.sendMessage(update)
	} else {
		logging.DebugLog("Contract not found: %s", contractID)
		c.sendError(ErrorTypeValidation, fmt.Sprintf("Contract not found: %s", contractID))
	}
}

// validateContractData validates the contract data
func (c *Client) validateContractData(data *ContractData) error {
	if data.ProductType == "" {
		return fmt.Errorf("productType is required")
	}

	if data.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}

	if data.Payoff <= 0 {
		return fmt.Errorf("payoff must be positive")
	}

	switch data.ProductType {
	case "LuckyLadder":
		if len(data.Rungs) == 0 {
			return fmt.Errorf("rungs are required for LuckyLadder")
		}

		// Check for duplicates first
		seen := make(map[float64]bool)
		for _, rung := range data.Rungs {
			if seen[rung] {
				return fmt.Errorf("duplicate rung values are not allowed")
			}
			seen[rung] = true
		}

		// Then check for ascending order
		for i := 1; i < len(data.Rungs); i++ {
			if data.Rungs[i] <= data.Rungs[i-1] {
				return fmt.Errorf("rungs must be in ascending order")
			}
		}

	case "MomentumCatcher":
		if data.TargetMovement <= 0 {
			return fmt.Errorf("targetMovement must be positive")
		}

	default:
		return fmt.Errorf("unsupported product type: %s", data.ProductType)
	}

	return nil
}

// handleContractSubmission processes contract submission requests
func (c *Client) handleContractSubmission(data json.RawMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var contractData ContractData
	if err := json.Unmarshal(data, &contractData); err != nil {
		logging.DebugLog("Failed to unmarshal contract data: %v", err)
		c.sendError(ErrorTypeParse, "Invalid contract data format")
		return
	}

	if err := c.validateContractData(&contractData); err != nil {
		logging.DebugLog("Contract validation failed: %v", err)
		c.sendError(ErrorTypeValidation, err.Error())
		return
	}

	contractID := GenerateUniqueID()
	logging.DebugLog("Creating new contract with ID: %s", contractID)

	// Create contract parameters for Python service
	var contractParams contracts.ContractParams
	parameters := map[string]interface{}{
		"duration": contractData.Duration,
		"payoff":   contractData.Payoff,
	}

	switch contractData.ProductType {
	case "LuckyLadder":
		contractParams = contracts.ContractParams{
			ContractType: "lucky_ladder",
			Parameters:   parameters,
		}
		contractParams.Parameters["rungs"] = contractData.Rungs
	case "MomentumCatcher":
		contractParams = contracts.ContractParams{
			ContractType: "momentum_catcher",
			Parameters:   parameters,
		}
		contractParams.Parameters["target_movement"] = contractData.TargetMovement
	}

	// Create a proxy for the contract
	proxy := contracts.NewContractProxy(contractID, nil, c.Hub.ContractService)

	// Set up a callback to handle Python service responses
	proxy.SetUpdateCallback(func(price float64, timestamp time.Time) {
		state := proxy.GetState()
		logging.DebugLog("Got state from proxy: %+v", state)

		update := map[string]interface{}{
			"type":       MessageTypeContractUpdate,
			"contractID": contractID,
			"data":       state,
		}

		if updateBytes, err := json.Marshal(update); err == nil {
			c.Send <- updateBytes
		} else {
			logging.DebugLog("Failed to marshal contract update: %v", err)
		}

		if status, ok := state["status"].(string); ok && (status == "inactive" || status == "expired" || status == "target_hit") {
			logging.DebugLog("Contract %s is no longer active (status: %s), unsubscribing", contractID, status)
			c.Hub.SimulationEngine.Unsubscribe(contractID)
			delete(c.Contracts, contractID)
		}
	})

	// Forward to Python service and subscribe to updates
	if err := c.Hub.ContractService.AddContract(contractID, contractParams); err != nil {
		logging.DebugLog("Failed to add contract to service: %v", err)
		c.sendError(ErrorTypeValidation, fmt.Sprintf("Failed to create contract: %v", err))
		return
	}

	c.Hub.SimulationEngine.Subscribe(contractID, proxy)
	proxy.Start()

	c.Contracts[contractID] = contractData.ProductType

	// Send confirmation
	c.sendMessage(map[string]interface{}{
		"type":       MessageTypeContractAccepted,
		"contractID": contractID,
	})
}

// sendError sends an error message to the client
func (c *Client) sendError(errorType string, message string) {
	response := ErrorResponse{
		Type:      MessageTypeError,
		ErrorType: errorType,
		Message:   message,
	}
	c.sendMessage(response)
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(data interface{}) {
	message, err := json.Marshal(data)
	if err != nil {
		logging.DebugLog("Failed to marshal message: %v", err)
		return
	}

	logging.DebugLog("Sending message: %s", string(message))
	c.Send <- message
}

// WritePump handles sending messages to the client
func (c *Client) WritePump() {
	ticker := time.NewTicker(time.Second * 54)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logging.DebugLog("Error writing message: %v", err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

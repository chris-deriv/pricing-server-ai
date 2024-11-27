package server

import (
	"encoding/json"
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
	MessageTypeError              = "Error"
)

// Message structure
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// ContractData represents data required to create a contract
type ContractData struct {
	ProductType    string    `json:"productType"`
	Rungs          []float64 `json:"rungs"`
	TargetMovement float64   `json:"targetMovement"`
	Duration       int64     `json:"duration"` // milliseconds
	Payoff         float64   `json:"payoff"`
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

// SendMessage implements MessageSender interface
func (c *Client) SendMessage(message []byte) {
	logging.DebugLog("Client SendMessage called with message: %s", string(message))
	select {
	case c.Send <- message:
		logging.DebugLog("Message queued successfully")
	default:
		logging.DebugLog("Warning: Send channel full, message dropped")
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
			logging.DebugLog("ReadPump error:", err)
			break
		}
		c.handleMessage(message)
	}
}

// handleMessage processes messages from the client
func (c *Client) handleMessage(message []byte) {
	var msg Message
	err := json.Unmarshal(message, &msg)
	if err != nil {
		logging.DebugLog("Invalid message format:", err)
		return
	}

	switch msg.Type {
	case MessageTypeContractSubmission:
		c.handleContractSubmission(msg.Data)
	default:
		logging.DebugLog("Unknown message type:", msg.Type)
	}
}

// handleContractSubmission processes contract submission requests
func (c *Client) handleContractSubmission(data json.RawMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var contractData ContractData
	err := json.Unmarshal(data, &contractData)
	if err != nil {
		logging.DebugLog("Invalid contract data:", err)
		return
	}

	contractID := GenerateUniqueID()
	logging.DebugLog("Creating new contract with ID: %s", contractID)
	logging.DebugLog("Contract duration: %d ms", contractData.Duration)

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
	default:
		logging.DebugLog("Unsupported product type:", contractData.ProductType)
		return
	}

	// Create a proxy for the contract
	proxy := contracts.NewContractProxy(contractID, nil, c.Hub.ContractService)

	// Set up a callback to handle Python service responses
	proxy.SetUpdateCallback(func(price float64, timestamp time.Time) {
		logging.DebugLog("Price update callback triggered with price: %f", price)

		// Get the current state from the proxy
		state := proxy.GetState()
		logging.DebugLog("Got state from proxy: %+v", state)

		// Create contract update message
		update := map[string]interface{}{
			"type": "ContractUpdate",
			"data": state,
		}

		// Marshal and send the update
		if updateBytes, err := json.Marshal(update); err == nil {
			logging.DebugLog("Sending contract update: %s", string(updateBytes))
			select {
			case c.Send <- updateBytes:
				logging.DebugLog("Contract update queued successfully")
			default:
				logging.DebugLog("Warning: Send channel full, dropped contract update")
			}
		} else {
			logging.DebugLog("Failed to marshal contract update: %v", err)
		}

		// If the contract is no longer active, unsubscribe from updates
		if status, ok := state["status"].(string); ok && (status == "inactive" || status == "expired" || status == "target_hit") {
			logging.DebugLog("Contract %s is no longer active (status: %s), unsubscribing", contractID, status)
			c.Hub.SimulationEngine.Unsubscribe(contractID)
			delete(c.Contracts, contractID)
		}
	})

	// Forward to Python service and subscribe to updates
	err = c.Hub.ContractService.AddContract(contractID, contractParams)
	if err != nil {
		logging.DebugLog("Failed to add contract to service: %v", err)
		return
	}

	logging.DebugLog("Subscribing contract %s to simulation engine", contractID)
	c.Hub.SimulationEngine.Subscribe(contractID, proxy)
	proxy.Start()

	c.Contracts[contractID] = contractData.ProductType

	// Send confirmation
	response := map[string]interface{}{
		"type":       MessageTypeContractAccepted,
		"contractID": contractID,
	}
	c.sendMessage(response)
}

// Helper function to marshal JSON
func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err) // In production code, you might want to handle this differently
	}
	return data
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(data interface{}) {
	message, err := json.Marshal(data)
	if err != nil {
		logging.DebugLog("Failed to marshal message:", err)
		return
	}

	logging.DebugLog("Sending message: %s", string(message))
	select {
	case c.Send <- message:
		logging.DebugLog("Message queued successfully")
	default:
		logging.DebugLog("Warning: Send channel full, message dropped")
	}
}

// WritePump handles sending messages to the client
func (c *Client) WritePump() {
	ticker := time.NewTicker(time.Second * 54) // Ping timer
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

			logging.DebugLog("WritePump sending message: %s", string(message))
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logging.DebugLog("Error writing message: %v", err)
				return
			}
			logging.DebugLog("Message sent successfully through websocket")
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

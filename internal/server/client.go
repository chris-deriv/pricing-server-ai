package server

import (
    "encoding/json"
    "fmt"
    "os"
    "strconv"
    "sync"
    "time"
    "github.com/gorilla/websocket"
    // "pricingserver/internal/contracts"
    "pricingserver/internal/products"
    // "pricingserver/internal/simulation"
    "pricingserver/internal/common/logging"
)

// Add this at package level
var debugLogging bool

func init() {
    // Check for DEBUG environment variable, default to false
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
    ProductType     string        `json:"productType"`
    Rungs          []float64     `json:"rungs"`
    TargetMovement float64       `json:"targetMovement"`
    Duration       time.Duration `json:"duration"`
    Payoff         float64       `json:"payoff"`
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

// Add this new type for price updates
type PriceUpdate struct {
    ContractID string    `json:"contractID"`
    Price      float64   `json:"price"`
    Timestamp  time.Time `json:"timestamp"`
}

// Add this method to the Client struct to implement MessageSender
func (c *Client) SendMessage(message []byte) {
    select {
    case c.Send <- message:
        logging.DebugLog("Message queued successfully")
    default:
        logging.DebugLog("Warning: Send channel full, message dropped")
    }
}

// Update the handleContractSubmission method to ensure Client is passed correctly
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
    
    // Create a closure to capture the contractID
    updateCallback := func(price float64, timestamp time.Time) {
        update := PriceUpdate{
            ContractID: contractID,
            Price:     price,
            Timestamp: timestamp,
        }
        response := Message{
            Type: MessageTypeContractUpdate,
            Data: json.RawMessage(mustMarshal(update)),
        }
        
        // Use select to prevent blocking
        select {
        case c.Send <- mustMarshal(response):
            logging.DebugLog("Price update sent for contract %s: %f", contractID, price)
        default:
            logging.DebugLog("Warning: Channel full, dropped price update for contract %s", contractID)
        }
    }

    // Initialize product with callback
    product, err := c.initializeProduct(contractData, contractID, updateCallback)
    if err != nil {
        logging.DebugLog("Failed to initialize product: %v", err)
        return
    }

    // Start the product and subscribe to updates
    c.Hub.ContractManager.AddContract(contractID, product)
    c.Hub.SimulationEngine.Subscribe(contractID, product)
    product.Start()
    
    c.Contracts[contractID] = contractData.ProductType

    // Send confirmation
    response := map[string]interface{}{
        "type":       MessageTypeContractAccepted,
        "contractID": contractID,
    }
    c.sendMessage(response)
}

// Helper method to initialize products
func (c *Client) initializeProduct(contractData ContractData, contractID string, callback func(float64, time.Time)) (products.Product, error) {
    params := map[string]interface{}{
        "ClientID":   c.ID,
        "ContractID": contractID,
        "Duration":   contractData.Duration,
        "Payoff":     contractData.Payoff,
    }

    var product products.Product
    switch contractData.ProductType {
    case "LuckyLadder":
        ll := &products.LuckyLadder{}
        params["Rungs"] = contractData.Rungs
        err := ll.Init(params)
        if err != nil {
            logging.DebugLog("Failed to initialize LuckyLadder:", err)
            return nil, err
        }
        ll.SetUpdateCallback(callback)
        product = ll
    case "MomentumCatcher":
        mc := &products.MomentumCatcher{}
        params["TargetMovement"] = contractData.TargetMovement
        params["Client"] = c
        err := mc.Init(params)
        if err != nil {
            logging.DebugLog("Failed to initialize MomentumCatcher:", err)
            return nil, err
        }
        mc.SetUpdateCallback(callback)
        product = mc
    default:
        logging.DebugLog("Unsupported product type:", contractData.ProductType)
        return nil, fmt.Errorf("unsupported product type: %s", contractData.ProductType)
    }

    return product, nil
}

// Helper function to marshal JSON (add this)
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

    // Use select for non-blocking send
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

            c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
                logging.DebugLog("Error writing message: %v", err)
                return
            }
            logging.DebugLog("Message sent successfully")
        case <-ticker.C:
            c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
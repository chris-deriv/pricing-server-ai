package products

import (
    "encoding/json"
    "time"
    "fmt"
    "pricingserver/internal/common/logging"
)

// MessageSender interface for sending messages
type MessageSender interface {
    SendMessage([]byte)
}

// MomentumCatcher represents the Momentum Catcher product
type MomentumCatcher struct {
    ClientID          string
    ContractID        string
    StartingPrice     *float64
    TargetMovement    float64
    StartTime         time.Time
    Duration          time.Duration
    ExpiryTime        time.Time
    IsActive          bool
    HasReachedTarget  bool
    Payoff            float64
    PriceHistory      []float64
    Client            MessageSender
    updateCallback    func(price float64, timestamp time.Time)
}

// Ensure MomentumCatcher implements the Product interface
var _ Product = (*MomentumCatcher)(nil)

// Init initializes the Momentum Catcher product
func (mc *MomentumCatcher) Init(params map[string]interface{}) error {
    var ok bool
    mc.ClientID, ok = params["ClientID"].(string)
    if !ok {
        logging.DebugLog("Error: ClientID is required and must be a string")
        return fmt.Errorf("ClientID is required and must be a string")
    }
    mc.ContractID, ok = params["ContractID"].(string)
    if !ok {
        logging.DebugLog("Error: ContractID is required and must be a string")
        return fmt.Errorf("ContractID is required and must be a string")
    }
    mc.TargetMovement, ok = params["TargetMovement"].(float64)
    if !ok {
        logging.DebugLog("Error: TargetMovement is required and must be a float64")
        return fmt.Errorf("TargetMovement is required and must be a float64")
    }
    mc.Duration, ok = params["Duration"].(time.Duration)
    if !ok {
        logging.DebugLog("Error: Duration is required and must be a time.Duration")
        return fmt.Errorf("Duration is required and must be a time.Duration")
    }
    mc.Payoff, ok = params["Payoff"].(float64)
    if !ok {
        logging.DebugLog("Error: Payoff is required and must be a float64")
        return fmt.Errorf("Payoff is required and must be a float64")
    }
    mc.Client, ok = params["Client"].(MessageSender)
    if !ok {
        logging.DebugLog("Error: Client is required and must be of type MessageSender")
        return fmt.Errorf("Client is required and must be of type MessageSender")
    }
    mc.StartTime = time.Now()
    mc.ExpiryTime = mc.StartTime.Add(mc.Duration)
    mc.IsActive = true
    mc.PriceHistory = []float64{}
    return nil
}

// Start begins any necessary processing
func (mc *MomentumCatcher) Start() {
    // No additional processing needed
}

// Stop ends the product's activity
func (mc *MomentumCatcher) Stop() {
    mc.IsActive = false
}

// HandlePriceUpdate processes incoming price updates
func (mc *MomentumCatcher) HandlePriceUpdate(price float64, timestamp time.Time) {
    logging.DebugLog("MomentumCatcher received price update: %f", price)
    
    // Always call the callback first, like LuckyLadder does
    if mc.updateCallback != nil {
        mc.updateCallback(price, timestamp)
    }

    if !mc.IsActive {
        return
    }

    // Set the StartingPrice on the first price update
    if mc.StartingPrice == nil {
        mc.StartingPrice = &price
    }

    // Record the price in history
    mc.PriceHistory = append(mc.PriceHistory, price)

    // Calculate the absolute price movement
    priceMovement := abs(price - *mc.StartingPrice)

    if priceMovement >= mc.TargetMovement {
        mc.HasReachedTarget = true
        mc.IsActive = false
        // Notify the client about the success
        mc.notifyClient("TargetReached", price, timestamp)
        return
    }

    // Check if the contract has expired
    if timestamp.After(mc.ExpiryTime) {
        mc.IsActive = false
        // Notify the client about the expiration
        mc.notifyClient("Expired", price, timestamp)
    }
}

// CheckConditions checks if the product conditions have been met
func (mc *MomentumCatcher) CheckConditions() {
    // Conditions are checked in HandlePriceUpdate
}

// GetState returns the current state of the product
func (mc *MomentumCatcher) GetState() map[string]interface{} {
    var startingPrice float64
    if mc.StartingPrice != nil {
        startingPrice = *mc.StartingPrice
    }
    return map[string]interface{}{
        "contractID":       mc.ContractID,
        "isActive":         mc.IsActive,
        "hasReachedTarget": mc.HasReachedTarget,
        "startingPrice":    startingPrice,
        "targetMovement":   mc.TargetMovement,
        "startTime":        mc.StartTime.Format(time.RFC3339),
        "expiryTime":       mc.ExpiryTime.Format(time.RFC3339),
        "priceHistory":     mc.PriceHistory,
        "payoff":           mc.Payoff,
    }
}

// notifyClient sends a contract update to the client
func (mc *MomentumCatcher) notifyClient(status string, price float64, timestamp time.Time) {
    update := map[string]interface{}{
        "type":       "ContractUpdate",
        "contractID": mc.ContractID,
        "status":     status,
        "payoff":     mc.Payoff,
        "price":      price,
        "timestamp":  timestamp.UnixMilli(),
    }
    message, err := json.Marshal(update)
    if err != nil {
        // Handle error (optional logging)
        return
    }
    mc.Client.SendMessage(message)
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
    if x < 0 {
        return -x
    }
    return x
}

// Add this method to MomentumCatcher
func (mc *MomentumCatcher) SetUpdateCallback(callback func(price float64, timestamp time.Time)) {
    mc.updateCallback = callback
}
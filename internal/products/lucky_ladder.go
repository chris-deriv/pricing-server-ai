package products

import (
    "time"
    "pricingserver/internal/common/logging"
)

// LuckyLadder represents the Lucky Ladder product
type LuckyLadder struct {
    ClientID     string
    ContractID   string
    Rungs        []float64
    StartTime    time.Time
    Duration     time.Duration
    ExpiryTime   time.Time
    IsActive     bool
    HasHitRung   bool
    HitRungs     map[float64]bool
    Payoff       float64
    PriceHistory []float64 // For tracking price movements
    updateCallback func(price float64, timestamp time.Time)
}

// Ensure LuckyLadder implements the Product interface
var _ Product = (*LuckyLadder)(nil)

// Init initializes the Lucky Ladder product
func (ll *LuckyLadder) Init(params map[string]interface{}) error {
    // Extract parameters from the map
    ll.ClientID = params["ClientID"].(string)
    ll.ContractID = params["ContractID"].(string)
    ll.Rungs = params["Rungs"].([]float64)
    ll.StartTime = time.Now()
    ll.Duration = params["Duration"].(time.Duration)
    ll.ExpiryTime = ll.StartTime.Add(ll.Duration)
    ll.IsActive = true
    ll.HasHitRung = false
    ll.HitRungs = make(map[float64]bool)
    ll.Payoff = params["Payoff"].(float64)
    ll.PriceHistory = []float64{}

    // Initialize HitRungs map
    for _, rung := range ll.Rungs {
        ll.HitRungs[rung] = false
    }
    return nil
}

// Start begins any necessary processing (if needed)
func (ll *LuckyLadder) Start() {
    logging.DebugLog("LuckyLadder starting")
}

// Stop ends the product's activity
func (ll *LuckyLadder) Stop() {
    logging.DebugLog("LuckyLadder stopping")
    ll.IsActive = false
}

// HandlePriceUpdate processes incoming price updates
func (ll *LuckyLadder) HandlePriceUpdate(price float64, timestamp time.Time) {
    logging.DebugLog("LuckyLadder HandlePriceUpdate called with price: %f", price)
    if ll.updateCallback == nil {
        logging.DebugLog("WARNING: Callback is nil!")
        return
    }
    ll.updateCallback(price, timestamp)
}

// CheckConditions checks if the product conditions have been met
func (ll *LuckyLadder) CheckConditions() {
    // In this case, conditions are checked during price updates
    // Additional periodic checks can be added if needed
}

// GetState returns the current state to send to the client
func (ll *LuckyLadder) GetState() map[string]interface{} {
    return map[string]interface{}{
        "ContractID":   ll.ContractID,
        "IsActive":     ll.IsActive,
        "HasHitRung":   ll.HasHitRung,
        "HitRungs":     ll.HitRungs,
        "ExpiryTime":   ll.ExpiryTime,
        "Payoff":       ll.Payoff,
        "PriceHistory": ll.PriceHistory,
    }
}

func (ll *LuckyLadder) SetUpdateCallback(callback func(price float64, timestamp time.Time)) {
    logging.DebugLog("Setting callback on LuckyLadder")
    ll.updateCallback = callback
}
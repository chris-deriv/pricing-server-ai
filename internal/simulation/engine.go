package simulation

import (
    "math"
    "math/rand"
    "sync"
    "time"
    
    "pricingserver/internal/products"
    "pricingserver/internal/common/logging"
)

// SimulationEngine generates simulated price data
type SimulationEngine struct {
    subscribers map[string]products.Product // Maps contract IDs to products
    mu          sync.Mutex
    ticker      *time.Ticker
    stopChan    chan bool
    // BasePrice allows products to set a starting price if needed
    BasePrice   float64
}

// NewSimulationEngine creates a new simulation engine
func NewSimulationEngine() *SimulationEngine {
    return &SimulationEngine{
        subscribers: make(map[string]products.Product),
        stopChan:    make(chan bool),
        BasePrice:   100.0, // Set a default base price
    }
}

// Start begins the simulation
func (se *SimulationEngine) Start() {
    se.ticker = time.NewTicker(100 * time.Millisecond) // Adjust the interval as needed

    go func() {
        for {
            select {
            case <-se.ticker.C:
                price := se.generatePrice()
                timestamp := time.Now()
                se.notifySubscribers(price, timestamp)
            case <-se.stopChan:
                se.ticker.Stop()
                return
            }
        }
    }()
}

// Stop ends the simulation
func (se *SimulationEngine) Stop() {
    se.stopChan <- true
}

// Subscribe adds a product to receive price updates
func (se *SimulationEngine) Subscribe(contractID string, product products.Product) {
    se.mu.Lock()
    defer se.mu.Unlock()
    logging.DebugLog("Adding subscription for contract %s", contractID)
    se.subscribers[contractID] = product
}

// Unsubscribe removes a product from receiving price updates
func (se *SimulationEngine) Unsubscribe(contractID string) {
    se.mu.Lock()
    defer se.mu.Unlock()
    delete(se.subscribers, contractID)
}

// notifySubscribers sends price updates to all subscribers
func (se *SimulationEngine) notifySubscribers(price float64, timestamp time.Time) {
    se.mu.Lock()
    defer se.mu.Unlock()
    logging.DebugLog("SimulationEngine generating price: %f", price)
    logging.DebugLog("Number of subscribers: %d", len(se.subscribers))
    for contractID, product := range se.subscribers {
        logging.DebugLog("Notifying contract %s", contractID)
        product.HandlePriceUpdate(price, timestamp)
    }
}

// generatePrice generates a simulated price
func (se *SimulationEngine) generatePrice() float64 {
    // Implement a stochastic process, e.g., Geometric Brownian Motion

    // Parameters for Geometric Brownian Motion
    mu := 0.0002     // Drift coefficient
    sigma := 0.01    // Volatility coefficient
    dt := 0.1        // Time step

    // Generate a random number from standard normal distribution
    epsilon := rand.NormFloat64()

    // Update the base price
    se.BasePrice = se.BasePrice * math.Exp((mu-(0.5*math.Pow(sigma, 2)))*dt + sigma*epsilon*math.Sqrt(dt))

    return se.BasePrice
}
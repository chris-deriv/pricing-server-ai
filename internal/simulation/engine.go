package simulation

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"pricingserver/internal/common/logging"
)

// PriceHandler represents any type that can handle price updates
type PriceHandler interface {
	HandlePriceUpdate(price float64, timestamp time.Time)
}

// SimulationEngine generates simulated price data
type SimulationEngine struct {
	subscribers map[string]PriceHandler // Maps contract IDs to price handlers
	mu          sync.Mutex
	ticker      *time.Ticker
	stopChan    chan bool
	// BasePrice allows products to set a starting price if needed
	BasePrice float64
}

// NewSimulationEngine creates a new simulation engine
func NewSimulationEngine() *SimulationEngine {
	return &SimulationEngine{
		subscribers: make(map[string]PriceHandler),
		stopChan:    make(chan bool),
		BasePrice:   100.0, // Set a default base price
	}
}

// Start begins the simulation
func (se *SimulationEngine) Start() {
	logging.DebugLog("Starting simulation engine")
	se.ticker = time.NewTicker(100 * time.Millisecond) // Adjust the interval as needed

	go func() {
		for {
			select {
			case <-se.ticker.C:
				se.mu.Lock()
				subscriberCount := len(se.subscribers)
				if subscriberCount > 0 {
					price := se.generatePrice()
					timestamp := time.Now()
					logging.DebugLog("Generated new price: %f at %v with %d subscribers", price, timestamp, subscriberCount)
					// Notify each subscriber independently
					for contractID, handler := range se.subscribers {
						go func(id string, h PriceHandler, p float64, t time.Time) {
							logging.DebugLog("Notifying contract %s of price update: %f at %v", id, p, t)
							h.HandlePriceUpdate(p, t)
						}(contractID, handler, price, timestamp)
					}
				}
				se.mu.Unlock()
			case <-se.stopChan:
				logging.DebugLog("Stopping simulation engine")
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

// Subscribe adds a handler to receive price updates
func (se *SimulationEngine) Subscribe(contractID string, handler PriceHandler) {
	se.mu.Lock()
	defer se.mu.Unlock()
	logging.DebugLog("Adding subscription for contract %s", contractID)
	se.subscribers[contractID] = handler
	logging.DebugLog("Current number of subscribers: %d", len(se.subscribers))

	// Send initial price update immediately
	timestamp := time.Now()
	logging.DebugLog("Sending initial price update to contract %s: %f at %v", contractID, se.BasePrice, timestamp)
	go handler.HandlePriceUpdate(se.BasePrice, timestamp)
}

// Unsubscribe removes a handler from receiving price updates
func (se *SimulationEngine) Unsubscribe(contractID string) {
	se.mu.Lock()
	defer se.mu.Unlock()
	logging.DebugLog("Removing subscription for contract %s", contractID)
	delete(se.subscribers, contractID)
	logging.DebugLog("Current number of subscribers: %d", len(se.subscribers))
}

// generatePrice generates a simulated price
func (se *SimulationEngine) generatePrice() float64 {
	// Implement a stochastic process, e.g., Geometric Brownian Motion

	// Parameters for Geometric Brownian Motion
	mu := 0.0002  // Drift coefficient
	sigma := 0.01 // Volatility coefficient
	dt := 0.1     // Time step

	// Generate a random number from standard normal distribution
	epsilon := rand.NormFloat64()

	// Update the base price
	se.BasePrice = se.BasePrice * math.Exp((mu-(0.5*math.Pow(sigma, 2)))*dt+sigma*epsilon*math.Sqrt(dt))

	return se.BasePrice
}

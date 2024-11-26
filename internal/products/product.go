package products

import "time"

// Product interface that all products must implement
type Product interface {
    // Initialize the product with necessary parameters
    Init(params map[string]interface{}) error
    // Start the simulation or processing for the product
    Start()
    // Stop the simulation or processing
    Stop()
    // Handle price updates from the simulation engine
    HandlePriceUpdate(price float64, timestamp time.Time)
    // Set update callback for price updates
    SetUpdateCallback(callback func(price float64, timestamp time.Time))
    // Check if the product's conditions have been met (e.g., barrier hit)
    CheckConditions()
    // Get the current state or result to send to the client
    GetState() map[string]interface{}
}
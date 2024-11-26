package contracts

import (
    "sync"

    "pricingserver/internal/products"
)

// ContractManager handles contracts and associated products
type ContractManager struct {
    contracts map[string]products.Product
    mu        sync.Mutex
}

// NewContractManager creates a new ContractManager
func NewContractManager() *ContractManager {
    return &ContractManager{
        contracts: make(map[string]products.Product),
    }
}

// AddContract adds a new contract and product
func (cm *ContractManager) AddContract(contractID string, product products.Product) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    cm.contracts[contractID] = product
}

// RemoveContract removes a contract and associated product
func (cm *ContractManager) RemoveContract(contractID string) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    delete(cm.contracts, contractID)
}

// GetProduct retrieves a product by contract ID
func (cm *ContractManager) GetProduct(contractID string) (products.Product, bool) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    product, exists := cm.contracts[contractID]
    return product, exists
}
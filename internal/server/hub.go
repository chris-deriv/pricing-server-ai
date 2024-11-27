package server

import (
	"sync"

	"pricingserver/internal/common/logging"
	"pricingserver/internal/contracts"
	"pricingserver/internal/simulation"
)

// Hub maintains active clients and coordinates communication
type Hub struct {
	Clients          map[*Client]bool
	Register         chan *Client
	Unregister       chan *Client
	Broadcast        chan []byte
	mu               sync.Mutex
	ContractService  *contracts.ContractServiceClient
	SimulationEngine *simulation.SimulationEngine
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		Clients:          make(map[*Client]bool),
		Register:         make(chan *Client),
		Unregister:       make(chan *Client),
		Broadcast:        make(chan []byte),
		ContractService:  contracts.NewContractServiceClient(),
		SimulationEngine: simulation.NewSimulationEngine(),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	// Start the simulation engine
	h.SimulationEngine.Start()

	// Create a proxy for each active contract
	if activeContracts, err := h.ContractService.GetActiveContracts(); err == nil {
		for _, contractID := range activeContracts {
			logging.DebugLog("Restoring contract: %s", contractID)
			state, err := h.ContractService.GetContractState(contractID)
			if err != nil {
				logging.DebugLog("Failed to get contract state: %v", err)
				continue
			}
			if state == nil {
				logging.DebugLog("Contract state not found: %s", contractID)
				continue
			}
			if status, ok := state["status"].(string); !ok || status != "active" {
				logging.DebugLog("Contract is not active: %s, status: %s", contractID, status)
				continue
			}

			proxy := contracts.NewContractProxy(contractID, nil, h.ContractService)
			h.SimulationEngine.Subscribe(contractID, proxy)
			proxy.Start()
			logging.DebugLog("Restored active contract: %s", contractID)
		}
	} else {
		logging.DebugLog("Failed to get active contracts: %v", err)
	}

	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				// Unsubscribe client's products from the simulation engine
				for contractID := range client.Contracts {
					h.SimulationEngine.Unsubscribe(contractID)
					h.ContractService.RemoveContract(contractID)
				}
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			h.mu.Lock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

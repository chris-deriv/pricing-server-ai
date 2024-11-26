package server

import (
    "sync"

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
    ContractManager  *contracts.ContractManager
    SimulationEngine *simulation.SimulationEngine
}

// NewHub creates a new Hub
func NewHub() *Hub {
    return &Hub{
        Clients:          make(map[*Client]bool),
        Register:         make(chan *Client),
        Unregister:       make(chan *Client),
        Broadcast:        make(chan []byte),
        ContractManager:  contracts.NewContractManager(),
        SimulationEngine: simulation.NewSimulationEngine(),
    }
}

// Run starts the hub's main loop
func (h *Hub) Run() {
    // Start the simulation engine
    h.SimulationEngine.Start()
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
                    h.ContractManager.RemoveContract(contractID)
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
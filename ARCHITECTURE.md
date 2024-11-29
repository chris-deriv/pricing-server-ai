# Pricing Server Architecture

The pricing server is a distributed system designed to manage and simulate financial contracts with real-time price updates. The system consists of several key components implemented in different programming languages to leverage their specific strengths:

- WebSocket Server Hub: Implemented in Go for high-performance concurrent WebSocket connections
- Contracts Service: Written in Python for flexible business logic and rapid development
- Storage Service: Built in Go for efficient data handling and concurrent database operations
- Simulation Engine: Developed in Go for high-performance price simulations

## System Overview

The system employs a polyglot microservices architecture where each service is implemented in the language best suited for its purpose. The Go-based components (WebSocket Server, Storage Service, and Simulation Engine) handle high-performance operations and concurrent processing, while the Python-based Contracts Service manages complex business logic and contract calculations.

```
                                    WebSocket
                                   Connection
┌──────────┐                    ┌───────-────────┐
│  Client  │◄────WebSocket─────►│                │
└──────────┘                    │   WebSocket    │         ┌─────────────────┐
                                │    Server      │         │   Simulation    │
┌──────────┐                    │     Hub        │◄───────►│    Engine       │
│  Client  │◄────WebSocket─────►│     [Go]       │         │     [Go]        │
└──────────┘                    │                │         │                 │
                                │                │         └─────────────────┘
┌──────────┐                    │                │
│  Client  │◄────WebSocket─────►│                │
└──────────┘                    └──────┬─────────┘
                                      │
                                      │
                                      │
                                ┌─────▼─────┐         ┌─────────────────┐
                                │           │   HTTP  │                 │
                                │ Contracts │◄───────►│    Storage      │
                                │ Service   │   REST  │    Service      │
                                │ [Python]  │         │     [Go]        │
                                └───────────┘         └─────────────────┘
                                                           │
                                                           │
                                                    ┌──────▼──────┐
                                                    │             │
                                                    │ PostgreSQL  │
                                                    │             │
                                                    └─────────────┘
```

## Core Components

### WebSocket Server Hub [Go]

The Hub is the central communication component, implemented in Go to handle high-performance WebSocket connections efficiently. Go's strong concurrency support through goroutines and channels makes it ideal for managing multiple client connections simultaneously. This component:
- Manages active WebSocket client connections
- Coordinates message broadcasting between clients
- Handles client registration and disconnection
- Maintains the lifecycle of contracts
- Integrates with the Simulation Engine and Contract Service

Communication:
- With Clients:
  - WebSocket protocol (ws://) for real-time bidirectional communication
  - JSON-formatted messages for contract operations and price updates
  - Client registration/deregistration through WebSocket connection lifecycle
- With Contracts Service:
  - HTTP/REST communication
  - Endpoints used:
    - POST /contracts - Create new contracts
    - GET /contracts/active - Retrieve active contracts
    - POST /contracts/{id}/price-update - Send price updates
    - GET /contracts/{id}/state - Get contract state
    - DELETE /contracts/{id} - Remove contracts
- With Simulation Engine:
  - In-memory communication (same process)
  - Price updates through the PriceHandler interface
  - Subscription management through Subscribe/Unsubscribe methods

### Simulation Engine [Go]

The Simulation Engine is implemented in Go for high-performance numerical computations and efficient price data generation. Go's strong performance characteristics make it suitable for continuous real-time simulations. This component:
- Implements a Geometric Brownian Motion model for price simulation
- Runs on a continuous tick (every 100ms) to generate price updates
- Maintains a subscription system for contracts to receive price updates
- Manages base price evolution using configurable parameters:
  - Drift coefficient (mu)
  - Volatility coefficient (sigma)
  - Time step (dt)

Communication:
- With WebSocket Server:
  - Direct in-memory communication
  - Provides price updates through PriceHandler interface
  - Receives subscription management commands
- Data Format:
  - Standardized numeric precision for price data
  - Timestamps in ISO 8601 format

### Contracts Service [Python]

The Contracts Service is implemented in Python to leverage its extensive mathematical libraries and flexibility in implementing complex business logic. Python's rich ecosystem and expressive syntax make it ideal for implementing various contract types and their calculations. This component:
- Provides REST API endpoints for contract operations
- Supports multiple contract types (Lucky Ladder, Momentum Catcher)
- Handles contract initialization and lifecycle management
- Processes price updates and calculates contract outcomes
- Integrates with the Storage Service for persistence

Communication:
- With WebSocket Server:
  - Exposes REST API endpoints
  - Handles contract lifecycle events
  - Processes price updates
- With Storage Service:
  - HTTP/REST communication
  - Endpoints used:
    - POST /contract - Save contract data
    - GET /contract?id={id} - Retrieve contract data
    - GET /contract - Retrieve all contracts
    - DELETE /contract?id={id} - Delete contract data
    - POST /clean - Clean database
- Error Handling:
  - HTTP status codes for API errors
  - Detailed error messages in JSON response bodies
- Data Format:
  - JSON for all request/response bodies
  - Standardized contract state representations

### Storage Service [Go]

The Storage Service is implemented in Go to provide efficient database operations and handle concurrent requests effectively. Go's strong standard library and excellent database connectivity make it well-suited for this role. This component:
- Implements a RESTful API for CRUD operations
- Manages contract persistence with automatic schema initialization
- Handles concurrent access to contract data
- Provides data cleanup and maintenance operations

Communication:
- With Contracts Service:
  - Exposes REST API endpoints
  - JSON-formatted data exchange
  - Endpoints:
    - POST /contract - Save contract data
    - GET /contract?id={id} - Retrieve contract data
    - GET /contract - Retrieve all contracts
    - DELETE /contract?id={id} - Delete contract data
    - POST /clean - Clean database
- Error Handling:
  - HTTP status codes for operation results
  - Detailed error messages in responses
- Data Format:
  - JSON for contract data storage
  - PostgreSQL for persistent storage
  - Timestamps in ISO 8601 format

## Data Flow

1. Clients connect to the Go-based WebSocket Server Hub
2. The Hub registers clients and manages their connections
3. The Go-based Simulation Engine generates price updates
4. Price updates are distributed to subscribed contracts
5. The Python-based Contracts Service processes updates and calculates outcomes
6. Contract states are persisted through the Go-based Storage Service
7. Results are broadcast back to clients through the Hub

## Contract Types

The system supports different types of financial contracts, all implemented in Python within the Contracts Service:

### Lucky Ladder
- Defines a series of price levels (rungs)
- Tracks price movements through the rungs
- Calculates payouts based on achieved rungs

### Momentum Catcher
- Monitors price momentum
- Tracks cumulative price movements
- Triggers based on target movement thresholds

Each contract type implements specific logic for processing price updates and determining outcomes.

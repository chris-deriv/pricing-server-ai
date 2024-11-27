# Pricing Server Architecture

The pricing server is a distributed system designed to manage and simulate financial contracts with real-time price updates. The system consists of several key components that work together to provide a robust and scalable solution.

## System Overview

The system is built using a microservices architecture with the following main components:
- WebSocket Server (Go)
- Contracts Service (Python)
- Storage Service (Go)
- Simulation Engine (Go)

```
                                    WebSocket
                                   Connection
┌──────────┐                    ┌───────-────┐
│  Client  │◄────WebSocket─────►│            │
└──────────┘                    │            │         ┌─────────────────┐
                                │ WebSocket  │         │                 │
┌──────────┐                    │  Server    │◄───────►│ Simulation      │
│  Client  │◄────WebSocket─────►│   Hub      │         │ Engine          │
└──────────┘                    │            │         │                 │
                                │            │         └─────────────────┘
┌──────────┐                    │            │
│  Client  │◄────WebSocket─────►│            │
└──────────┘                    └──────┬─────┘
                                       │
                                       │
                                       │
                                 ┌─────▼─────┐         ┌─────────────────┐
                                 │           │         │                 │
                                 │ Contracts │◄───────►│    Storage      │
                                 │ Service   │   REST  │    Service      │
                                 │           │         │                 │
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

### WebSocket Server Hub

The Hub is the central communication component that:
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

### Simulation Engine

The Simulation Engine is responsible for generating realistic price data:
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

### Contracts Service

The Contracts Service manages the business logic for different types of financial contracts:
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
    - DELETE /contract?id={id} - Delete contract data
- Error Handling:
  - HTTP status codes for API errors
  - Detailed error messages in JSON response bodies
- Data Format:
  - JSON for all request/response bodies
  - Standardized contract state representations

### Storage Service

The Storage Service provides persistent storage for contract data using PostgreSQL:
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

1. Clients connect to the WebSocket Server Hub
2. The Hub registers clients and manages their connections
3. The Simulation Engine generates price updates
4. Price updates are distributed to subscribed contracts
5. The Contracts Service processes updates and calculates outcomes
6. Contract states are persisted through the Storage Service
7. Results are broadcast back to clients through the Hub

## Contract Types

The system supports different types of financial contracts:

### Lucky Ladder
- Defines a series of price levels (rungs)
- Tracks price movements through the rungs
- Calculates payouts based on achieved rungs

### Momentum Catcher
- Monitors price momentum
- Tracks cumulative price movements
- Triggers based on target movement thresholds

Each contract type implements specific logic for processing price updates and determining outcomes.

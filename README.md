# Pricing Server

A real-time pricing server that supports various financial products like Lucky Ladder and Momentum Catcher.

## Prerequisites

- Docker
- Docker Compose (optional)
- Go 1.21+ (for local development)

## Environment Configuration

The system uses environment variables for configuration. A sample environment file (`sample_environment`) is provided as a template.

### Setup

1. Copy the sample environment file:
```bash
cp sample_environment .env
```

2. Update the values in `.env` with your actual configuration
3. The `.env` file is automatically excluded from version control via `.gitignore`

### Available Environment Variables

#### Database Configuration
- `DB_HOST`: PostgreSQL host (default: postgres)
- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_USER`: Database username
- `DB_PASSWORD`: Database password
- `DB_NAME`: Database name

#### Service Ports
- `WEBSOCKET_SERVER_PORT`: WebSocket server port (default: 8080)
- `CONTRACTS_SERVICE_PORT`: Contracts service port (default: 8000)
- `STORAGE_SERVICE_PORT`: Storage service port (default: 8001)

#### Simulation Configuration
- `SIMULATION_TICK_INTERVAL_MS`: Price update interval in milliseconds (default: 100)
- `SIMULATION_BASE_PRICE`: Starting price for simulation (default: 100.0)

#### Contract Configuration
- `CONTRACT_MAX_DURATION_MS`: Maximum contract duration (default: 3600000)
- `CONTRACT_MIN_DURATION_MS`: Minimum contract duration (default: 1000)

#### Other Settings
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `DEBUG`: Enable debug logging (default: false)

## Running with Docker

### Build and Run with Docker

```bash
# Build the Docker image
docker build -t pricing-server .

# Run the container
docker run -p 8080:8080 --env-file .env pricing-server
```

### Run with Docker Compose

```bash
# Start the service
docker compose up

# Start in detached mode
docker compose up -d

# Stop the service
docker compose down
```

## API Endpoints

WebSocket endpoint: `ws://localhost:8080/ws`

### Contract Types

1. Lucky Ladder
```json:README.md
{
    "type": "ContractSubmission",
    "data": {
        "productType": "LuckyLadder",
        "rungs": [101, 102, 103],
        "duration": "1m",
        "payoff": 100
    }
}
```

2. Momentum Catcher
```json:README.md
{
    "type": "ContractSubmission",
    "data": {
        "productType": "MomentumCatcher",
        "targetMovement": 1.5,
        "duration": "1m",
        "payoff": 100
    }
}
```

## Development

### Local Build

```bash
go build -o main .

# Pricing Server

A real-time pricing server that supports various financial products like Lucky Ladder and Momentum Catcher.

## Prerequisites

- Docker
- Docker Compose (optional)
- Go 1.21+ (for local development)

## Running with Docker

### Build and Run with Docker

```bash
# Build the Docker image
docker build -t pricing-server .

# Run the container
docker run -p 8080:8080 -e DEBUG=true pricing-server
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

## Environment Variables

- `DEBUG`: Enable debug logging (default: false)
  - Example: `DEBUG=true`

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
```


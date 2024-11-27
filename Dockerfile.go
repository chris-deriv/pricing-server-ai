# Build stage
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /pricingserver ./cmd/pricingserver

# Final stage
FROM alpine:latest

WORKDIR /

# Copy the binary from builder
COPY --from=builder /pricingserver /pricingserver

# Add ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Default port (can be overridden by environment variable)
ENV WEBSOCKET_SERVER_PORT=8080

# Expose the port
EXPOSE ${WEBSOCKET_SERVER_PORT}

# Run the binary
CMD ["/pricingserver"]

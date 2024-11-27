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

# Expose the port
EXPOSE 8080

# Run the binary
CMD ["/pricingserver"]

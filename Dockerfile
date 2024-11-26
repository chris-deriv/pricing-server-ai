# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.21-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Now copy the source code
COPY ./cmd/pricingserver/main.go ./
COPY internal/ ./internal/

# Build the application with the correct output path
RUN mkdir -p /app/cmd/pricingserver && go build -o /app/cmd/pricingserver/main .

# Final stage
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/cmd/pricingserver/main ./main

# Set the environment variable
ENV DEBUG=true

# Specify the full path to the executable
CMD ["/app/main"]
FROM golang:1.20-alpine

WORKDIR /app

# Install netcat and postgresql-client for health checks
RUN apk add --no-cache netcat-openbsd postgresql-client

# Copy Go module files and source code
COPY go.mod go.sum main.go ./

# Download dependencies
RUN go mod download

RUN go build -o storage_service

EXPOSE 8001

CMD ["./storage_service"]

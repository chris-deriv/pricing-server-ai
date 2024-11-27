FROM golang:1.20-alpine

WORKDIR /app

# Install netcat and postgresql-client for health checks
RUN apk add --no-cache netcat-openbsd postgresql-client

COPY go.mod .

# Download dependencies
RUN go mod download

COPY main.go .

RUN go build -o storage_service

EXPOSE 8001

CMD ["./storage_service"]

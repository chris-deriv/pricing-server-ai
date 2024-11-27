FROM golang:1.20-alpine

WORKDIR /app

COPY go.mod .
COPY main.go .

RUN go mod download
RUN go mod tidy

RUN go build -o storage_service

EXPOSE 8001

CMD ["./storage_service"]

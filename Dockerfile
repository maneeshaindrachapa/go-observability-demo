FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/order-service ./cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/order-service .

EXPOSE 8080

CMD ["./order-service"]

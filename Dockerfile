# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /telegram-shop ./cmd/server

# Run stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /telegram-shop .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./telegram-shop"]

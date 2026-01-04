# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api ./cmd/api

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binaries from builder
COPY --from=builder /app/bin/ /app/bin/
COPY --from=builder /app/config.yaml /app/config.yaml
COPY --from=builder /app/internal/pb/swagger/ /app/internal/pb/swagger/

# Default command (can be overridden)
CMD ["/app/bin/api"]


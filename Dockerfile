# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git make
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /api ./cmd/app

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /api .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/docs ./docs
ENV TZ=UTC
ENV MIGRATIONS_PATH=file:///root/migrations
EXPOSE 8080
CMD ["./api"]
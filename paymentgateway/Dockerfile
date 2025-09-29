FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/config ./config

RUN adduser -D -s /bin/sh appuser
USER appuser

EXPOSE 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8082/health || exit 1

# Run the application
CMD ["./main"]

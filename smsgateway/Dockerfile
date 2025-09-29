FROM golang:1.24-alpine AS builder
WORKDIR /app
#RUN apk add --no-cache git ca-certificates tzdata
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE=api
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/${SERVICE}

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config

RUN adduser -D -s /bin/sh appuser
USER appuser

CMD ["./main"]
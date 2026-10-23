# Multi-stage Dockerfile for Go Fiber Backend
FROM golang:alpine AS builder

WORKDIR /app

# Install git and certificates
RUN apk add --no-cache git ca-certificates tzdata

ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=auto

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN go build -ldflags="-w -s" -o /app/server cmd/server/main.go

# Production Runner Stage
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

# Copy compiled binary
COPY --from=builder /app/server /app/server

# Create directory for media uploads
RUN mkdir -p /app/uploads

EXPOSE 8080

CMD ["/app/server"]

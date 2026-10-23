# Multi-stage Dockerfile for Go Fiber Backend
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git and build essentials
RUN apk add --no-cache git ca-certificates tzdata

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/server cmd/server/main.go

# Production Runner Stage
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

# Copy compiled binary
COPY --from=builder /app/server /app/server

# Create directory for media uploads
RUN mkdir -p /app/uploads

EXPOSE 8080

CMD ["/app/server"]

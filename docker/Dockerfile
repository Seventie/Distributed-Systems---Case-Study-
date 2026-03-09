# ═══════════════════════════════════════════════════════════
# Dockerfile for Distributed Ticket Booking System Node
# Multi-stage build for a minimal production image
# ═══════════════════════════════════════════════════════════

# --- Stage 1: Build ---
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install protoc and dependencies
RUN apk add --no-cache protobuf protobuf-dev git

# Install Go protobuf plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy go.mod first for dependency caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy source code
COPY . .

# Generate protobuf code
RUN protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    -I . \
    proto/ticket/ticket.proto

# Build the node binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/node cmd/node/main.go

# Build the beam analytics binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/beam-analytics beam/analytics/pipeline.go

# --- Stage 2: Runtime ---
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binaries
COPY --from=builder /bin/node /app/node
COPY --from=builder /bin/beam-analytics /app/beam-analytics

# Copy web static files and configs
COPY web/ /app/web/
COPY configs/ /app/configs/

# Create logs directory
RUN mkdir -p /app/logs

# Expose gRPC and HTTP ports (default for node 1; override with env)
EXPOSE 50051 8081

# Default command: run node 1 (override with docker-compose)
ENTRYPOINT ["/app/node"]
CMD ["-config", "/app/configs/node1.yaml"]

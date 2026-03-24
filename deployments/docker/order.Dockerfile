# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o order-service ./cmd/order/main.go

# Stage 2: Final minimal image
FROM gcr.io/distroless/static-debian11

WORKDIR /

# Copy the binary from builder
COPY --from=builder /app/order-service /order-service

# Copy certificates for mTLS
COPY --from=builder /app/deployments/certs /deployments/certs

# Run as non-root user (distroless has 'nonroot' user with UID 65532)
USER nonroot:nonroot

# Expose port
EXPOSE 8080

ENTRYPOINT ["/order-service"]

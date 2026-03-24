# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o inventory-service ./cmd/inventory/main.go

# Stage 2: Final minimal image
FROM gcr.io/distroless/static-debian11

WORKDIR /

COPY --from=builder /app/inventory-service /inventory-service

USER nonroot:nonroot

# Inventory service is a worker, but exposes port 8082 for Prometheus metrics
EXPOSE 8082

ENTRYPOINT ["/inventory-service"]

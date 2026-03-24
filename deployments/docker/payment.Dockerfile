# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o payment-service ./cmd/payment/main.go

# Stage 2: Final minimal image
FROM gcr.io/distroless/static-debian11

WORKDIR /

COPY --from=builder /app/payment-service /payment-service
COPY --from=builder /app/deployments/certs /deployments/certs

USER nonroot:nonroot

EXPOSE 8081

ENTRYPOINT ["/payment-service"]

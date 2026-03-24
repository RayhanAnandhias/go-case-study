# Distributed Order Processing System (Golang Case Study)

Sistem pemrosesan pesanan e-commerce terdistribusi yang dibangun menggunakan Go dengan fokus pada **Reliability**, **Scalability**, dan **Observability**.

## 🏗 Arsitektur
Sistem ini terdiri dari 3 microservices:
1.  **Order Service (Port 8080)**: HTTP API untuk menerima pesanan, melakukan rate limiting (Redis), caching produk, dan memulai transaksi.
2.  **Payment Service (Port 8081)**: Internal API terproteksi mTLS untuk memproses pembayaran (mock).
3.  **Inventory Service (Port 8082)**: Kafka Worker pool yang memproses pemotongan stok secara idempotent dengan mekanisme retry & DLQ.

### Tech Stack
- **Language**: Go 1.23+
- **Architecture**: Clean Architecture
- **Database**: PostgreSQL (Multi-schema)
- **Cache & Rate Limiter**: Redis
- **Message Broker**: Apache Kafka (KRaft mode)
- **Observability**: Prometheus (Metrics), Jaeger (Tracing via OpenTelemetry)
- **Security**: JWT (OIDC Mock), mTLS (Service-to-Service)

## 🚀 Persiapan & Instalasi

### 1. Prasyarat
- Docker / Podman
- Go 1.23+ (untuk menjalankan lokal)
- OpenSSL (untuk men-generate sertifikat)

### 2. Setup Infrastruktur
Jalankan komponen pendukung (DB, Redis, Kafka, Monitoring) menggunakan Docker Compose:
```bash
docker compose up -d
```

### 3. Generate Sertifikat mTLS
Eksekusi script berikut untuk membuat CA dan sertifikat lokal:
- **Windows**: `powershell ./deployments/certs/generate.ps1`
- **Linux/macOS**: `bash ./deployments/certs/generate.sh`

### 4. Menjalankan Service (Lokal)
Pastikan file `.env` sudah sesuai, lalu jalankan di terminal terpisah:
```bash
# Order Service
go run cmd/order/main.go

# Payment Service
go run cmd/payment/main.go

# Inventory Service
go run cmd/inventory/main.go
```

## 🛠 Penggunaan API

### 1. Dapatkan JWT Token
Endpoint dummy untuk mendapatkan token testing:
`GET http://localhost:8080/order-service/api/v1/auth/token`

### 2. Buat Pesanan
Gunakan token dari langkah 1 pada header Authorization:
```bash
curl -X POST http://localhost:8080/order-service/api/v1/orders \
-H "Authorization: Bearer <TOKEN_ANDA>" \
-H "Content-Type: application/json" \
-d '{
    "items": [
        {
            "product_id": "11111111-1111-1111-1111-111111111111",
            "quantity": 2
        }
    ]
}'
```

## 📊 Observability
- **Metrics**: `http://localhost:9090` (Prometheus)
- **Tracing**: `http://localhost:16686` (Jaeger UI)
- **Kafka UI**: `http://localhost:8090` (Melihat pesan di topic `order.created` atau `order.failed`)

## 🧪 Testing
Jalankan unit test untuk semua service:
```bash
go test ./...
```

## 🐳 Dockerization
Dockerfile tersedia di `deployments/docker/` menggunakan pendekatan **multi-stage build** dan image akhir berbasis **distroless** untuk keamanan maksimal.

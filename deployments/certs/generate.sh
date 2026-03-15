#!/bin/bash
set -e

# Pindah ke direktori script ini berada
cd "$(dirname "$0")"

echo "Mulai men-generate sertifikat mTLS..."

# 1. Generate CA (Certificate Authority)
# ca.key = Kunci Privat CA (Rahasia!)
# ca.crt = Sertifikat Publik CA (Dibagikan ke semua pihak)
openssl req -new -x509 -nodes -days 3650 -keyout ca.key -out ca.crt -subj "/C=ID/ST=West Java/L=Bandung/O=GoCaseStudy/OU=IT/CN=CA"

# 2. Generate Server Certificate (Payment Service)
# server.key = Kunci Privat Server (Hanya Payment Service yang tahu)
# server.csr = Certificate Signing Request (Permintaan tanda tangan ke CA)
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/C=ID/ST=West Java/L=Bandung/O=GoCaseStudy/OU=Payment/CN=localhost"

# Gunakan extfile agar localhost/127.0.0.1 dikenali sebagai host yang valid
echo "subjectAltName=DNS:localhost,IP:127.0.0.1" > server.ext

# server.crt = Sertifikat Server yang sah (Telah ditandatangani oleh CA)
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -extfile server.ext

# 3. Generate Client Certificate (Order Service)
# client.key = Kunci Privat Client (Hanya Order Service yang tahu)
# client.csr = Certificate Signing Request
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr -subj "/C=ID/ST=West Java/L=Bandung/O=GoCaseStudy/OU=Order/CN=order-client"

# client.crt = Sertifikat Client yang sah (Telah ditandatangani oleh CA)
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365

# Bersihkan file .csr dan .ext yang sudah tidak dipakai
rm -f server.csr client.csr server.ext ca.srl

echo "Berhasil membuat sertifikat mTLS!"

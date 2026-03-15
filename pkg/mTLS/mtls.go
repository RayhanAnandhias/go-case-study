package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// LoadServerTLSConfig creates a TLS config for the server that requires client certificates.
func LoadServerTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Load server certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("could not load server key pair: %v", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("could not read CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate to pool")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// We use VerifyClientCertIfGiven instead of RequireAndVerifyClientCert
		// so that Prometheus (which doesn't have a client cert) can still scrape /metrics.
		// For actual API endpoints, we will verify the certificate presence in middleware/handler.
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    caCertPool,
	}

	return tlsConfig, nil
}

// LoadClientTLSConfig creates a TLS config for the client to authenticate with the server.
func LoadClientTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("could not load client key pair: %v", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("could not read CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate to pool")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	return tlsConfig, nil
}

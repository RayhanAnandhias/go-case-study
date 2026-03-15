package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AppEnv           string `envconfig:"APP_ENV"`
	OrderPort        string `envconfig:"ORDER_PORT"`
	OrderBaseUrl     string `envconfig:"ORDER_BASE_URL"`
	PaymentPort      string `envconfig:"PAYMENT_PORT"`
	PaymentBaseUrl   string `envconfig:"PAYMENT_BASE_URL"`
	InventoryPort    string `envconfig:"INVENTORY_PORT"`
	InventoryBaseUrl string `envconfig:"INVENTORY_BASE_URL"`
	DatabaseURL      string `envconfig:"DATABASE_URL"`
	RedisURL         string `envconfig:"REDIS_URL"`
	KafkaBrokers     string `envconfig:"KAFKA_BROKERS"`
	OtlpEndpoint     string `envconfig:"OTLP_ENDPOINT"`
	JwtSecret        string `envconfig:"JWT_SECRET"`
	PaymentService   string `envconfig:"PAYMENT_SERVICE_URL"`
}

func Load() *Config {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	return &cfg
}

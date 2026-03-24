package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"go-case-study/internal/order/handler"
	"go-case-study/internal/order/repository"
	"go-case-study/internal/order/service"
	"go-case-study/pkg/config"
	"go-case-study/pkg/database"
	"go-case-study/pkg/kafka"
	"go-case-study/pkg/logger"
	"go-case-study/pkg/mTLS"
	"go-case-study/pkg/middleware"
	"go-case-study/pkg/redis"
	"go-case-study/pkg/tracer"
)

func main() {
	// 1. Load config
	_ = godotenv.Load()
	cfg := config.Load()

	// 2. Initialize logger
	logger.SetupLogger()
	log.Println("Starting Order Service...")

	// 2.5 Initialize Tracer
	tp, err := tracer.InitTracer("order-service", cfg.OtlpEndpoint)
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	} else {
		defer tp.Shutdown(context.Background())
	}

	// 3. Initialize DB
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer db.Close()

	// 4. Initialize Redis
	redisClient, err := redis.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatal("Failed to connect to redis: ", err)
	}
	defer redisClient.Close()

	// 5. Initialize Kafka Producer
	kafkaProducer := kafka.NewWriter([]string{cfg.KafkaBrokers}, "order.created")
	if strings.Contains(cfg.KafkaBrokers, ",") {
		kafkaProducer = kafka.NewWriter(strings.Split(cfg.KafkaBrokers, ","), "order.created")
	}
	defer kafkaProducer.Close()

	// 6. Setup Router
	router := gin.Default()
	router.Use(otelgin.Middleware("order-service"))
	router.Use(middleware.TraceLogger())
	router.Use(middleware.PrometheusMetrics("order-service"))

	router.GET(cfg.OrderBaseUrl+"/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Expose metrics endpoint
	router.GET(cfg.OrderBaseUrl+"/metrics", gin.WrapH(promhttp.Handler()))

	// Initialize repositories, services, and handlers
	orderRepo := repository.NewOrderRepository(db)
	productRepo := repository.NewProductRepository(redisClient, db)

	tlsConfig, err := mtls.LoadClientTLSConfig(
		"deployments/certs/client.crt",
		"deployments/certs/client.key",
		"deployments/certs/ca.crt",
	)
	if err != nil {
		log.Fatal("Failed to load client TLS config: ", err)
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	orderSvc := service.NewOrderService(
		orderRepo,
		productRepo,
		redisClient,
		db,
		kafkaProducer,
		cfg.PaymentService,
		httpClient,
	)

	orderHandler := handler.NewOrderHandler(orderSvc, cfg.JwtSecret)

	// Register routes
	v1 := router.Group(cfg.OrderBaseUrl + "/api/v1")
	{
		v1.POST("/auth/token", orderHandler.GenerateToken)
		v1.POST("/orders", middleware.RequireAuth(cfg.JwtSecret), orderHandler.CreateOrder)
	}

	// 7. Start HTTP server with Graceful Shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.OrderPort),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to listen and serve: ", err)
		}
	}()
	log.Println(fmt.Sprintf("Order Service is running on port %s", cfg.OrderPort))

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go-case-study/internal/payment/handler"
	"go-case-study/internal/payment/repository"
	"go-case-study/internal/payment/service"
	"go-case-study/pkg/config"
	"go-case-study/pkg/database"
	"go-case-study/pkg/logger"
	"go-case-study/pkg/mTLS"
	"go-case-study/pkg/middleware"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Load config
	_ = godotenv.Load()
	cfg := config.Load()

	// 1.5. Initialize logger
	logger.SetupLogger()
	log.Println("Starting Payment Service...")

	// 2. Database
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 3. Domain initialization
	repo := repository.NewPaymentRepository(db)
	svc := service.NewPaymentService(repo)
	h := handler.NewPaymentHandler(svc)

	// 4. Gin Router setup
	router := gin.Default()
	router.Use(middleware.TraceLogger())
	router.Use(middleware.PrometheusMetrics("payment-service"))

	router.GET(cfg.OrderBaseUrl+"/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	router.GET(cfg.PaymentBaseUrl+"/metrics", gin.WrapH(promhttp.Handler()))

	// Middleware to enforce mTLS for API endpoints
	router.Use(func(c *gin.Context) {
		if c.Request.TLS != nil && len(c.Request.TLS.PeerCertificates) == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "client certificate required"})
			return
		}
		c.Next()
	})

	// Register routes
	v1 := router.Group(cfg.PaymentBaseUrl + "/api/v1")
	{
		v1.POST("/payments", middleware.RequireAuth(cfg.JwtSecret), h.ProcessPayment)
	}

	// 5. Start Server
	port := cfg.PaymentPort
	if port == "" {
		port = "8081" // fallback if empty
	}

	tlsConfig, err := mtls.LoadServerTLSConfig(
		"deployments/certs/server.crt",
		"deployments/certs/server.key",
		"deployments/certs/ca.crt",
	)
	if err != nil {
		log.Fatalf("Failed to load TLS config: %v", err)
	}

	srv := &http.Server{
		Addr:      ":" + port,
		Handler:   router,
		TLSConfig: tlsConfig,
	}

	go func() {
		log.Printf("Starting Payment Service on port %s", port)
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

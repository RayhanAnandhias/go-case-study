package main

import (
	"context"
	"fmt"
	"go-case-study/pkg/middleware"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go-case-study/internal/inventory/repository"
	"go-case-study/internal/inventory/service"
	"go-case-study/internal/inventory/worker"
	"go-case-study/pkg/config"
	"go-case-study/pkg/database"
	"go-case-study/pkg/kafka"
	"go-case-study/pkg/logger"
	"go-case-study/pkg/tracer"
)

func main() {
	// 1. Load config
	_ = godotenv.Load()
	cfg := config.Load()

	// 2. Initialize logger
	logger.SetupLogger()
	log.Println("Starting Inventory Service...")

	// 2.5 Initialize Tracer
	tp, err := tracer.InitTracer("inventory-service", cfg.OtlpEndpoint)
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

	// 4. Initialize Kafka Reader
	brokers := []string{cfg.KafkaBrokers}
	if strings.Contains(cfg.KafkaBrokers, ",") {
		brokers = strings.Split(cfg.KafkaBrokers, ",")
	}

	kafkaReader := kafka.NewReader(brokers, "order.created", "inventory-service-group")

	// 5. Initialize Repositories and Services
	repo := repository.NewInventoryRepository(db)
	svc := service.NewInventoryService(repo, db)

	dlqWriter := kafka.NewWriter(strings.Split(cfg.KafkaBrokers, ","), "order.failed")
	defer dlqWriter.Close()

	// 6. Start Worker Pool
	workerPoolSize := 10
	kafkaWorker := worker.NewKafkaWorker(kafkaReader, dlqWriter, svc, workerPoolSize)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	kafkaWorker.Start(ctx, &wg)

	log.Printf("Inventory Service Kafka Worker is running with %d goroutines\n", workerPoolSize)

	// 6.5 Start HTTP Server for metrics and health
	router := gin.Default()
	router.Use(middleware.TraceLogger())
	router.Use(middleware.PrometheusMetrics("inventory-service"))

	router.GET(cfg.InventoryBaseUrl+"/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Expose metrics endpoint
	router.GET(cfg.InventoryBaseUrl+"/metrics", gin.WrapH(promhttp.Handler()))

	port := cfg.InventoryPort
	if port == "" {
		port = "8082"
	}
	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		log.Printf("Starting Inventory HTTP server for metrics on port %s", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 7. Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down worker...")

	// Cancel context to stop fetching messages
	cancel()

	// Shutdown HTTP Server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Close reader
	if err := kafkaReader.Close(); err != nil {
		log.Printf("Error closing kafka reader: %v", err)
	}

	// Wait for workers to finish
	wg.Wait()
	log.Println("Worker exiting gracefully")
}

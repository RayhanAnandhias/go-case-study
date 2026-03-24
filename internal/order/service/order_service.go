package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"go-case-study/internal/order/domain"
	"go-case-study/internal/order/repository"
	pkgkafka "go-case-study/pkg/kafka"
	"go-case-study/pkg/logger"
)

type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type orderService struct {
	repo           domain.OrderRepository
	productRepo    domain.ProductRepository
	redisClient    *redis.Client
	db             *sql.DB
	kafkaWriter    KafkaWriter
	paymentService string
	httpClient     *http.Client
	tracer         trace.Tracer
}

func NewOrderService(
	repo domain.OrderRepository,
	productRepo domain.ProductRepository,
	redisClient *redis.Client,
	db *sql.DB,
	kafkaWriter KafkaWriter,
	paymentService string,
	httpClient *http.Client,
) domain.OrderService {
	// Wrap httpClient transport with otelhttp
	if httpClient.Transport == nil {
		httpClient.Transport = http.DefaultTransport
	}
	httpClient.Transport = otelhttp.NewTransport(httpClient.Transport)

	return &orderService{
		repo:           repo,
		productRepo:    productRepo,
		redisClient:    redisClient,
		db:             db,
		kafkaWriter:    kafkaWriter,
		paymentService: paymentService,
		httpClient:     httpClient,
		tracer:         otel.Tracer("order-service"),
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {
	ctx, span := s.tracer.Start(ctx, "CreateOrder")
	defer span.End()

	// 1. Rate limiting per-user using Redis token bucket logic
	if err := s.checkRateLimit(ctx, req.UserID); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// 2. Fetch product prices and calculate total amount
	var totalAmount float64
	var orderItems []domain.OrderItem
	for _, reqItem := range req.Items {
		price, err := s.productRepo.GetPrice(ctx, reqItem.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch product price: %w", err)
		}
		totalAmount += price * float64(reqItem.Quantity)
		orderItems = append(orderItems, domain.OrderItem{
			ID:        uuid.New(),
			ProductID: reqItem.ProductID,
			Quantity:  reqItem.Quantity,
			Price:     price,
			CreatedAt: time.Now().UTC(),
		})
	}

	order := &domain.Order{
		ID:          uuid.New(),
		UserID:      req.UserID,
		TotalAmount: totalAmount,
		Status:      domain.OrderStatusPending,
		Items:       orderItems,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	for i := range order.Items {
		order.Items[i].OrderID = order.ID
	}

	// 3. Start DB transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	ctxWithTx := repository.InjectTx(ctx, tx)

	// 4. Create order (PENDING)
	if err := s.repo.CreateOrder(ctxWithTx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 5. Call Payment Service synchronously
	if err := s.processPayment(ctx, order); err != nil {
		// 6. If payment fails, return error, rollback tx (done by defer)
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	// 7. Publish order.created event to Kafka
	if err := s.publishOrderEvent(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}

	// Update order status to PAID
	order.Status = domain.OrderStatusPaid
	if err := s.repo.UpdateOrderStatus(ctxWithTx, order.ID, order.Status); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	// 8. Commit tx
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return order, nil
}

func (s *orderService) GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	return s.repo.GetOrder(ctx, id)
}

func (s *orderService) checkRateLimit(ctx context.Context, userID uuid.UUID) error {
	// Simple Redis Token Bucket Logic via Lua Script
	script := redis.NewScript(`
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = 1
		
		local bucket = redis.call("HMGET", key, "tokens", "last_refresh")
		local tokens = tonumber(bucket[1])
		local last_refresh = tonumber(bucket[2])
		
		if not tokens then
			tokens = capacity
			last_refresh = now
		end
		
		local time_passed = math.max(0, now - last_refresh)
		tokens = math.min(capacity, tokens + time_passed * rate)
		
		if tokens < requested then
			return 0
		end
		
		redis.call("HMSET", key, "tokens", tokens - requested, "last_refresh", now)
		redis.call("EXPIRE", key, math.ceil(capacity / rate))
		return 1
	`)

	capacity := 10 // max 10 requests
	rate := 1      // 1 request per second
	now := time.Now().Unix()

	res, err := script.Run(ctx, s.redisClient, []string{"rate_limit:" + userID.String()}, capacity, rate, now).Result()
	if err != nil {
		return err
	}
	if allowed, ok := res.(int64); ok && allowed == 0 {
		return errors.New("too many requests")
	}
	return nil
}

func (s *orderService) processPayment(ctx context.Context, order *domain.Order) error {
	ctx, span := s.tracer.Start(ctx, "processPayment")
	defer span.End()

	payload, _ := json.Marshal(map[string]interface{}{
		"order_id": order.ID,
		"amount":   order.TotalAmount,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", s.paymentService+"/api/v1/payments", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ctx.Value("auth_token").(string))
	if traceID, ok := ctx.Value(logger.TraceIDKey).(string); ok {
		req.Header.Set("X-Trace-Id", traceID)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return errors.New("payment service returned non-200")
	}
	return nil
}

func (s *orderService) publishOrderEvent(ctx context.Context, order *domain.Order) error {
	ctx, span := s.tracer.Start(ctx, "publishOrderEvent")
	defer span.End()

	eventPayload, _ := json.Marshal(order)
	msg := kafka.Message{
		Key:   []byte(order.ID.String()),
		Value: eventPayload,
	}

	// Inject trace context into Kafka headers
	carrier := pkgkafka.HeaderCarrier(msg.Headers)
	otel.GetTextMapPropagator().Inject(ctx, &carrier)
	msg.Headers = []kafka.Header(carrier)

	// Keep existing trace ID logic if needed
	if traceID, ok := ctx.Value(logger.TraceIDKey).(string); ok {
		msg.Headers = append(msg.Headers, kafka.Header{Key: "trace_id", Value: []byte(traceID)})
	}

	return s.kafkaWriter.WriteMessages(ctx, msg)
}

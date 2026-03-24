package worker

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go-case-study/internal/inventory/domain"
	pkgkafka "go-case-study/pkg/kafka"
)

var (
	ordersProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "orders_processed_total",
		Help: "The total number of processed order events",
	})
)

type KafkaWorker struct {
	reader     *kafka.Reader
	dlqWriter  *kafka.Writer
	service    domain.InventoryService
	workerPool int
	tracer     trace.Tracer
}

func NewKafkaWorker(reader *kafka.Reader, dlqWriter *kafka.Writer, service domain.InventoryService, poolSize int) *KafkaWorker {
	return &KafkaWorker{
		reader:     reader,
		dlqWriter:  dlqWriter,
		service:    service,
		workerPool: poolSize,
		tracer:     otel.Tracer("inventory-service"),
	}
}

func (w *KafkaWorker) sendToDLQ(ctx context.Context, value []byte) error {
	if w.dlqWriter == nil {
		return nil
	}
	return w.dlqWriter.WriteMessages(ctx, kafka.Message{
		Value: value,
	})
}

func (w *KafkaWorker) Start(ctx context.Context, wg *sync.WaitGroup) {
	for i := 0; i < w.workerPool; i++ {
		wg.Add(1)
		go w.worker(ctx, wg, i)
	}
}

func (w *KafkaWorker) worker(ctx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	log.Printf("Starting Inventory Kafka worker %d", id)

	for {
		msg, err := w.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("Worker %d stopping due to context cancellation", id)
				return
			}
			log.Printf("Worker %d failed to fetch message: %v", id, err)
			continue
		}

		// Extract context from Kafka headers
		carrier := pkgkafka.HeaderCarrier(msg.Headers)
		msgCtx := otel.GetTextMapPropagator().Extract(ctx, &carrier)

		msgCtx, span := w.tracer.Start(msgCtx, "ProcessOrderEvent")

		var event domain.OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Worker %d failed to unmarshal message: %v", id, err)
			span.RecordError(err)
			span.End()
			// Commit poison pill to avoid getting stuck
			_ = w.reader.CommitMessages(ctx, msg)
			continue
		}

		var processErr error
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			processErr = w.service.ProcessOrderEvent(msgCtx, &event)
			if processErr == nil {
				break
			}
			log.Printf("Worker %d failed to process event %s (attempt %d/%d): %v", id, event.ID, i+1, maxRetries, processErr)
			time.Sleep(1 * time.Second) // Simple backoff
		}

		if processErr != nil {
			log.Printf("Worker %d failed to process event %s after %d retries. Sending to DLQ.", id, event.ID, maxRetries)
			span.RecordError(processErr)
			// Send to DLQ
			errDLQ := w.sendToDLQ(msgCtx, msg.Value)
			if errDLQ != nil {
				log.Printf("Worker %d failed to send event %s to DLQ: %v", id, event.ID, errDLQ)
			}
		}

		span.End()

		// Commit message
		if err := w.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Worker %d failed to commit message: %v", id, err)
		} else {
			ordersProcessedTotal.Inc()
			log.Printf("Worker %d successfully processed event %s", id, event.ID)
		}
	}
}

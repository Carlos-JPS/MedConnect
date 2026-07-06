package consumer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/MedConnect/notification-service/internal/processor"
)

func TestDLQPublisherPublishesToRealKafka(t *testing.T) {
	if os.Getenv("KAFKA_INTEGRATION") != "1" {
		t.Skip("set KAFKA_INTEGRATION=1 to run this test against a real Kafka broker")
	}

	brokers := []string{envOrDefault("KAFKA_INTEGRATION_BROKERS", "localhost:9092")}
	topic := envOrDefault("KAFKA_INTEGRATION_DLQ_TOPIC", "medconnect.booking.events.dlq.v1")
	publisher, err := NewDLQPublisher(brokers, topic)
	if err != nil {
		t.Fatalf("expected DLQ publisher, got %v", err)
	}
	defer publisher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	message := processor.Message{
		Topic: "medconnect.booking.events.v1",
		Key:   []byte("integration-test"),
		Value: []byte(`{"schema_version":99}`),
	}
	if err := publisher.Publish(ctx, message, "integration test invalid schema"); err != nil {
		t.Fatalf("expected DLQ publish to succeed, got %v", err)
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

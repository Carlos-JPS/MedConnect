package kafka

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MedConnect/booking-service/internal/outbox"
)

func TestPublisherPublishesToRealKafka(t *testing.T) {
	if os.Getenv("KAFKA_INTEGRATION") != "1" {
		t.Skip("set KAFKA_INTEGRATION=1 to run this test against a real Kafka broker")
	}

	brokers := []string{envOrDefault("KAFKA_INTEGRATION_BROKERS", "localhost:9092")}
	topic := envOrDefault("KAFKA_INTEGRATION_TOPIC", "medconnect.booking.events.v1")
	publisher, err := NewPublisher(brokers)
	if err != nil {
		t.Fatalf("expected publisher, got %v", err)
	}
	defer publisher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	eventID := fmt.Sprintf("11111111-1111-1111-1111-%012d", time.Now().UnixNano()%1_000_000_000_000)
	event := outbox.Event{
		EventID:     eventID,
		AggregateID: "22222222-2222-2222-2222-222222222222",
		EventType:   "booking.created",
		Topic:       topic,
		EventKey:    "22222222-2222-2222-2222-222222222222",
		Payload:     []byte(`{"event_id":"` + eventID + `","event_type":"booking.created","schema_version":1,"aggregate_id":"22222222-2222-2222-2222-222222222222","payload":{"booking_id":"22222222-2222-2222-2222-222222222222","patient_id":"33333333-3333-3333-3333-333333333333"}}`),
		CreatedAt:   time.Now().UTC(),
	}

	if err := publisher.Publish(ctx, event); err != nil {
		t.Fatalf("expected Kafka publish to succeed, got %v", err)
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

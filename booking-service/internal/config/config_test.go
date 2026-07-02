package config

import "testing"
import "time"

func TestLoadUsesDefaultsWhenEnvironmentIsMissing(t *testing.T) {
	t.Setenv("BOOKING_SERVICE_HOST", "")
	t.Setenv("BOOKING_SERVICE_PORT", "")
	t.Setenv("BOOKING_DB_DSN", "")
	t.Setenv("AVAILABILITY_SERVICE_TARGET", "")
	t.Setenv("PAYMENT_SERVICE_TARGET", "")
	t.Setenv("BOOKING_EXTERNAL_CALL_TIMEOUT", "")
	t.Setenv("KAFKA_BROKERS", "")
	t.Setenv("BOOKING_EVENTS_TOPIC", "")
	t.Setenv("OUTBOX_DISPATCHER_ENABLED", "")
	t.Setenv("OUTBOX_BATCH_SIZE", "")
	t.Setenv("OUTBOX_POLL_INTERVAL", "")
	t.Setenv("OUTBOX_RETRY_INITIAL_DELAY", "")
	t.Setenv("OUTBOX_RETRY_MAX_DELAY", "")

	cfg := Load()

	if cfg.GRPCHost != "0.0.0.0" {
		t.Fatalf("expected default gRPC host 0.0.0.0, got %q", cfg.GRPCHost)
	}
	if cfg.GRPCPort != "50051" {
		t.Fatalf("expected default gRPC port 50051, got %q", cfg.GRPCPort)
	}
	if cfg.DatabaseDSN != "" {
		t.Fatalf("expected empty default database dsn, got %q", cfg.DatabaseDSN)
	}
	if cfg.AvailabilityServiceTarget != "availability-service:50051" {
		t.Fatalf("expected default availability target, got %q", cfg.AvailabilityServiceTarget)
	}
	if cfg.PaymentServiceTarget != "payment-service:50051" {
		t.Fatalf("expected default payment target, got %q", cfg.PaymentServiceTarget)
	}
	if cfg.ExternalCallTimeout != 3*time.Second {
		t.Fatalf("expected default external call timeout 3s, got %s", cfg.ExternalCallTimeout)
	}
	if len(cfg.KafkaBrokers) != 1 || cfg.KafkaBrokers[0] != "kafka:9092" {
		t.Fatalf("expected default kafka broker, got %v", cfg.KafkaBrokers)
	}
	if cfg.BookingEventsTopic != "medconnect.booking.events.v1" {
		t.Fatalf("expected default booking topic, got %q", cfg.BookingEventsTopic)
	}
	if !cfg.OutboxDispatcherEnabled {
		t.Fatalf("expected outbox dispatcher to be enabled by default")
	}
	if cfg.OutboxBatchSize != 50 {
		t.Fatalf("expected default outbox batch size 50, got %d", cfg.OutboxBatchSize)
	}
	if cfg.OutboxPollInterval != 500*time.Millisecond {
		t.Fatalf("expected default outbox poll interval 500ms, got %s", cfg.OutboxPollInterval)
	}
	if cfg.OutboxInitialRetryDelay != time.Second {
		t.Fatalf("expected default outbox retry initial delay 1s, got %s", cfg.OutboxInitialRetryDelay)
	}
	if cfg.OutboxMaxRetryDelay != 30*time.Second {
		t.Fatalf("expected default outbox max retry delay 30s, got %s", cfg.OutboxMaxRetryDelay)
	}
}

func TestLoadUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("BOOKING_SERVICE_HOST", "127.0.0.1")
	t.Setenv("BOOKING_SERVICE_PORT", "60061")
	t.Setenv("BOOKING_DB_DSN", "postgres://booking:secret@db:5432/booking")
	t.Setenv("AVAILABILITY_SERVICE_TARGET", "availability:6000")
	t.Setenv("PAYMENT_SERVICE_TARGET", "payment:7000")
	t.Setenv("BOOKING_EXTERNAL_CALL_TIMEOUT", "1500ms")
	t.Setenv("KAFKA_BROKERS", "kafka-1:9092, kafka-2:9092")
	t.Setenv("BOOKING_EVENTS_TOPIC", "custom.booking.events")
	t.Setenv("OUTBOX_DISPATCHER_ENABLED", "false")
	t.Setenv("OUTBOX_BATCH_SIZE", "25")
	t.Setenv("OUTBOX_POLL_INTERVAL", "250ms")
	t.Setenv("OUTBOX_RETRY_INITIAL_DELAY", "2s")
	t.Setenv("OUTBOX_RETRY_MAX_DELAY", "45s")

	cfg := Load()

	if cfg.GRPCHost != "127.0.0.1" {
		t.Fatalf("expected env gRPC host, got %q", cfg.GRPCHost)
	}
	if cfg.GRPCPort != "60061" {
		t.Fatalf("expected env gRPC port, got %q", cfg.GRPCPort)
	}
	if cfg.DatabaseDSN != "postgres://booking:secret@db:5432/booking" {
		t.Fatalf("expected env database dsn, got %q", cfg.DatabaseDSN)
	}
	if cfg.AvailabilityServiceTarget != "availability:6000" {
		t.Fatalf("expected env availability target, got %q", cfg.AvailabilityServiceTarget)
	}
	if cfg.PaymentServiceTarget != "payment:7000" {
		t.Fatalf("expected env payment target, got %q", cfg.PaymentServiceTarget)
	}
	if cfg.ExternalCallTimeout != 1500*time.Millisecond {
		t.Fatalf("expected env external timeout, got %s", cfg.ExternalCallTimeout)
	}
	if len(cfg.KafkaBrokers) != 2 || cfg.KafkaBrokers[0] != "kafka-1:9092" || cfg.KafkaBrokers[1] != "kafka-2:9092" {
		t.Fatalf("expected env kafka brokers, got %v", cfg.KafkaBrokers)
	}
	if cfg.BookingEventsTopic != "custom.booking.events" {
		t.Fatalf("expected env booking topic, got %q", cfg.BookingEventsTopic)
	}
	if cfg.OutboxDispatcherEnabled {
		t.Fatalf("expected env outbox dispatcher disabled")
	}
	if cfg.OutboxBatchSize != 25 {
		t.Fatalf("expected env outbox batch size 25, got %d", cfg.OutboxBatchSize)
	}
	if cfg.OutboxPollInterval != 250*time.Millisecond {
		t.Fatalf("expected env outbox poll interval, got %s", cfg.OutboxPollInterval)
	}
	if cfg.OutboxInitialRetryDelay != 2*time.Second {
		t.Fatalf("expected env initial retry delay, got %s", cfg.OutboxInitialRetryDelay)
	}
	if cfg.OutboxMaxRetryDelay != 45*time.Second {
		t.Fatalf("expected env max retry delay, got %s", cfg.OutboxMaxRetryDelay)
	}
}

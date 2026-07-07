package config

import "testing"

func TestLoadUsesDefaultsWhenEnvironmentIsMissing(t *testing.T) {
	t.Setenv("NOTIFICATION_DB_DSN", "")
	t.Setenv("NOTIFICATION_HTTP_HOST", "")
	t.Setenv("NOTIFICATION_HTTP_PORT", "")
	t.Setenv("KAFKA_BROKERS", "")
	t.Setenv("BOOKING_EVENTS_TOPIC", "")
	t.Setenv("BOOKING_EVENTS_DLQ_TOPIC", "")
	t.Setenv("KAFKA_CONSUMER_GROUP", "")
	t.Setenv("NOTIFICATION_RETRY_LIMIT", "")
	t.Setenv("NOTIFICATION_LOG_ENABLED", "")

	cfg := Load()

	if cfg.HTTPHost != defaultHTTPHost {
		t.Fatalf("expected default http host, got %q", cfg.HTTPHost)
	}
	if cfg.HTTPPort != defaultHTTPPort {
		t.Fatalf("expected default http port, got %q", cfg.HTTPPort)
	}
	if cfg.DatabaseDSN != defaultDatabaseDSN {
		t.Fatalf("expected default database dsn, got %q", cfg.DatabaseDSN)
	}
	if len(cfg.KafkaBrokers) != 1 || cfg.KafkaBrokers[0] != "kafka:9092" {
		t.Fatalf("expected default kafka broker, got %v", cfg.KafkaBrokers)
	}
	if cfg.BookingTopic != defaultBookingTopic {
		t.Fatalf("expected default booking topic, got %q", cfg.BookingTopic)
	}
	if cfg.DLQTopic != defaultDLQTopic {
		t.Fatalf("expected default dlq topic, got %q", cfg.DLQTopic)
	}
	if cfg.ConsumerGroup != defaultConsumerGroup {
		t.Fatalf("expected default consumer group, got %q", cfg.ConsumerGroup)
	}
	if cfg.RetryLimit != defaultRetryLimit {
		t.Fatalf("expected default retry limit, got %d", cfg.RetryLimit)
	}
	if !cfg.NotificationLog {
		t.Fatalf("expected notification log enabled by default")
	}
}

func TestLoadUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("NOTIFICATION_DB_DSN", "postgres://custom")
	t.Setenv("NOTIFICATION_HTTP_HOST", "127.0.0.1")
	t.Setenv("NOTIFICATION_HTTP_PORT", "9095")
	t.Setenv("KAFKA_BROKERS", "kafka-1:9092, kafka-2:9092")
	t.Setenv("BOOKING_EVENTS_TOPIC", "booking.custom")
	t.Setenv("BOOKING_EVENTS_DLQ_TOPIC", "booking.custom.dlq")
	t.Setenv("KAFKA_CONSUMER_GROUP", "group-custom")
	t.Setenv("NOTIFICATION_RETRY_LIMIT", "5")
	t.Setenv("NOTIFICATION_LOG_ENABLED", "false")

	cfg := Load()

	if cfg.HTTPHost != "127.0.0.1" {
		t.Fatalf("expected env http host, got %q", cfg.HTTPHost)
	}
	if cfg.HTTPPort != "9095" {
		t.Fatalf("expected env http port, got %q", cfg.HTTPPort)
	}
	if cfg.DatabaseDSN != "postgres://custom" {
		t.Fatalf("expected env database dsn, got %q", cfg.DatabaseDSN)
	}
	if len(cfg.KafkaBrokers) != 2 || cfg.KafkaBrokers[0] != "kafka-1:9092" || cfg.KafkaBrokers[1] != "kafka-2:9092" {
		t.Fatalf("expected env kafka brokers, got %v", cfg.KafkaBrokers)
	}
	if cfg.BookingTopic != "booking.custom" {
		t.Fatalf("expected env booking topic, got %q", cfg.BookingTopic)
	}
	if cfg.DLQTopic != "booking.custom.dlq" {
		t.Fatalf("expected env dlq topic, got %q", cfg.DLQTopic)
	}
	if cfg.ConsumerGroup != "group-custom" {
		t.Fatalf("expected env consumer group, got %q", cfg.ConsumerGroup)
	}
	if cfg.RetryLimit != 5 {
		t.Fatalf("expected env retry limit, got %d", cfg.RetryLimit)
	}
	if cfg.NotificationLog {
		t.Fatalf("expected env notification log disabled")
	}
}

package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultGRPCHost                  = "0.0.0.0"
	defaultGRPCPort                  = "50051"
	defaultAvailabilityServiceTarget = "availability-service:50051"
	defaultPaymentServiceTarget      = "payment-service:50051"
	defaultExternalCallTimeout       = 3 * time.Second
	defaultKafkaBrokers              = "kafka:9092"
	defaultBookingEventsTopic        = "medconnect.booking.events.v1"
	defaultOutboxDispatcherEnabled   = true
	defaultOutboxBatchSize           = 50
	defaultOutboxPollInterval        = 500 * time.Millisecond
	defaultOutboxInitialRetryDelay   = time.Second
	defaultOutboxMaxRetryDelay       = 30 * time.Second
	defaultOutboxClaimTimeout        = 2 * time.Minute
	defaultOutboxMaxAttempts         = 10
	defaultSagaMaxRetries            = 3
	defaultSagaRetryDelay            = 200 * time.Millisecond
)

type Config struct {
	GRPCHost                  string
	GRPCPort                  string
	DatabaseDSN               string
	AvailabilityServiceTarget string
	PaymentServiceTarget      string
	ExternalCallTimeout       time.Duration
	KafkaBrokers              []string
	BookingEventsTopic        string
	OutboxDispatcherEnabled   bool
	OutboxBatchSize           int
	OutboxPollInterval        time.Duration
	OutboxInitialRetryDelay   time.Duration
	OutboxMaxRetryDelay       time.Duration
	OutboxClaimTimeout        time.Duration
	OutboxMaxAttempts         int
	SagaMaxRetries            int
	SagaRetryDelay            time.Duration
}

func Load() Config {
	return Config{
		GRPCHost:                  envOrDefault("BOOKING_SERVICE_HOST", defaultGRPCHost),
		GRPCPort:                  envOrDefault("BOOKING_SERVICE_PORT", defaultGRPCPort),
		DatabaseDSN:               os.Getenv("BOOKING_DB_DSN"),
		AvailabilityServiceTarget: envOrDefault("AVAILABILITY_SERVICE_TARGET", defaultAvailabilityServiceTarget),
		PaymentServiceTarget:      envOrDefault("PAYMENT_SERVICE_TARGET", defaultPaymentServiceTarget),
		ExternalCallTimeout:       durationOrDefault("BOOKING_EXTERNAL_CALL_TIMEOUT", defaultExternalCallTimeout),
		KafkaBrokers:              csvEnvOrDefault("KAFKA_BROKERS", defaultKafkaBrokers),
		BookingEventsTopic:        envOrDefault("BOOKING_EVENTS_TOPIC", defaultBookingEventsTopic),
		OutboxDispatcherEnabled:   boolOrDefault("OUTBOX_DISPATCHER_ENABLED", defaultOutboxDispatcherEnabled),
		OutboxBatchSize:           positiveIntOrDefault("OUTBOX_BATCH_SIZE", defaultOutboxBatchSize),
		OutboxPollInterval:        durationOrDefault("OUTBOX_POLL_INTERVAL", defaultOutboxPollInterval),
		OutboxInitialRetryDelay:   durationOrDefault("OUTBOX_RETRY_INITIAL_DELAY", defaultOutboxInitialRetryDelay),
		OutboxMaxRetryDelay:       durationOrDefault("OUTBOX_RETRY_MAX_DELAY", defaultOutboxMaxRetryDelay),
		OutboxClaimTimeout:        durationOrDefault("OUTBOX_CLAIM_TIMEOUT", defaultOutboxClaimTimeout),
		OutboxMaxAttempts:         positiveIntOrDefault("OUTBOX_MAX_ATTEMPTS", defaultOutboxMaxAttempts),
		SagaMaxRetries:            nonNegativeIntOrDefault("BOOKING_SAGA_MAX_RETRIES", defaultSagaMaxRetries),
		SagaRetryDelay:            durationOrDefault("BOOKING_SAGA_RETRY_DELAY", defaultSagaRetryDelay),
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func csvEnvOrDefault(key string, fallback string) []string {
	value := envOrDefault(key, fallback)
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func boolOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func positiveIntOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func nonNegativeIntOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

package config

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultGRPCHost                  = "0.0.0.0"
	defaultGRPCPort                  = "50051"
	defaultAvailabilityServiceTarget = "availability-service:50051"
	defaultPaymentServiceTarget      = "payment-service:50051"
	defaultExternalCallTimeout       = 3 * time.Second
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

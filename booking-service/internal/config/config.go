package config

import "os"
import "time"

const (
	defaultGRPCHost                  = "0.0.0.0"
	defaultGRPCPort                  = "50051"
	defaultAvailabilityServiceTarget = "availability-service:50051"
	defaultPaymentServiceTarget      = "payment-service:50051"
	defaultExternalCallTimeout       = 3 * time.Second
)

type Config struct {
	GRPCHost                  string
	GRPCPort                  string
	DatabaseDSN               string
	AvailabilityServiceTarget string
	PaymentServiceTarget      string
	ExternalCallTimeout       time.Duration
}

func Load() Config {
	return Config{
		GRPCHost:                  envOrDefault("BOOKING_SERVICE_HOST", defaultGRPCHost),
		GRPCPort:                  envOrDefault("BOOKING_SERVICE_PORT", defaultGRPCPort),
		DatabaseDSN:               os.Getenv("BOOKING_DB_DSN"),
		AvailabilityServiceTarget: envOrDefault("AVAILABILITY_SERVICE_TARGET", defaultAvailabilityServiceTarget),
		PaymentServiceTarget:      envOrDefault("PAYMENT_SERVICE_TARGET", defaultPaymentServiceTarget),
		ExternalCallTimeout:       durationOrDefault("BOOKING_EXTERNAL_CALL_TIMEOUT", defaultExternalCallTimeout),
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

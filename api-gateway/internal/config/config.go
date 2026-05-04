package config

import (
	"os"
	"time"
)

const (
	defaultHTTPHost             = "0.0.0.0"
	defaultHTTPPort             = "8080"
	defaultBookingServiceTarget = "booking-service:50051"
	defaultRequestTimeout       = 5 * time.Second
)

type Config struct {
	HTTPHost             string
	HTTPPort             string
	BookingServiceTarget string
	RequestTimeout       time.Duration
}

func Load() Config {
	return Config{
		HTTPHost:             envOrDefault("API_GATEWAY_HOST", defaultHTTPHost),
		HTTPPort:             envOrDefault("API_GATEWAY_PORT", defaultHTTPPort),
		BookingServiceTarget: envOrDefault("BOOKING_SERVICE_TARGET", defaultBookingServiceTarget),
		RequestTimeout:       durationOrDefault("API_GATEWAY_REQUEST_TIMEOUT", defaultRequestTimeout),
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
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

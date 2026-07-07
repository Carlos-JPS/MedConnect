package config

import (
	"os"
	"time"
)

const (
	defaultHTTPHost                  = "0.0.0.0"
	defaultHTTPPort                  = "8080"
	defaultBookingServiceTarget      = "booking-service:50051"
	defaultPaymentServiceTarget      = "payment-service:50051"
	defaultAvailabilityServiceTarget = "availability-service:50051"
	defaultAuthServiceTarget         = "auth-service:50051"
	defaultNotificationServiceURL    = "http://notification-service:8081"
	defaultRequestTimeout            = 5 * time.Second
)

type Config struct {
	HTTPHost                  string
	HTTPPort                  string
	BookingServiceTarget      string
	PaymentServiceTarget      string
	AvailabilityServiceTarget string
	AuthServiceTarget         string
	NotificationServiceURL    string
	RequestTimeout            time.Duration
}

func Load() Config {
	return Config{
		HTTPHost:                  envOrDefault("API_GATEWAY_HOST", defaultHTTPHost),
		HTTPPort:                  envOrDefault("API_GATEWAY_PORT", defaultHTTPPort),
		BookingServiceTarget:      envOrDefault("BOOKING_SERVICE_TARGET", defaultBookingServiceTarget),
		PaymentServiceTarget:      envOrDefault("PAYMENT_SERVICE_TARGET", defaultPaymentServiceTarget),
		AvailabilityServiceTarget: envOrDefault("AVAILABILITY_SERVICE_TARGET", defaultAvailabilityServiceTarget),
		AuthServiceTarget:         envOrDefault("AUTH_SERVICE_TARGET", defaultAuthServiceTarget),
		NotificationServiceURL:    envOrDefault("NOTIFICATION_SERVICE_URL", defaultNotificationServiceURL),
		RequestTimeout:            durationOrDefault("API_GATEWAY_REQUEST_TIMEOUT", defaultRequestTimeout),
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

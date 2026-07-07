package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultHTTPHost        = "0.0.0.0"
	defaultHTTPPort        = "8081"
	defaultDatabaseDSN     = "postgres://notification:notification_password@notification_db:5432/notification_db?sslmode=disable"
	defaultKafkaBrokers    = "kafka:9092"
	defaultBookingTopic    = "medconnect.booking.events.v1"
	defaultDLQTopic        = "medconnect.booking.events.dlq.v1"
	defaultConsumerGroup   = "medconnect-notification-service-v1"
	defaultRetryLimit      = 3
	defaultNotificationLog = true
)

type Config struct {
	HTTPHost        string
	HTTPPort        string
	DatabaseDSN     string
	KafkaBrokers    []string
	BookingTopic    string
	DLQTopic        string
	ConsumerGroup   string
	RetryLimit      int
	NotificationLog bool
}

func Load() Config {
	return Config{
		HTTPHost:        envOrDefault("NOTIFICATION_HTTP_HOST", defaultHTTPHost),
		HTTPPort:        envOrDefault("NOTIFICATION_HTTP_PORT", defaultHTTPPort),
		DatabaseDSN:     envOrDefault("NOTIFICATION_DB_DSN", defaultDatabaseDSN),
		KafkaBrokers:    csvEnvOrDefault("KAFKA_BROKERS", defaultKafkaBrokers),
		BookingTopic:    envOrDefault("BOOKING_EVENTS_TOPIC", defaultBookingTopic),
		DLQTopic:        envOrDefault("BOOKING_EVENTS_DLQ_TOPIC", defaultDLQTopic),
		ConsumerGroup:   envOrDefault("KAFKA_CONSUMER_GROUP", defaultConsumerGroup),
		RetryLimit:      intOrDefault("NOTIFICATION_RETRY_LIMIT", defaultRetryLimit),
		NotificationLog: boolOrDefault("NOTIFICATION_LOG_ENABLED", defaultNotificationLog),
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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

func intOrDefault(key string, fallback int) int {
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

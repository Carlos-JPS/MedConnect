package config

import "os"

const (
	defaultGRPCHost = "0.0.0.0"
	defaultGRPCPort = "50051"
)

type Config struct {
	GRPCHost    string
	GRPCPort    string
	DatabaseDSN string
}

func Load() Config {
	return Config{
		GRPCHost:    envOrDefault("BOOKING_SERVICE_HOST", defaultGRPCHost),
		GRPCPort:    envOrDefault("BOOKING_SERVICE_PORT", defaultGRPCPort),
		DatabaseDSN: os.Getenv("BOOKING_DB_DSN"),
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

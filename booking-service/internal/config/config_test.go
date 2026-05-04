package config

import "testing"

func TestLoadUsesDefaultsWhenEnvironmentIsMissing(t *testing.T) {
	t.Setenv("BOOKING_SERVICE_HOST", "")
	t.Setenv("BOOKING_SERVICE_PORT", "")
	t.Setenv("BOOKING_DB_DSN", "")

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
}

func TestLoadUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("BOOKING_SERVICE_HOST", "127.0.0.1")
	t.Setenv("BOOKING_SERVICE_PORT", "60061")
	t.Setenv("BOOKING_DB_DSN", "postgres://booking:secret@db:5432/booking")

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
}

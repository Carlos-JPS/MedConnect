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
	t.Setenv("BOOKING_SAGA_MAX_RETRIES", "")
	t.Setenv("BOOKING_SAGA_RETRY_DELAY", "")

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
	if cfg.SagaMaxRetries != 3 {
		t.Fatalf("expected default saga max retries 3, got %d", cfg.SagaMaxRetries)
	}
	if cfg.SagaRetryDelay != 200*time.Millisecond {
		t.Fatalf("expected default saga retry delay 200ms, got %s", cfg.SagaRetryDelay)
	}
}

func TestLoadUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("BOOKING_SERVICE_HOST", "127.0.0.1")
	t.Setenv("BOOKING_SERVICE_PORT", "60061")
	t.Setenv("BOOKING_DB_DSN", "postgres://booking:secret@db:5432/booking")
	t.Setenv("AVAILABILITY_SERVICE_TARGET", "availability:6000")
	t.Setenv("PAYMENT_SERVICE_TARGET", "payment:7000")
	t.Setenv("BOOKING_EXTERNAL_CALL_TIMEOUT", "1500ms")
	t.Setenv("BOOKING_SAGA_MAX_RETRIES", "5")
	t.Setenv("BOOKING_SAGA_RETRY_DELAY", "750ms")

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
	if cfg.SagaMaxRetries != 5 {
		t.Fatalf("expected env saga max retries, got %d", cfg.SagaMaxRetries)
	}
	if cfg.SagaRetryDelay != 750*time.Millisecond {
		t.Fatalf("expected env saga retry delay, got %s", cfg.SagaRetryDelay)
	}
}

func TestLoadUsesSagaRetryDefaultsWhenOverridesAreInvalid(t *testing.T) {
	t.Setenv("BOOKING_SAGA_MAX_RETRIES", "-1")
	t.Setenv("BOOKING_SAGA_RETRY_DELAY", "not-a-duration")

	cfg := Load()

	if cfg.SagaMaxRetries != 3 {
		t.Fatalf("expected default saga max retries for invalid env, got %d", cfg.SagaMaxRetries)
	}
	if cfg.SagaRetryDelay != 200*time.Millisecond {
		t.Fatalf("expected default saga retry delay for invalid env, got %s", cfg.SagaRetryDelay)
	}
}

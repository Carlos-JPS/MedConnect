package processor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MedConnect/notification-service/internal/domain"
)

type fakeStore struct {
	notifications []domain.Notification
	inserted      bool
	err           error
	calls         int
}

func (s *fakeStore) SaveNotification(_ context.Context, notification domain.Notification) (bool, error) {
	s.calls++
	if s.err != nil {
		return false, s.err
	}
	s.notifications = append(s.notifications, notification)
	return s.inserted, nil
}

type fakeDLQ struct {
	messages []Message
	reasons  []string
	err      error
}

func (d *fakeDLQ) Publish(_ context.Context, message Message, reason string) error {
	if d.err != nil {
		return d.err
	}
	d.messages = append(d.messages, message)
	d.reasons = append(d.reasons, reason)
	return nil
}

func TestProcessValidBookingEventSavesNotification(t *testing.T) {
	store := &fakeStore{inserted: true}
	dlq := &fakeDLQ{}
	processor := New(store, dlq, Config{RetryLimit: 3}, nil)
	now := time.Date(2026, time.May, 3, 23, 30, 0, 0, time.UTC)
	processor.clock = func() time.Time { return now }

	err := processor.Process(context.Background(), Message{
		Topic: "medconnect.booking.events.v1",
		Key:   []byte("22222222-2222-2222-2222-222222222222"),
		Value: []byte(validBookingEventJSON()),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(store.notifications) != 1 {
		t.Fatalf("expected one notification, got %d", len(store.notifications))
	}
	notification := store.notifications[0]
	if notification.EventID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected event id to be persisted, got %q", notification.EventID)
	}
	if notification.RecipientID != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("expected patient as recipient, got %q", notification.RecipientID)
	}
	if notification.CreatedAt != now {
		t.Fatalf("expected processor clock to set created_at")
	}
	if len(dlq.messages) != 0 {
		t.Fatalf("expected no DLQ messages, got %d", len(dlq.messages))
	}
}

func TestProcessInvalidBookingEventPublishesDLQ(t *testing.T) {
	store := &fakeStore{}
	dlq := &fakeDLQ{}
	processor := New(store, dlq, Config{RetryLimit: 3}, nil)

	err := processor.Process(context.Background(), Message{
		Topic: "medconnect.booking.events.v1",
		Value: []byte(`{"schema_version":99}`),
	})
	if err != nil {
		t.Fatalf("expected invalid event to be handled through DLQ, got %v", err)
	}
	if store.calls != 0 {
		t.Fatalf("expected invalid event not to be persisted")
	}
	if len(dlq.messages) != 1 {
		t.Fatalf("expected one DLQ message, got %d", len(dlq.messages))
	}
	if !strings.Contains(dlq.reasons[0], "event_id") {
		t.Fatalf("expected DLQ reason to mention validation error, got %q", dlq.reasons[0])
	}
}

func TestProcessPublishesDLQAfterPersistenceRetries(t *testing.T) {
	store := &fakeStore{err: errors.New("postgres down")}
	dlq := &fakeDLQ{}
	processor := New(store, dlq, Config{RetryLimit: 2}, nil)
	processor.sleep = func(time.Duration) {}

	err := processor.Process(context.Background(), Message{
		Topic: "medconnect.booking.events.v1",
		Value: []byte(validBookingEventJSON()),
	})
	if err != nil {
		t.Fatalf("expected exhausted persistence errors to be handled through DLQ, got %v", err)
	}
	if store.calls != 3 {
		t.Fatalf("expected initial try plus two retries, got %d calls", store.calls)
	}
	if len(dlq.messages) != 1 {
		t.Fatalf("expected one DLQ message, got %d", len(dlq.messages))
	}
	if !strings.Contains(dlq.reasons[0], "postgres down") {
		t.Fatalf("expected DLQ reason to include persistence error, got %q", dlq.reasons[0])
	}
}

func validBookingEventJSON() string {
	return `{
		"event_id":"11111111-1111-1111-1111-111111111111",
		"event_type":"booking.created",
		"schema_version":1,
		"occurred_at":"2026-05-03T23:00:00Z",
		"aggregate_id":"22222222-2222-2222-2222-222222222222",
		"payload":{
			"booking_id":"22222222-2222-2222-2222-222222222222",
			"patient_id":"33333333-3333-3333-3333-333333333333",
			"doctor_id":"44444444-4444-4444-4444-444444444444",
			"slot_id":"55555555-5555-5555-5555-555555555555",
			"status":"PENDING_PAYMENT",
			"event_payload":{"status":"PENDING_PAYMENT"}
		}
	}`
}

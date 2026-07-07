package outbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	events        []Event
	publishedIDs  []string
	failedIDs     []string
	nextAttemptAt time.Time
	lastError     string
	maxAttempts   int
}

func (s *fakeStore) FetchPendingOutboxEvents(_ context.Context, _ int, _ time.Duration) ([]Event, error) {
	return s.events, nil
}

func (s *fakeStore) MarkOutboxPublished(_ context.Context, eventID string, _ time.Time) error {
	s.publishedIDs = append(s.publishedIDs, eventID)
	return nil
}

func (s *fakeStore) MarkOutboxFailed(_ context.Context, eventID string, nextAttemptAt time.Time, lastError string, maxAttempts int) error {
	s.failedIDs = append(s.failedIDs, eventID)
	s.nextAttemptAt = nextAttemptAt
	s.lastError = lastError
	s.maxAttempts = maxAttempts
	return nil
}

type fakePublisher struct {
	published []Event
	err       error
}

func (p *fakePublisher) Publish(_ context.Context, event Event) error {
	p.published = append(p.published, event)
	return p.err
}

func TestDispatchBatchPublishesAndMarksEvent(t *testing.T) {
	store := &fakeStore{events: []Event{{EventID: "event-1", Topic: "topic", EventKey: "booking-1"}}}
	publisher := &fakePublisher{}
	dispatcher := NewDispatcher(store, publisher, DispatcherConfig{BatchSize: 10}, nil)

	processed, err := dispatcher.DispatchBatch(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed event, got %d", processed)
	}
	if len(publisher.published) != 1 || publisher.published[0].EventID != "event-1" {
		t.Fatalf("expected event to be published, got %+v", publisher.published)
	}
	if len(store.publishedIDs) != 1 || store.publishedIDs[0] != "event-1" {
		t.Fatalf("expected event to be marked published, got %v", store.publishedIDs)
	}
}

func TestDispatchBatchSchedulesRetryOnPublishError(t *testing.T) {
	now := time.Date(2026, time.May, 3, 22, 0, 0, 0, time.UTC)
	store := &fakeStore{events: []Event{{EventID: "event-1", Attempts: 1}}}
	publisher := &fakePublisher{err: errors.New("kafka down")}
	dispatcher := NewDispatcher(
		store,
		publisher,
		DispatcherConfig{BatchSize: 10, InitialRetryDelay: time.Second, MaxRetryDelay: 30 * time.Second, MaxAttempts: 5},
		nil,
	)
	dispatcher.clock = func() time.Time { return now }

	processed, err := dispatcher.DispatchBatch(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed event, got %d", processed)
	}
	if len(store.failedIDs) != 1 || store.failedIDs[0] != "event-1" {
		t.Fatalf("expected failed event to be marked, got %v", store.failedIDs)
	}
	expectedNextAttempt := now.Add(2 * time.Second)
	if !store.nextAttemptAt.Equal(expectedNextAttempt) {
		t.Fatalf("expected next attempt at %s, got %s", expectedNextAttempt, store.nextAttemptAt)
	}
	if store.lastError != "kafka down" {
		t.Fatalf("expected last error to be stored, got %q", store.lastError)
	}
	if store.maxAttempts != 5 {
		t.Fatalf("expected max attempts to be passed to store, got %d", store.maxAttempts)
	}
}

func TestDispatchBatchMarksFinalFailureAtMaxAttempts(t *testing.T) {
	now := time.Date(2026, time.May, 3, 22, 0, 0, 0, time.UTC)
	store := &fakeStore{events: []Event{{EventID: "event-1", Attempts: 2}}}
	publisher := &fakePublisher{err: errors.New("kafka down")}
	dispatcher := NewDispatcher(
		store,
		publisher,
		DispatcherConfig{BatchSize: 10, InitialRetryDelay: time.Second, MaxRetryDelay: 30 * time.Second, MaxAttempts: 3},
		nil,
	)
	dispatcher.clock = func() time.Time { return now }

	processed, err := dispatcher.DispatchBatch(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed event, got %d", processed)
	}
	if len(store.failedIDs) != 1 || store.failedIDs[0] != "event-1" {
		t.Fatalf("expected failed event to be marked, got %v", store.failedIDs)
	}
	if store.maxAttempts != 3 {
		t.Fatalf("expected max attempts 3, got %d", store.maxAttempts)
	}
}

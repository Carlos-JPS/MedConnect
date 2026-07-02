package outbox

import (
	"context"
	"time"
)

type Event struct {
	EventID     string
	AggregateID string
	EventType   string
	Topic       string
	EventKey    string
	Payload     []byte
	Attempts    int
	CreatedAt   time.Time
}

type Store interface {
	FetchPendingOutboxEvents(ctx context.Context, limit int) ([]Event, error)
	MarkOutboxPublished(ctx context.Context, eventID string, publishedAt time.Time) error
	MarkOutboxFailed(ctx context.Context, eventID string, nextAttemptAt time.Time, lastError string) error
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

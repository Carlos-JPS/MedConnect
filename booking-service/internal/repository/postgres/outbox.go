package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MedConnect/booking-service/internal/outbox"
	"github.com/MedConnect/booking-service/internal/service"
)

const defaultBookingEventsTopic = "medconnect.booking.events.v1"

type bookingEventEnvelope struct {
	EventID       string         `json:"event_id"`
	EventType     string         `json:"event_type"`
	SchemaVersion int            `json:"schema_version"`
	OccurredAt    time.Time      `json:"occurred_at"`
	AggregateID   string         `json:"aggregate_id"`
	Payload       map[string]any `json:"payload"`
}

func (r *Repository) insertOutboxEvent(ctx context.Context, tx *sql.Tx, booking service.Booking, event service.BookingEvent) error {
	kafkaEventType, ok := eventTypeToKafka(event.EventType)
	if !ok || event.EventID == "" {
		return nil
	}

	payload, err := buildBookingOutboxPayload(booking, event, kafkaEventType)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO outbox_events (id, aggregate_id, event_type, topic, event_key, payload, created_at, next_attempt_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		event.EventID,
		booking.BookingID,
		kafkaEventType,
		r.bookingEventsTopic,
		booking.BookingID,
		payload,
		event.CreatedAt,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insertar evento outbox: %w", err)
	}

	return nil
}

func buildBookingOutboxPayload(booking service.Booking, event service.BookingEvent, kafkaEventType string) ([]byte, error) {
	eventPayload := map[string]any{}
	if len(event.Payload) > 0 {
		if err := json.Unmarshal(event.Payload, &eventPayload); err != nil {
			return nil, fmt.Errorf("decodificar payload de appointment_event: %w", err)
		}
	}

	payload := map[string]any{
		"booking_id":     booking.BookingID,
		"patient_id":     booking.PatientID,
		"doctor_id":      booking.DoctorID,
		"slot_id":        booking.SlotID,
		"status":         booking.Status.String(),
		"reserved_until": booking.ReservedUntil,
		"event_payload":  eventPayload,
	}
	if booking.PaymentID != "" {
		payload["payment_id"] = booking.PaymentID
	}
	if booking.ConfirmationCode != "" {
		payload["confirmation_code"] = booking.ConfirmationCode
	}
	if booking.ConfirmedAt != nil {
		payload["confirmed_at"] = booking.ConfirmedAt
	}
	if booking.CancelledAt != nil {
		payload["cancelled_at"] = booking.CancelledAt
	}

	envelope := bookingEventEnvelope{
		EventID:       event.EventID,
		EventType:     kafkaEventType,
		SchemaVersion: 1,
		OccurredAt:    event.CreatedAt,
		AggregateID:   booking.BookingID,
		Payload:       payload,
	}

	return json.Marshal(envelope)
}

func eventTypeToKafka(eventType service.EventType) (string, bool) {
	switch eventType {
	case service.EventCreated:
		return "booking.created", true
	case service.EventConfirmed:
		return "booking.confirmed", true
	case service.EventCancelled:
		return "booking.cancelled", true
	case service.EventExpired:
		return "booking.expired", true
	default:
		return "", false
	}
}

func (r *Repository) FetchPendingOutboxEvents(ctx context.Context, limit int, claimTimeout time.Duration) ([]outbox.Event, error) {
	if limit <= 0 {
		limit = 50
	}
	claimTimeoutMillis := durationMillis(claimTimeout, 2*time.Minute)

	rows, err := r.db.QueryContext(
		ctx,
		`WITH candidates AS (
    SELECT id
    FROM outbox_events
    WHERE status = 'PENDING'
      AND published_at IS NULL
      AND next_attempt_at <= NOW()
      AND (locked_until IS NULL OR locked_until <= NOW())
    ORDER BY created_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT $1
), claimed AS (
    UPDATE outbox_events AS o
    SET locked_until = NOW() + ($2 * INTERVAL '1 millisecond')
    FROM candidates
    WHERE o.id = candidates.id
    RETURNING o.id::text, o.aggregate_id::text, o.event_type, o.topic, o.event_key, o.payload::text, o.attempts, o.created_at
)
SELECT id, aggregate_id, event_type, topic, event_key, payload, attempts, created_at
FROM claimed
ORDER BY created_at ASC`,
		limit,
		claimTimeoutMillis,
	)
	if err != nil {
		return nil, fmt.Errorf("reclamar outbox_events pendientes: %w", err)
	}
	defer rows.Close()

	var events []outbox.Event
	for rows.Next() {
		var event outbox.Event
		var payload string
		if err := rows.Scan(
			&event.EventID,
			&event.AggregateID,
			&event.EventType,
			&event.Topic,
			&event.EventKey,
			&payload,
			&event.Attempts,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("leer outbox_event: %w", err)
		}
		event.Payload = []byte(payload)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar outbox_events: %w", err)
	}

	return events, nil
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE outbox_events
SET status = 'PUBLISHED', published_at = $2, locked_until = NULL, last_error = NULL
WHERE id = $1 AND status = 'PENDING'`,
		eventID,
		publishedAt,
	)
	if err != nil {
		return fmt.Errorf("marcar outbox_event publicado: %w", err)
	}
	return nil
}

func (r *Repository) MarkOutboxFailed(ctx context.Context, eventID string, nextAttemptAt time.Time, lastError string, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE outbox_events
SET attempts = attempts + 1,
    status = CASE WHEN attempts + 1 >= $4 THEN 'FAILED' ELSE 'PENDING' END,
    next_attempt_at = $2,
    locked_until = NULL,
    failed_at = CASE WHEN attempts + 1 >= $4 THEN NOW() ELSE NULL END,
    last_error = $3
WHERE id = $1 AND status = 'PENDING' AND published_at IS NULL`,
		eventID,
		nextAttemptAt,
		truncateError(lastError),
		maxAttempts,
	)
	if err != nil {
		return fmt.Errorf("marcar outbox_event fallido: %w", err)
	}
	return nil
}

func (r *Repository) OutboxStats(ctx context.Context) (outbox.Stats, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT
    COUNT(*) FILTER (WHERE status = 'PENDING' AND published_at IS NULL),
    COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at) FILTER (WHERE status = 'PENDING' AND published_at IS NULL))), 0),
    COUNT(*) FILTER (WHERE status = 'FAILED')
FROM outbox_events`,
	)

	var stats outbox.Stats
	var oldestPendingAgeSeconds float64
	if err := row.Scan(&stats.PendingEvents, &oldestPendingAgeSeconds, &stats.FinalFailedEvents); err != nil {
		return outbox.Stats{}, fmt.Errorf("consultar metricas outbox: %w", err)
	}
	stats.OldestPendingAge = time.Duration(oldestPendingAgeSeconds * float64(time.Second))
	return stats, nil
}

func durationMillis(value time.Duration, fallback time.Duration) int64 {
	if value <= 0 {
		value = fallback
	}
	millis := value.Milliseconds()
	if millis <= 0 {
		return 1
	}
	return millis
}

func truncateError(value string) string {
	const maxLength = 1000
	if len(value) <= maxLength {
		return value
	}
	return value[:maxLength]
}

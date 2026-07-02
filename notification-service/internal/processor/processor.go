package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MedConnect/notification-service/internal/domain"
)

const (
	channelInApp = "IN_APP"
	statusSent   = "SENT"
)

type Message struct {
	Topic string
	Key   []byte
	Value []byte
}

type Store interface {
	SaveNotification(ctx context.Context, notification domain.Notification) (bool, error)
}

type DLQPublisher interface {
	Publish(ctx context.Context, message Message, reason string) error
}

type Config struct {
	RetryLimit      int
	NotificationLog bool
}

type Processor struct {
	store           Store
	dlq             DLQPublisher
	retryLimit      int
	notificationLog bool
	logger          *log.Logger
	sleep           func(time.Duration)
	clock           func() time.Time
}

type bookingEvent struct {
	EventID       string         `json:"event_id"`
	EventType     string         `json:"event_type"`
	SchemaVersion int            `json:"schema_version"`
	OccurredAt    time.Time      `json:"occurred_at"`
	AggregateID   string         `json:"aggregate_id"`
	Payload       bookingPayload `json:"payload"`
}

type bookingPayload struct {
	BookingID        string         `json:"booking_id"`
	PatientID        string         `json:"patient_id"`
	DoctorID         string         `json:"doctor_id"`
	SlotID           string         `json:"slot_id"`
	Status           string         `json:"status"`
	PaymentID        string         `json:"payment_id"`
	ConfirmationCode string         `json:"confirmation_code"`
	EventPayload     map[string]any `json:"event_payload"`
}

func New(store Store, dlq DLQPublisher, cfg Config, logger *log.Logger) *Processor {
	if cfg.RetryLimit < 0 {
		cfg.RetryLimit = 0
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Processor{
		store:           store,
		dlq:             dlq,
		retryLimit:      cfg.RetryLimit,
		notificationLog: cfg.NotificationLog,
		logger:          logger,
		sleep:           time.Sleep,
		clock:           func() time.Time { return time.Now().UTC() },
	}
}

func (p *Processor) Process(ctx context.Context, message Message) error {
	event, err := decodeAndValidate(message.Value)
	if err != nil {
		return p.publishDLQ(ctx, message, err)
	}

	notification, err := p.buildNotification(event, message.Value)
	if err != nil {
		return p.publishDLQ(ctx, message, err)
	}

	var lastErr error
	for attempt := 0; attempt <= p.retryLimit; attempt++ {
		inserted, err := p.store.SaveNotification(ctx, notification)
		if err == nil {
			if p.notificationLog && inserted {
				p.logger.Printf("notification-service: notificacion generada event_id=%s booking_id=%s type=%s", notification.EventID, notification.BookingID, notification.EventType)
			}
			if p.notificationLog && !inserted {
				p.logger.Printf("notification-service: evento duplicado ignorado event_id=%s", notification.EventID)
			}
			return nil
		}
		lastErr = err
		if attempt < p.retryLimit {
			p.sleep(retryDelay(attempt))
		}
	}

	return p.publishDLQ(ctx, message, fmt.Errorf("persistir notificacion tras %d reintentos: %w", p.retryLimit, lastErr))
}

func decodeAndValidate(value []byte) (bookingEvent, error) {
	var event bookingEvent
	if err := json.Unmarshal(value, &event); err != nil {
		return bookingEvent{}, fmt.Errorf("payload JSON invalido: %w", err)
	}
	if event.EventID == "" {
		return bookingEvent{}, errors.New("event_id es obligatorio")
	}
	if event.AggregateID == "" {
		return bookingEvent{}, errors.New("aggregate_id es obligatorio")
	}
	if event.SchemaVersion != 1 {
		return bookingEvent{}, fmt.Errorf("schema_version no soportada: %d", event.SchemaVersion)
	}
	if !isSupportedEventType(event.EventType) {
		return bookingEvent{}, fmt.Errorf("event_type no soportado: %s", event.EventType)
	}
	if event.Payload.PatientID == "" {
		return bookingEvent{}, errors.New("payload.patient_id es obligatorio")
	}
	if event.Payload.BookingID == "" {
		event.Payload.BookingID = event.AggregateID
	}
	return event, nil
}

func (p *Processor) buildNotification(event bookingEvent, rawPayload []byte) (domain.Notification, error) {
	message, err := notificationMessage(event)
	if err != nil {
		return domain.Notification{}, err
	}
	return domain.Notification{
		EventID:     event.EventID,
		BookingID:   event.Payload.BookingID,
		EventType:   event.EventType,
		RecipientID: event.Payload.PatientID,
		Channel:     channelInApp,
		Message:     message,
		Status:      statusSent,
		Payload:     rawPayload,
		CreatedAt:   p.clock(),
	}, nil
}

func notificationMessage(event bookingEvent) (string, error) {
	switch event.EventType {
	case "booking.created":
		return fmt.Sprintf("Reserva %s creada y pendiente de pago para el slot %s.", event.Payload.BookingID, event.Payload.SlotID), nil
	case "booking.confirmed":
		return fmt.Sprintf("Reserva %s confirmada. Codigo: %s.", event.Payload.BookingID, event.Payload.ConfirmationCode), nil
	case "booking.cancelled":
		return fmt.Sprintf("Reserva %s cancelada.", event.Payload.BookingID), nil
	case "booking.expired":
		return fmt.Sprintf("Reserva %s expirada.", event.Payload.BookingID), nil
	default:
		return "", fmt.Errorf("event_type no soportado: %s", event.EventType)
	}
}

func isSupportedEventType(eventType string) bool {
	switch eventType {
	case "booking.created", "booking.confirmed", "booking.cancelled", "booking.expired":
		return true
	default:
		return false
	}
}

func (p *Processor) publishDLQ(ctx context.Context, message Message, err error) error {
	if p.dlq == nil {
		return err
	}
	if publishErr := p.dlq.Publish(ctx, message, err.Error()); publishErr != nil {
		return fmt.Errorf("publicar en DLQ: %w", publishErr)
	}
	p.logger.Printf("notification-service: mensaje enviado a DLQ: %v", err)
	return nil
}

func retryDelay(attempt int) time.Duration {
	delay := time.Second
	for i := 0; i < attempt; i++ {
		delay *= 2
	}
	return delay
}

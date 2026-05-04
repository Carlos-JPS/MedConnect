package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MedConnect/booking-service/internal/service"
	_ "github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func Open(dsn string) (*Repository, error) {
	if dsn == "" {
		return nil, errors.New("BOOKING_DB_DSN no esta configurado")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir conexion PostgreSQL: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("conectar a PostgreSQL: %w", err)
	}

	return NewRepositoryFromDB(db), nil
}

func NewRepositoryFromDB(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *Repository) CreateBooking(ctx context.Context, booking service.Booking, event service.BookingEvent) (service.Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return service.Booking{}, fmt.Errorf("iniciar transaccion booking: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO appointments (id, patient_id, doctor_id, slot_id, status, payment_id, confirmation_code, notes, created_at, updated_at, reserved_until, confirmed_at, cancelled_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		booking.BookingID,
		booking.PatientID,
		booking.DoctorID,
		booking.SlotID,
		statusToDB(booking.Status),
		nullableString(booking.PaymentID),
		nullableString(booking.ConfirmationCode),
		booking.Notes,
		booking.CreatedAt,
		booking.UpdatedAt,
		booking.ReservedUntil,
		nullableTime(booking.ConfirmedAt),
		nullableTime(booking.CancelledAt),
	)
	if err != nil {
		return service.Booking{}, fmt.Errorf("insertar appointment: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO appointment_events (id, appointment_id, event_type, payload, created_at)
VALUES ($1, $2, $3, $4, $5)`,
		event.EventID,
		event.BookingID,
		eventTypeToDB(event.EventType),
		event.Payload,
		event.CreatedAt,
	)
	if err != nil {
		return service.Booking{}, fmt.Errorf("insertar appointment_event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return service.Booking{}, fmt.Errorf("confirmar transaccion booking: %w", err)
	}
	committed = true

	return booking, nil
}

func (r *Repository) GetBooking(ctx context.Context, bookingID string) (service.Booking, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, patient_id, doctor_id, slot_id, status, payment_id, confirmation_code, notes, created_at, updated_at, reserved_until, confirmed_at, cancelled_at FROM appointments WHERE id = $1`,
		bookingID,
	)

	booking, err := scanBooking(row)
	if err != nil {
		return service.Booking{}, err
	}
	return booking, nil
}

func (r *Repository) ListBookingEvents(ctx context.Context, bookingID string) ([]service.BookingEvent, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, appointment_id, event_type, payload, created_at FROM appointment_events WHERE appointment_id = $1 ORDER BY created_at ASC`,
		bookingID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar appointment_events: %w", err)
	}
	defer rows.Close()

	var events []service.BookingEvent
	for rows.Next() {
		event, err := scanBookingEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar appointment_events: %w", err)
	}

	return events, nil
}

func (r *Repository) ListBookingsByPatient(ctx context.Context, patientID string, status service.Status) ([]service.Booking, error) {
	query := `SELECT id, patient_id, doctor_id, slot_id, status, payment_id, confirmation_code, notes, created_at, updated_at, reserved_until, confirmed_at, cancelled_at FROM appointments WHERE patient_id = $1 ORDER BY created_at DESC`
	args := []any{patientID}
	if status != service.StatusUnspecified {
		query = `SELECT id, patient_id, doctor_id, slot_id, status, payment_id, confirmation_code, notes, created_at, updated_at, reserved_until, confirmed_at, cancelled_at FROM appointments WHERE patient_id = $1 AND status = $2 ORDER BY created_at DESC`
		args = append(args, statusToDB(status))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar appointments por paciente: %w", err)
	}
	defer rows.Close()

	var bookings []service.Booking
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar appointments: %w", err)
	}

	return bookings, nil
}

func (r *Repository) UpdateBookingStatus(ctx context.Context, input service.UpdateBookingStatusInput) (service.Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return service.Booking{}, fmt.Errorf("iniciar transaccion update booking: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRowContext(
		ctx,
		`UPDATE appointments SET status = $2, payment_id = $3, confirmation_code = $4, updated_at = $5, confirmed_at = $6, cancelled_at = $7 WHERE id = $1 RETURNING id, patient_id, doctor_id, slot_id, status, payment_id, confirmation_code, notes, created_at, updated_at, reserved_until, confirmed_at, cancelled_at`,
		input.BookingID,
		statusToDB(input.Status),
		nullableString(input.PaymentID),
		nullableString(input.ConfirmationCode),
		input.UpdatedAt,
		nullableTime(input.ConfirmedAt),
		nullableTime(input.CancelledAt),
	)
	booking, err := scanBooking(row)
	if err != nil {
		return service.Booking{}, err
	}

	if input.Event.EventID != "" {
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO appointment_events (id, appointment_id, event_type, payload, created_at)
VALUES ($1, $2, $3, $4, $5)`,
			input.Event.EventID,
			input.Event.BookingID,
			eventTypeToDB(input.Event.EventType),
			input.Event.Payload,
			input.Event.CreatedAt,
		)
		if err != nil {
			return service.Booking{}, fmt.Errorf("insertar appointment_event de update: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return service.Booking{}, fmt.Errorf("confirmar transaccion update booking: %w", err)
	}
	committed = true

	return booking, nil
}

func statusToDB(status service.Status) string {
	switch status {
	case service.StatusPendingPayment:
		return "PENDING_PAYMENT"
	case service.StatusConfirmed:
		return "CONFIRMED"
	case service.StatusCancelled:
		return "CANCELLED"
	case service.StatusExpired:
		return "EXPIRED"
	default:
		return "UNSPECIFIED"
	}
}

func statusFromDB(status string) service.Status {
	switch status {
	case "PENDING_PAYMENT":
		return service.StatusPendingPayment
	case "CONFIRMED":
		return service.StatusConfirmed
	case "CANCELLED":
		return service.StatusCancelled
	case "EXPIRED":
		return service.StatusExpired
	default:
		return service.StatusUnspecified
	}
}

func eventTypeToDB(eventType service.EventType) string {
	switch eventType {
	case service.EventCreated:
		return "CREATED"
	case service.EventPaymentApproved:
		return "PAYMENT_APPROVED"
	case service.EventConfirmed:
		return "CONFIRMED"
	case service.EventCancelled:
		return "CANCELLED"
	case service.EventExpired:
		return "EXPIRED"
	default:
		return "UNSPECIFIED"
	}
}

func eventTypeFromDB(eventType string) service.EventType {
	switch eventType {
	case "CREATED":
		return service.EventCreated
	case "PAYMENT_APPROVED":
		return service.EventPaymentApproved
	case "CONFIRMED":
		return service.EventConfirmed
	case "CANCELLED":
		return service.EventCancelled
	case "EXPIRED":
		return service.EventExpired
	default:
		return service.EventUnspecified
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBooking(row rowScanner) (service.Booking, error) {
	var booking service.Booking
	var status string
	var paymentID sql.NullString
	var confirmationCode sql.NullString
	var confirmedAt sql.NullTime
	var cancelledAt sql.NullTime

	err := row.Scan(
		&booking.BookingID,
		&booking.PatientID,
		&booking.DoctorID,
		&booking.SlotID,
		&status,
		&paymentID,
		&confirmationCode,
		&booking.Notes,
		&booking.CreatedAt,
		&booking.UpdatedAt,
		&booking.ReservedUntil,
		&confirmedAt,
		&cancelledAt,
	)
	if err != nil {
		return service.Booking{}, fmt.Errorf("leer appointment: %w", err)
	}

	booking.Status = statusFromDB(status)
	booking.PaymentID = stringFromNull(paymentID)
	booking.ConfirmationCode = stringFromNull(confirmationCode)
	booking.ConfirmedAt = timeFromNull(confirmedAt)
	booking.CancelledAt = timeFromNull(cancelledAt)

	return booking, nil
}

func scanBookingEvent(row rowScanner) (service.BookingEvent, error) {
	var event service.BookingEvent
	var eventType string

	err := row.Scan(&event.EventID, &event.BookingID, &eventType, &event.Payload, &event.CreatedAt)
	if err != nil {
		return service.BookingEvent{}, fmt.Errorf("leer appointment_event: %w", err)
	}
	event.EventType = eventTypeFromDB(eventType)

	return event, nil
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func stringFromNull(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func timeFromNull(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

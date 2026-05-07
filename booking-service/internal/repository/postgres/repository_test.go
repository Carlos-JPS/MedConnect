package postgres

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/MedConnect/booking-service/internal/service"
)

func TestCreateBookingPersistsAppointmentAndEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := NewRepositoryFromDB(db)
	now := time.Date(2026, time.May, 3, 20, 0, 0, 0, time.UTC)
	booking := service.Booking{
		BookingID:     "booking-1",
		PatientID:     "patient-1",
		DoctorID:      "doctor-1",
		SlotID:        "slot-1",
		Status:        service.StatusPendingPayment,
		Notes:         "Control anual",
		CreatedAt:     now,
		UpdatedAt:     now,
		ReservedUntil: now.Add(15 * time.Minute),
	}
	event := service.BookingEvent{
		EventID:   "event-1",
		BookingID: "booking-1",
		EventType: service.EventCreated,
		Payload:   []byte(`{"status":"PENDING_PAYMENT"}`),
		CreatedAt: now,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO appointments")).
		WithArgs(
			booking.BookingID,
			booking.PatientID,
			booking.DoctorID,
			booking.SlotID,
			statusToDB(booking.Status),
			sql.NullString{},
			sql.NullString{},
			booking.Notes,
			booking.CreatedAt,
			booking.UpdatedAt,
			booking.ReservedUntil,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO appointment_events")).
		WithArgs(event.EventID, event.BookingID, eventTypeToDB(event.EventType), event.Payload, event.CreatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	got, err := repo.CreateBooking(context.Background(), booking, event)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.BookingID != booking.BookingID {
		t.Fatalf("expected booking id %q, got %q", booking.BookingID, got.BookingID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetBookingReadsAppointmentAndEvents(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := NewRepositoryFromDB(db)
	now := time.Date(2026, time.May, 3, 20, 0, 0, 0, time.UTC)

	appointmentRows := sqlmock.NewRows([]string{
		"id",
		"patient_id",
		"doctor_id",
		"slot_id",
		"status",
		"payment_id",
		"confirmation_code",
		"notes",
		"created_at",
		"updated_at",
		"reserved_until",
		"confirmed_at",
		"cancelled_at",
	}).AddRow(
		"booking-1",
		"patient-1",
		"doctor-1",
		"slot-1",
		"PENDING_PAYMENT",
		nil,
		nil,
		"Control anual",
		now,
		now,
		now.Add(15*time.Minute),
		nil,
		nil,
	)
	eventRows := sqlmock.NewRows([]string{"id", "appointment_id", "event_type", "payload", "created_at"}).
		AddRow("event-1", "booking-1", "CREATED", []byte(`{"status":"PENDING_PAYMENT"}`), now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, patient_id, doctor_id, slot_id, status, payment_id, confirmation_code, notes, created_at, updated_at, reserved_until, confirmed_at, cancelled_at FROM appointments WHERE id = $1")).
		WithArgs("booking-1").
		WillReturnRows(appointmentRows)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, appointment_id, event_type, payload, created_at FROM appointment_events WHERE appointment_id = $1 ORDER BY created_at ASC")).
		WithArgs("booking-1").
		WillReturnRows(eventRows)

	booking, err := repo.GetBooking(context.Background(), "booking-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	events, err := repo.ListBookingEvents(context.Background(), "booking-1")
	if err != nil {
		t.Fatalf("expected no error listing events, got %v", err)
	}

	if booking.Status != service.StatusPendingPayment {
		t.Fatalf("expected pending payment status, got %v", booking.Status)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if events[0].EventType != service.EventCreated {
		t.Fatalf("expected created event, got %v", events[0].EventType)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUpdateBookingStatusPersistsAppointmentStateAndEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := NewRepositoryFromDB(db)
	now := time.Date(2026, time.May, 3, 20, 30, 0, 0, time.UTC)
	cancelledAt := now
	input := service.UpdateBookingStatusInput{
		BookingID:   "booking-1",
		Status:      service.StatusCancelled,
		UpdatedAt:   now,
		CancelledAt: &cancelledAt,
		Event: service.BookingEvent{
			EventID:   "event-2",
			BookingID: "booking-1",
			EventType: service.EventCancelled,
			Payload:   []byte(`{"reason":"patient request"}`),
			CreatedAt: now,
		},
	}

	updatedRows := sqlmock.NewRows([]string{
		"id",
		"patient_id",
		"doctor_id",
		"slot_id",
		"status",
		"payment_id",
		"confirmation_code",
		"notes",
		"created_at",
		"updated_at",
		"reserved_until",
		"confirmed_at",
		"cancelled_at",
	}).AddRow(
		"booking-1",
		"patient-1",
		"doctor-1",
		"slot-1",
		"CANCELLED",
		nil,
		nil,
		"Control anual",
		now.Add(-30*time.Minute),
		now,
		now.Add(15*time.Minute),
		nil,
		cancelledAt,
	)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE appointments SET status = $2, payment_id = $3, confirmation_code = $4, updated_at = $5, confirmed_at = $6, cancelled_at = $7 WHERE id = $1 RETURNING id, patient_id, doctor_id, slot_id, status, payment_id, confirmation_code, notes, created_at, updated_at, reserved_until, confirmed_at, cancelled_at")).
		WithArgs(
			input.BookingID,
			statusToDB(input.Status),
			sql.NullString{},
			sql.NullString{},
			input.UpdatedAt,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnRows(updatedRows)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO appointment_events")).
		WithArgs(input.Event.EventID, input.Event.BookingID, eventTypeToDB(input.Event.EventType), input.Event.Payload, input.Event.CreatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	booking, err := repo.UpdateBookingStatus(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if booking.Status != service.StatusCancelled {
		t.Fatalf("expected cancelled status, got %v", booking.Status)
	}
	if booking.CancelledAt == nil || !booking.CancelledAt.Equal(cancelledAt) {
		t.Fatalf("expected cancelled_at to be persisted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

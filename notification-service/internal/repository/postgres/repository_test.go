package postgres

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/MedConnect/notification-service/internal/domain"
)

func TestSaveNotificationInsertsWhenEventIsNew(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := NewRepositoryFromDB(db)
	now := time.Date(2026, time.May, 3, 23, 0, 0, 0, time.UTC)
	notification := domain.Notification{
		EventID:     "11111111-1111-1111-1111-111111111111",
		BookingID:   "22222222-2222-2222-2222-222222222222",
		EventType:   "booking.created",
		RecipientID: "33333333-3333-3333-3333-333333333333",
		Channel:     "IN_APP",
		Message:     "Reserva creada.",
		Status:      "SENT",
		Payload:     []byte(`{"event_id":"11111111-1111-1111-1111-111111111111"}`),
		CreatedAt:   now,
	}

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO notifications")).
		WithArgs(
			notification.EventID,
			notification.BookingID,
			notification.EventType,
			notification.RecipientID,
			notification.Channel,
			notification.Message,
			notification.Status,
			notification.Payload,
			notification.CreatedAt,
		).
		WillReturnRows(sqlmock.NewRows([]string{"inserted"}).AddRow(true))

	inserted, err := repo.SaveNotification(context.Background(), notification)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !inserted {
		t.Fatalf("expected notification to be inserted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestSaveNotificationIgnoresDuplicateEventID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := NewRepositoryFromDB(db)
	notification := domain.Notification{
		EventID:     "11111111-1111-1111-1111-111111111111",
		BookingID:   "22222222-2222-2222-2222-222222222222",
		EventType:   "booking.created",
		RecipientID: "33333333-3333-3333-3333-333333333333",
		Channel:     "IN_APP",
		Message:     "Reserva creada.",
		Status:      "SENT",
		Payload:     []byte(`{}`),
		CreatedAt:   time.Date(2026, time.May, 3, 23, 0, 0, 0, time.UTC),
	}

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO notifications")).
		WithArgs(
			notification.EventID,
			notification.BookingID,
			notification.EventType,
			notification.RecipientID,
			notification.Channel,
			notification.Message,
			notification.Status,
			notification.Payload,
			notification.CreatedAt,
		).
		WillReturnError(sql.ErrNoRows)

	inserted, err := repo.SaveNotification(context.Background(), notification)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inserted {
		t.Fatalf("expected duplicate event to be ignored")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

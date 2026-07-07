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

func TestListNotificationsReturnsRecentRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := NewRepositoryFromDB(db)
	createdAt := time.Date(2026, time.May, 4, 13, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id",
		"event_id",
		"booking_id",
		"event_type",
		"recipient_id",
		"channel",
		"message",
		"status",
		"payload",
		"created_at",
		"read_at",
	}).AddRow(
		int64(7),
		"11111111-1111-1111-1111-111111111111",
		"22222222-2222-2222-2222-222222222222",
		"booking.created",
		"33333333-3333-3333-3333-333333333333",
		"IN_APP",
		"Reserva creada.",
		"SENT",
		[]byte(`{"booking_id":"22222222-2222-2222-2222-222222222222"}`),
		createdAt,
		nil,
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, event_id::text, booking_id::text, event_type, recipient_id::text, channel, message, status, payload, created_at, read_at")).
		WithArgs("33333333-3333-3333-3333-333333333333", 10).
		WillReturnRows(rows)

	notifications, err := repo.ListNotifications(context.Background(), "33333333-3333-3333-3333-333333333333", 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected one notification, got %d", len(notifications))
	}
	if notifications[0].ID != 7 || notifications[0].ReadAt != nil {
		t.Fatalf("expected unread notification with id 7, got %+v", notifications[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestCountUnreadFiltersByRecipient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := NewRepositoryFromDB(db)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM notifications WHERE recipient_id = $1 AND read_at IS NULL")).
		WithArgs("patient-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(3)))

	total, err := repo.CountUnread(context.Background(), "patient-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 3 {
		t.Fatalf("expected unread count 3, got %d", total)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMarkNotificationReadIsScopedToRecipient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := NewRepositoryFromDB(db)
	readAt := time.Date(2026, time.May, 4, 13, 5, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE notifications")).
		WithArgs(int64(7), "patient-1", readAt).
		WillReturnRows(sqlmock.NewRows([]string{"read_at"}).AddRow(readAt))

	storedReadAt, found, err := repo.MarkNotificationRead(context.Background(), 7, "patient-1", readAt)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !found {
		t.Fatalf("expected notification to be found")
	}
	if storedReadAt == nil || !storedReadAt.Equal(readAt) {
		t.Fatalf("expected stored read_at %s, got %v", readAt, storedReadAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

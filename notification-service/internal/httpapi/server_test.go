package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MedConnect/notification-service/internal/domain"
)

type fakeRepository struct {
	listRecipient  string
	listLimit      int
	countRecipient string
	readID         int64
	readRecipient  string
}

func (r *fakeRepository) ListNotifications(_ context.Context, recipientID string, limit int) ([]domain.Notification, error) {
	r.listRecipient = recipientID
	r.listLimit = limit
	return []domain.Notification{
		{
			ID:          7,
			EventID:     "11111111-1111-1111-1111-111111111111",
			BookingID:   "22222222-2222-2222-2222-222222222222",
			EventType:   "booking.created",
			RecipientID: recipientID,
			Channel:     "IN_APP",
			Message:     "Reserva creada.",
			Status:      "SENT",
			Payload:     []byte(`{"booking_id":"22222222-2222-2222-2222-222222222222"}`),
			CreatedAt:   time.Date(2026, time.May, 4, 13, 0, 0, 0, time.UTC),
		},
	}, nil
}

func (r *fakeRepository) CountUnread(_ context.Context, recipientID string) (int64, error) {
	r.countRecipient = recipientID
	return 1, nil
}

func (r *fakeRepository) CountNotifications(_ context.Context, recipientID string) (int64, error) {
	if recipientID == "" {
		return 2, nil
	}
	return 1, nil
}

func (r *fakeRepository) MarkNotificationRead(_ context.Context, notificationID int64, recipientID string, readAt time.Time) (*time.Time, bool, error) {
	r.readID = notificationID
	r.readRecipient = recipientID
	return &readAt, true, nil
}

func TestListNotificationsRequiresRecipient(t *testing.T) {
	server := NewServer(&fakeRepository{}, Config{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListNotificationsMapsRepositoryResponse(t *testing.T) {
	repo := &fakeRepository{}
	server := NewServer(repo, Config{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/notifications?recipient_id=patient-1&limit=2", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.listRecipient != "patient-1" || repo.listLimit != 2 {
		t.Fatalf("expected recipient patient-1 and limit 2, got recipient=%q limit=%d", repo.listRecipient, repo.listLimit)
	}
	if !strings.Contains(rec.Body.String(), `"message":"Reserva creada."`) {
		t.Fatalf("expected notification response, got %s", rec.Body.String())
	}
}

func TestMarkReadUsesNotificationIDAndRecipient(t *testing.T) {
	repo := &fakeRepository{}
	server := NewServer(repo, Config{}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/notifications/7/read?recipient_id=patient-1", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.readID != 7 || repo.readRecipient != "patient-1" {
		t.Fatalf("expected read id 7 for patient-1, got id=%d recipient=%q", repo.readID, repo.readRecipient)
	}
}

func TestDevStatusIncludesKafkaMetadata(t *testing.T) {
	repo := &fakeRepository{}
	server := NewServer(repo, Config{
		KafkaBrokers:  []string{"kafka:9092"},
		BookingTopic:  "medconnect.booking.events.v1",
		DLQTopic:      "medconnect.booking.events.dlq.v1",
		ConsumerGroup: "medconnect-notification-service-v1",
	}, nil)
	req := httptest.NewRequest(http.MethodGet, "/dev/status?recipient_id=patient-1&limit=4", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.listRecipient != "patient-1" || repo.listLimit != 4 {
		t.Fatalf("expected latest list for patient-1 with limit 4, got recipient=%q limit=%d", repo.listRecipient, repo.listLimit)
	}
	if !strings.Contains(rec.Body.String(), `"booking_topic":"medconnect.booking.events.v1"`) {
		t.Fatalf("expected kafka metadata response, got %s", rec.Body.String())
	}
}

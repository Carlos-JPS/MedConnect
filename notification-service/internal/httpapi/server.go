package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MedConnect/notification-service/internal/domain"
)

type Repository interface {
	ListNotifications(ctx context.Context, recipientID string, limit int) ([]domain.Notification, error)
	CountUnread(ctx context.Context, recipientID string) (int64, error)
	CountNotifications(ctx context.Context, recipientID string) (int64, error)
	MarkNotificationRead(ctx context.Context, notificationID int64, recipientID string, readAt time.Time) (*time.Time, bool, error)
}

type Config struct {
	KafkaBrokers  []string
	BookingTopic  string
	DLQTopic      string
	ConsumerGroup string
}

type Server struct {
	repo   Repository
	cfg    Config
	logger *log.Logger
}

func NewServer(repo Repository, cfg Config, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}
	return &Server{repo: repo, cfg: cfg, logger: logger}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/health":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "notification-service"})
	case r.Method == http.MethodGet && r.URL.Path == "/notifications":
		s.listNotifications(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/notifications/unread-count":
		s.unreadCount(w, r)
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/notifications/") && strings.HasSuffix(r.URL.Path, "/read"):
		s.markRead(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/dev/status":
		s.devStatus(w, r)
	default:
		writeError(w, http.StatusNotFound, "ruta no encontrada")
	}
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	recipientID := r.URL.Query().Get("recipient_id")
	if recipientID == "" {
		writeError(w, http.StatusBadRequest, "recipient_id es obligatorio")
		return
	}

	notifications, err := s.repo.ListNotifications(r.Context(), recipientID, limitFromQuery(r, 10))
	if err != nil {
		s.logger.Printf("error listando notificaciones: %v", err)
		writeError(w, http.StatusInternalServerError, "no fue posible listar notificaciones")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": mapNotifications(notifications),
	})
}

func (s *Server) unreadCount(w http.ResponseWriter, r *http.Request) {
	recipientID := r.URL.Query().Get("recipient_id")
	if recipientID == "" {
		writeError(w, http.StatusBadRequest, "recipient_id es obligatorio")
		return
	}

	total, err := s.repo.CountUnread(r.Context(), recipientID)
	if err != nil {
		s.logger.Printf("error contando notificaciones no leidas: %v", err)
		writeError(w, http.StatusInternalServerError, "no fue posible contar notificaciones")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"unread_count": total})
}

func (s *Server) markRead(w http.ResponseWriter, r *http.Request) {
	recipientID := r.URL.Query().Get("recipient_id")
	if recipientID == "" {
		writeError(w, http.StatusBadRequest, "recipient_id es obligatorio")
		return
	}

	notificationID, ok := notificationIDFromPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "notificacion no encontrada")
		return
	}

	readAt, found, err := s.repo.MarkNotificationRead(r.Context(), notificationID, recipientID, time.Now().UTC())
	if err != nil {
		s.logger.Printf("error marcando notificacion como leida: %v", err)
		writeError(w, http.StatusInternalServerError, "no fue posible marcar la notificacion")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "notificacion no encontrada")
		return
	}

	readAtValue := ""
	if readAt != nil {
		readAtValue = readAt.Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      notificationID,
		"read_at": readAtValue,
	})
}

func (s *Server) devStatus(w http.ResponseWriter, r *http.Request) {
	recipientID := r.URL.Query().Get("recipient_id")
	limit := limitFromQuery(r, 5)

	total, err := s.repo.CountNotifications(r.Context(), "")
	if err != nil {
		s.logger.Printf("error contando total de notificaciones: %v", err)
		writeError(w, http.StatusInternalServerError, "no fue posible consultar el estado dev")
		return
	}

	recipientTotal := int64(0)
	unread := int64(0)
	if recipientID != "" {
		recipientTotal, err = s.repo.CountNotifications(r.Context(), recipientID)
		if err != nil {
			s.logger.Printf("error contando notificaciones por receptor: %v", err)
			writeError(w, http.StatusInternalServerError, "no fue posible consultar el estado dev")
			return
		}
		unread, err = s.repo.CountUnread(r.Context(), recipientID)
		if err != nil {
			s.logger.Printf("error contando no leidas por receptor: %v", err)
			writeError(w, http.StatusInternalServerError, "no fue posible consultar el estado dev")
			return
		}
	}

	notifications, err := s.repo.ListNotifications(r.Context(), recipientID, limit)
	if err != nil {
		s.logger.Printf("error listando ultimas notificaciones para dev: %v", err)
		writeError(w, http.StatusInternalServerError, "no fue posible consultar el estado dev")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"service":              "notification-service",
		"store":                "postgres",
		"kafka_brokers":        s.cfg.KafkaBrokers,
		"booking_topic":        s.cfg.BookingTopic,
		"dlq_topic":            s.cfg.DLQTopic,
		"consumer_group":       s.cfg.ConsumerGroup,
		"total_notifications":  total,
		"recipient_id":         recipientID,
		"recipient_total":      recipientTotal,
		"recipient_unread":     unread,
		"latest_notifications": mapNotifications(notifications),
	})
}

type notificationResponse struct {
	ID          int64           `json:"id"`
	EventID     string          `json:"event_id"`
	BookingID   string          `json:"booking_id"`
	EventType   string          `json:"event_type"`
	RecipientID string          `json:"recipient_id"`
	Channel     string          `json:"channel"`
	Message     string          `json:"message"`
	Status      string          `json:"status"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   string          `json:"created_at"`
	ReadAt      string          `json:"read_at,omitempty"`
}

func mapNotifications(notifications []domain.Notification) []notificationResponse {
	result := make([]notificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		payload := json.RawMessage(notification.Payload)
		if !json.Valid(payload) {
			payload = json.RawMessage(`{}`)
		}

		readAt := ""
		if notification.ReadAt != nil {
			readAt = notification.ReadAt.Format(time.RFC3339)
		}

		result = append(result, notificationResponse{
			ID:          notification.ID,
			EventID:     notification.EventID,
			BookingID:   notification.BookingID,
			EventType:   notification.EventType,
			RecipientID: notification.RecipientID,
			Channel:     notification.Channel,
			Message:     notification.Message,
			Status:      notification.Status,
			Payload:     payload,
			CreatedAt:   notification.CreatedAt.Format(time.RFC3339),
			ReadAt:      readAt,
		})
	}
	return result
}

func limitFromQuery(r *http.Request, fallback int) int {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > 50 {
		return 50
	}
	return parsed
}

func notificationIDFromPath(path string) (int64, bool) {
	trimmed := strings.TrimPrefix(path, "/notifications/")
	if trimmed == path || trimmed == "" {
		return 0, false
	}
	trimmed = strings.TrimSuffix(trimmed, "/read")
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return 0, false
	}

	id, err := strconv.ParseInt(trimmed, 10, 64)
	return id, err == nil && id > 0
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

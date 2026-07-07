package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MedConnect/notification-service/internal/domain"
	_ "github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func Open(dsn string) (*Repository, error) {
	if dsn == "" {
		return nil, errors.New("NOTIFICATION_DB_DSN no esta configurado")
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

func (r *Repository) SaveNotification(ctx context.Context, notification domain.Notification) (bool, error) {
	var inserted bool
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO notifications (event_id, booking_id, event_type, recipient_id, channel, message, status, payload, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (event_id) DO NOTHING
RETURNING TRUE`,
		notification.EventID,
		notification.BookingID,
		notification.EventType,
		notification.RecipientID,
		notification.Channel,
		notification.Message,
		notification.Status,
		notification.Payload,
		notification.CreatedAt,
	).Scan(&inserted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("guardar notificacion: %w", err)
	}
	return inserted, nil
}

func (r *Repository) ListNotifications(ctx context.Context, recipientID string, limit int) ([]domain.Notification, error) {
	if limit <= 0 {
		limit = 10
	}

	var rows *sql.Rows
	var err error
	if recipientID == "" {
		rows, err = r.db.QueryContext(
			ctx,
			`SELECT id, event_id::text, booking_id::text, event_type, recipient_id::text, channel, message, status, payload, created_at, read_at
FROM notifications
ORDER BY created_at DESC
LIMIT $1`,
			limit,
		)
	} else {
		rows, err = r.db.QueryContext(
			ctx,
			`SELECT id, event_id::text, booking_id::text, event_type, recipient_id::text, channel, message, status, payload, created_at, read_at
FROM notifications
WHERE recipient_id = $1
ORDER BY created_at DESC
LIMIT $2`,
			recipientID,
			limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("listar notificaciones: %w", err)
	}
	defer rows.Close()

	notifications := make([]domain.Notification, 0)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer notificaciones: %w", err)
	}
	return notifications, nil
}

func (r *Repository) CountUnread(ctx context.Context, recipientID string) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM notifications WHERE recipient_id = $1 AND read_at IS NULL`,
		recipientID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("contar notificaciones no leidas: %w", err)
	}
	return total, nil
}

func (r *Repository) CountNotifications(ctx context.Context, recipientID string) (int64, error) {
	var total int64
	var err error
	if recipientID == "" {
		err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&total)
	} else {
		err = r.db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM notifications WHERE recipient_id = $1`,
			recipientID,
		).Scan(&total)
	}
	if err != nil {
		return 0, fmt.Errorf("contar notificaciones: %w", err)
	}
	return total, nil
}

func (r *Repository) MarkNotificationRead(ctx context.Context, notificationID int64, recipientID string, readAt time.Time) (*time.Time, bool, error) {
	var storedReadAt sql.NullTime
	err := r.db.QueryRowContext(
		ctx,
		`UPDATE notifications
SET read_at = COALESCE(read_at, $3)
WHERE id = $1 AND recipient_id = $2
RETURNING read_at`,
		notificationID,
		recipientID,
		readAt,
	).Scan(&storedReadAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("marcar notificacion como leida: %w", err)
	}
	if !storedReadAt.Valid {
		return nil, true, nil
	}
	return &storedReadAt.Time, true, nil
}

type notificationScanner interface {
	Scan(dest ...any) error
}

func scanNotification(scanner notificationScanner) (domain.Notification, error) {
	var notification domain.Notification
	var readAt sql.NullTime
	if err := scanner.Scan(
		&notification.ID,
		&notification.EventID,
		&notification.BookingID,
		&notification.EventType,
		&notification.RecipientID,
		&notification.Channel,
		&notification.Message,
		&notification.Status,
		&notification.Payload,
		&notification.CreatedAt,
		&readAt,
	); err != nil {
		return domain.Notification{}, fmt.Errorf("mapear notificacion: %w", err)
	}
	if readAt.Valid {
		notification.ReadAt = &readAt.Time
	}
	return notification, nil
}

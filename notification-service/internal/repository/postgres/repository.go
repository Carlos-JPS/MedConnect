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

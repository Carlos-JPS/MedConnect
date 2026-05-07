package repository

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

type postgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(dsn string) (PaymentRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error abriendo conexión a postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error verificando conexión a postgres: %w", err)
	}
	return &postgresRepository{db: db}, nil
}

func (r *postgresRepository) CreatePayment(p *Payment) (*Payment, error) {
	query := `
		INSERT INTO payments (booking_id, user_id, amount, currency, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING payment_id, booking_id, user_id, amount, currency, status, created_at, updated_at`

	row := r.db.QueryRow(query, p.BookingID, p.UserID, p.Amount, p.Currency, p.Status)

	var created Payment
	err := row.Scan(
		&created.PaymentID,
		&created.BookingID,
		&created.UserID,
		&created.Amount,
		&created.Currency,
		&created.Status,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creando pago: %w", err)
	}
	return &created, nil
}

func (r *postgresRepository) GetPaymentByID(paymentID string) (*Payment, error) {
	query := `
		SELECT payment_id, booking_id, user_id, amount, currency, status, created_at, updated_at
		FROM payments
		WHERE payment_id = $1`

	var p Payment
	err := r.db.QueryRow(query, paymentID).Scan(
		&p.PaymentID, &p.BookingID, &p.UserID,
		&p.Amount, &p.Currency, &p.Status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("pago no encontrado: %s", paymentID)
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo pago: %w", err)
	}
	return &p, nil
}

func (r *postgresRepository) GetPaymentsByUserID(userID string) ([]*Payment, error) {
	query := `
		SELECT payment_id, booking_id, user_id, amount, currency, status, created_at, updated_at
		FROM payments
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error listando pagos del usuario: %w", err)
	}
	defer rows.Close()

	var payments []*Payment
	for rows.Next() {
		var p Payment
		if err := rows.Scan(
			&p.PaymentID, &p.BookingID, &p.UserID,
			&p.Amount, &p.Currency, &p.Status,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error leyendo fila de pago: %w", err)
		}
		payments = append(payments, &p)
	}
	return payments, rows.Err()
}

func (r *postgresRepository) GetPaymentByBookingID(bookingID string) (*Payment, error) {
	query := `
		SELECT payment_id, booking_id, user_id, amount, currency, status, created_at, updated_at
		FROM payments
		WHERE booking_id = $1`

	var p Payment
	err := r.db.QueryRow(query, bookingID).Scan(
		&p.PaymentID, &p.BookingID, &p.UserID,
		&p.Amount, &p.Currency, &p.Status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("pago no encontrado para la reserva: %s", bookingID)
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo pago por reserva: %w", err)
	}
	return &p, nil
}

func (r *postgresRepository) UpdatePaymentStatus(paymentID, status string) error {
	query := `
		UPDATE payments
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE payment_id = $2`

	result, err := r.db.Exec(query, status, paymentID)
	if err != nil {
		return fmt.Errorf("error actualizando estado del pago: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("pago no encontrado al actualizar estado: %s", paymentID)
	}
	return nil
}

func (r *postgresRepository) CreateTransactionLog(log *TransactionLog) error {
	query := `
		INSERT INTO transaction_logs (payment_id, external_reference, status, message)
		VALUES ($1, $2, $3, $4)
		RETURNING transaction_id, created_at`

	return r.db.QueryRow(query,
		log.PaymentID,
		log.ExternalReference,
		log.Status,
		log.Message,
	).Scan(&log.TransactionID, &log.CreatedAt)
}

func (r *postgresRepository) CreateRefund(ref *Refund) (*Refund, error) {
	query := `
		INSERT INTO refunds (payment_id, amount, reason, status)
		VALUES ($1, $2, $3, $4)
		RETURNING refund_id, payment_id, amount, reason, status, created_at`

	var created Refund
	err := r.db.QueryRow(query,
		ref.PaymentID, ref.Amount, ref.Reason, ref.Status,
	).Scan(
		&created.RefundID,
		&created.PaymentID,
		&created.Amount,
		&created.Reason,
		&created.Status,
		&created.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creando reembolso: %w", err)
	}
	return &created, nil
}

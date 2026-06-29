package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(dsn string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) GetAvailableSlots(ctx context.Context, specialty string, startDate, endDate time.Time) ([]*Slot, error) {
	query := `
		SELECT s.id, s.calendar_id, c.doctor_id, s.start_time, s.end_time, s.status, s.created_at, s.updated_at
		FROM availability_slots s
		JOIN doctor_calendars c ON s.calendar_id = c.id
		WHERE c.specialty = $1 
		  AND s.status = 'available'
		  AND s.start_time >= $2 
		  AND s.end_time <= $3
		ORDER BY s.start_time ASC
	`
	rows, err := r.db.QueryContext(ctx, query, specialty, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("error querying available slots: %w", err)
	}
	defer rows.Close()

	var slots []*Slot
	for rows.Next() {
		var s Slot
		var statusStr string
		if err := rows.Scan(&s.ID, &s.CalendarID, &s.DoctorID, &s.StartTime, &s.EndTime, &statusStr, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning slot: %w", err)
		}
		s.Status = SlotStatus(statusStr)
		slots = append(slots, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating slots: %w", err)
	}
	return slots, nil
}

func (r *PostgresRepository) updateSlotStatus(ctx context.Context, slotID string, bookingID string, expectedStatus SlotStatus, newStatus SlotStatus) (*Slot, error) {
	query := `
		UPDATE availability_slots
		SET status = $1, booking_id = COALESCE(booking_id, $4), updated_at = NOW()
		WHERE id = $2 AND status = $3 AND (booking_id = $4 OR booking_id IS NULL)
		RETURNING id, calendar_id, start_time, end_time, status, created_at, updated_at
	`
	var s Slot
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, newStatus, slotID, expectedStatus, bookingID).Scan(
		&s.ID, &s.CalendarID, &s.StartTime, &s.EndTime, &statusStr, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: expected status '%s'", ErrSlotNotAvailable, expectedStatus)
	}
	if err != nil {
		return nil, fmt.Errorf("error updating slot: %w", err)
	}

	s.Status = SlotStatus(statusStr)

	// Fetch doctor_id to complete the model
	err = r.db.QueryRowContext(ctx, "SELECT doctor_id FROM doctor_calendars WHERE id = $1", s.CalendarID).Scan(&s.DoctorID)
	if err != nil {
		return nil, fmt.Errorf("error fetching doctor id: %w", err)
	}

	return &s, nil
}

func (r *PostgresRepository) HoldSlot(ctx context.Context, slotID string, bookingID string, heldUntil time.Time) (*Slot, error) {
	query := `
		UPDATE availability_slots
		SET status = $1, booking_id = $2, held_until = $3, updated_at = NOW()
		WHERE id = $4 AND status = $5
		RETURNING id, calendar_id, start_time, end_time, status, created_at, updated_at
	`
	var s Slot
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, StatusHeld, bookingID, heldUntil, slotID, StatusAvailable).Scan(
		&s.ID, &s.CalendarID, &s.StartTime, &s.EndTime, &statusStr, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: expected status '%s'", ErrSlotNotAvailable, StatusAvailable)
	}
	if err != nil {
		return nil, fmt.Errorf("error updating slot: %w", err)
	}
	s.Status = SlotStatus(statusStr)
	if err := r.db.QueryRowContext(ctx, "SELECT doctor_id FROM doctor_calendars WHERE id = $1", s.CalendarID).Scan(&s.DoctorID); err != nil {
		return nil, fmt.Errorf("error fetching doctor id: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) ConfirmSlotBooking(ctx context.Context, slotID string, bookingID string) (*Slot, error) {
	return r.updateSlotStatus(ctx, slotID, bookingID, StatusHeld, StatusBooked)
}

func (r *PostgresRepository) ReleaseHeldSlot(ctx context.Context, slotID string, bookingID string) (*Slot, error) {
	query := `
		UPDATE availability_slots
		SET status = $1, held_until = NULL, booking_id = NULL, updated_at = NOW()
		WHERE id = $2 AND status IN ($3, $4) AND (booking_id = $5 OR booking_id IS NULL)
		RETURNING id, calendar_id, start_time, end_time, status, created_at, updated_at
	`
	var s Slot
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, StatusAvailable, slotID, StatusHeld, StatusBooked, bookingID).Scan(
		&s.ID, &s.CalendarID, &s.StartTime, &s.EndTime, &statusStr, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: expected status '%s' or '%s'", ErrSlotNotAvailable, StatusHeld, StatusBooked)
	}
	if err != nil {
		return nil, fmt.Errorf("error updating slot: %w", err)
	}

	s.Status = SlotStatus(statusStr)

	err = r.db.QueryRowContext(ctx, "SELECT doctor_id FROM doctor_calendars WHERE id = $1", s.CalendarID).Scan(&s.DoctorID)
	if err != nil {
		return nil, fmt.Errorf("error fetching doctor id: %w", err)
	}

	return &s, nil
}

func (r *PostgresRepository) GetDoctorAgenda(ctx context.Context, doctorID string, startDate, endDate time.Time) ([]*Slot, error) {
	query := `
		SELECT s.id, s.calendar_id, c.doctor_id, s.start_time, s.end_time, s.status, s.created_at, s.updated_at
		FROM availability_slots s
		JOIN doctor_calendars c ON s.calendar_id = c.id
		WHERE c.doctor_id = $1 
		  AND s.start_time >= $2 
		  AND s.end_time <= $3
		ORDER BY s.start_time ASC
	`
	rows, err := r.db.QueryContext(ctx, query, doctorID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("error querying doctor agenda: %w", err)
	}
	defer rows.Close()

	var slots []*Slot
	for rows.Next() {
		var s Slot
		var statusStr string
		if err := rows.Scan(&s.ID, &s.CalendarID, &s.DoctorID, &s.StartTime, &s.EndTime, &statusStr, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning slot: %w", err)
		}
		s.Status = SlotStatus(statusStr)
		slots = append(slots, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating doctor agenda: %w", err)
	}
	return slots, nil
}

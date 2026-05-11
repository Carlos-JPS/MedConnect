package repository

import (
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

func (r *PostgresRepository) GetAvailableSlots(specialty string, startDate, endDate time.Time) ([]*Slot, error) {
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
	rows, err := r.db.Query(query, specialty, startDate, endDate)
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
	return slots, nil
}

func (r *PostgresRepository) updateSlotStatus(slotID string, expectedStatus SlotStatus, newStatus SlotStatus) (*Slot, error) {
	query := `
		UPDATE availability_slots
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3
		RETURNING id, calendar_id, start_time, end_time, status, created_at, updated_at
	`
	var s Slot
	var statusStr string
	err := r.db.QueryRow(query, newStatus, slotID, expectedStatus).Scan(
		&s.ID, &s.CalendarID, &s.StartTime, &s.EndTime, &statusStr, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("slot not found or not in expected status '%s'", expectedStatus)
	}
	if err != nil {
		return nil, fmt.Errorf("error updating slot: %w", err)
	}

	s.Status = SlotStatus(statusStr)

	// Fetch doctor_id to complete the model
	err = r.db.QueryRow("SELECT doctor_id FROM doctor_calendars WHERE id = $1", s.CalendarID).Scan(&s.DoctorID)
	if err != nil {
		return nil, fmt.Errorf("error fetching doctor id: %w", err)
	}

	return &s, nil
}

func (r *PostgresRepository) HoldSlot(slotID string) (*Slot, error) {
	return r.updateSlotStatus(slotID, StatusAvailable, StatusHeld)
}

func (r *PostgresRepository) ConfirmSlotBooking(slotID string) (*Slot, error) {
	return r.updateSlotStatus(slotID, StatusHeld, StatusBooked)
}

func (r *PostgresRepository) ReleaseHeldSlot(slotID string) (*Slot, error) {
	query := `
		UPDATE availability_slots
		SET status = $1, held_until = NULL, booking_id = NULL, updated_at = NOW()
		WHERE id = $2 AND status IN ($3, $4)
		RETURNING id, calendar_id, start_time, end_time, status, created_at, updated_at
	`
	var s Slot
	var statusStr string
	err := r.db.QueryRow(query, StatusAvailable, slotID, StatusHeld, StatusBooked).Scan(
		&s.ID, &s.CalendarID, &s.StartTime, &s.EndTime, &statusStr, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("slot not found or not in expected status '%s' or '%s'", StatusHeld, StatusBooked)
	}
	if err != nil {
		return nil, fmt.Errorf("error updating slot: %w", err)
	}

	s.Status = SlotStatus(statusStr)

	err = r.db.QueryRow("SELECT doctor_id FROM doctor_calendars WHERE id = $1", s.CalendarID).Scan(&s.DoctorID)
	if err != nil {
		return nil, fmt.Errorf("error fetching doctor id: %w", err)
	}

	return &s, nil
}

func (r *PostgresRepository) GetDoctorAgenda(doctorID string, startDate, endDate time.Time) ([]*Slot, error) {
	query := `
		SELECT s.id, s.calendar_id, c.doctor_id, s.start_time, s.end_time, s.status, s.created_at, s.updated_at
		FROM availability_slots s
		JOIN doctor_calendars c ON s.calendar_id = c.id
		WHERE c.doctor_id = $1 
		  AND s.start_time >= $2 
		  AND s.end_time <= $3
		ORDER BY s.start_time ASC
	`
	rows, err := r.db.Query(query, doctorID, startDate, endDate)
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
	return slots, nil
}

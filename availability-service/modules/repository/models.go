package repository

import (
	"errors"
	"time"
)

var ErrSlotNotAvailable = errors.New("slot not found or not in expected status")

// SlotStatus represents the current state of a slot
type SlotStatus string

const (
	StatusAvailable SlotStatus = "available"
	StatusHeld      SlotStatus = "held"
	StatusBooked    SlotStatus = "booked"
)

// Slot represents an availability slot in the database
type Slot struct {
	ID         string     `json:"id"`
	CalendarID string     `json:"calendar_id"`
	DoctorID   string     `json:"doctor_id"` // Denormalized or fetched via calendar
	StartTime  time.Time  `json:"start_time"`
	EndTime    time.Time  `json:"end_time"`
	Status     SlotStatus `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// DoctorCalendar represents a doctor's agenda
type DoctorCalendar struct {
	ID        string    `json:"id"`
	DoctorID  string    `json:"doctor_id"`
	Specialty string    `json:"specialty"`
	CreatedAt time.Time `json:"created_at"`
}

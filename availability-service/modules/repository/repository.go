package repository

import (
	"context"
	"time"
)

type AvailabilityRepository interface {
	// GetAvailableSlots returns slots for a specific specialty and date range that are 'available'
	GetAvailableSlots(ctx context.Context, specialty string, startDate, endDate time.Time) ([]*Slot, error)

	// HoldSlot temporarily reserves a slot if it is 'available'
	HoldSlot(ctx context.Context, slotID string, bookingID string, heldUntil time.Time) (*Slot, error)

	// ConfirmSlotBooking confirms a 'held' slot, making it 'booked'
	ConfirmSlotBooking(ctx context.Context, slotID string, bookingID string) (*Slot, error)

	// ReleaseHeldSlot returns a 'held' slot back to 'available'
	ReleaseHeldSlot(ctx context.Context, slotID string, bookingID string) (*Slot, error)

	// GetDoctorAgenda returns all slots (regardless of status) for a specific doctor and date range
	GetDoctorAgenda(ctx context.Context, doctorID string, startDate, endDate time.Time) ([]*Slot, error)
}

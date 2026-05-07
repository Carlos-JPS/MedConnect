package repository

import "time"

type AvailabilityRepository interface {
	// GetAvailableSlots returns slots for a specific specialty and date range that are 'available'
	GetAvailableSlots(specialty string, startDate, endDate time.Time) ([]*Slot, error)

	// HoldSlot temporarily reserves a slot if it is 'available'
	HoldSlot(slotID string) (*Slot, error)

	// ConfirmSlotBooking confirms a 'held' slot, making it 'booked'
	ConfirmSlotBooking(slotID string) (*Slot, error)

	// ReleaseHeldSlot returns a 'held' slot back to 'available'
	ReleaseHeldSlot(slotID string) (*Slot, error)

	// GetDoctorAgenda returns all slots (regardless of status) for a specific doctor and date range
	GetDoctorAgenda(doctorID string, startDate, endDate time.Time) ([]*Slot, error)
}

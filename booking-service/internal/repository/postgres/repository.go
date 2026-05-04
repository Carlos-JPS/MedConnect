package postgres

import (
	"context"
	"time"

	"github.com/MedConnect/booking-service/internal/service"
)

type Repository struct {
	dsn string
}

func NewRepository(dsn string) *Repository {
	return &Repository{dsn: dsn}
}

func (r *Repository) GetBooking(_ context.Context, bookingID string) (service.Booking, error) {
	createdAt := time.Date(2026, time.April, 30, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	reservedUntil := createdAt.Add(1 * time.Hour)
	confirmedAt := updatedAt

	return service.Booking{
		BookingID:        bookingID,
		PatientID:        "patient-123",
		DoctorID:         "doctor-456",
		SlotID:           "slot-789",
		Status:           service.StatusConfirmed,
		PaymentID:        "payment-001",
		ConfirmationCode: "MED-001",
		Notes:            "Control general",
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		ReservedUntil:    reservedUntil,
		ConfirmedAt:      &confirmedAt,
	}, nil
}

package service

import (
	"context"
	"time"
)

type Status int

const (
	StatusUnspecified Status = iota
	StatusPendingPayment
	StatusConfirmed
	StatusCancelled
	StatusExpired
)

type Booking struct {
	BookingID        string
	PatientID        string
	DoctorID         string
	SlotID           string
	Status           Status
	PaymentID        string
	ConfirmationCode string
	Notes            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ReservedUntil    time.Time
	ConfirmedAt      *time.Time
	CancelledAt      *time.Time
}

type Repository interface {
	GetBooking(ctx context.Context, bookingID string) (Booking, error)
}

type BookingService interface {
	GetBooking(ctx context.Context, bookingID string) (Booking, error)
}

type bookingService struct {
	repo Repository
}

func NewBookingService(repo Repository) BookingService {
	return &bookingService{repo: repo}
}

func (s *bookingService) GetBooking(ctx context.Context, bookingID string) (Booking, error) {
	return s.repo.GetBooking(ctx, bookingID)
}

package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const defaultReservationTTL = 15 * time.Minute

type Status int

const (
	StatusUnspecified Status = iota
	StatusPendingPayment
	StatusConfirmed
	StatusCancelled
	StatusExpired
)

type EventType int

const (
	EventUnspecified EventType = iota
	EventCreated
	EventPaymentApproved
	EventConfirmed
	EventCancelled
	EventExpired
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

type BookingEvent struct {
	EventID   string
	BookingID string
	EventType EventType
	Payload   []byte
	CreatedAt time.Time
}

type BookingDetails struct {
	Booking Booking
	Events  []BookingEvent
}

type CreateBookingInput struct {
	PatientID string
	DoctorID  string
	SlotID    string
	Notes     string
}

type UpdateBookingStatusInput struct {
	BookingID        string
	Status           Status
	PaymentID        string
	ConfirmationCode string
	UpdatedAt        time.Time
	ConfirmedAt      *time.Time
	CancelledAt      *time.Time
	Event            BookingEvent
}

type Repository interface {
	CreateBooking(ctx context.Context, booking Booking, event BookingEvent) (Booking, error)
	GetBooking(ctx context.Context, bookingID string) (Booking, error)
	ListBookingEvents(ctx context.Context, bookingID string) ([]BookingEvent, error)
	ListBookingsByPatient(ctx context.Context, patientID string, status Status) ([]Booking, error)
	UpdateBookingStatus(ctx context.Context, input UpdateBookingStatusInput) (Booking, error)
}

type BookingService interface {
	CreateBooking(ctx context.Context, input CreateBookingInput) (Booking, error)
	GetBooking(ctx context.Context, bookingID string) (BookingDetails, error)
	ListBookingsByPatient(ctx context.Context, patientID string, status Status) ([]Booking, error)
	UpdateBookingStatus(ctx context.Context, input UpdateBookingStatusInput) (Booking, error)
}

type bookingService struct {
	repo  Repository
	clock func() time.Time
}

type Option func(*bookingService)

func WithClock(clock func() time.Time) Option {
	return func(s *bookingService) {
		s.clock = clock
	}
}

func NewBookingService(repo Repository, opts ...Option) BookingService {
	service := &bookingService{
		repo:  repo,
		clock: time.Now,
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *bookingService) CreateBooking(ctx context.Context, input CreateBookingInput) (Booking, error) {
	now := s.clock().UTC()
	bookingID := uuid.NewString()
	booking := Booking{
		BookingID:     bookingID,
		PatientID:     input.PatientID,
		DoctorID:      input.DoctorID,
		SlotID:        input.SlotID,
		Status:        StatusPendingPayment,
		Notes:         input.Notes,
		CreatedAt:     now,
		UpdatedAt:     now,
		ReservedUntil: now.Add(defaultReservationTTL),
	}
	payload, err := json.Marshal(map[string]string{
		"status":  "PENDING_PAYMENT",
		"slot_id": input.SlotID,
	})
	if err != nil {
		return Booking{}, err
	}
	event := BookingEvent{
		EventID:   uuid.NewString(),
		BookingID: bookingID,
		EventType: EventCreated,
		Payload:   payload,
		CreatedAt: now,
	}

	return s.repo.CreateBooking(ctx, booking, event)
}

func (s *bookingService) GetBooking(ctx context.Context, bookingID string) (BookingDetails, error) {
	booking, err := s.repo.GetBooking(ctx, bookingID)
	if err != nil {
		return BookingDetails{}, err
	}
	events, err := s.repo.ListBookingEvents(ctx, bookingID)
	if err != nil {
		return BookingDetails{}, err
	}

	return BookingDetails{Booking: booking, Events: events}, nil
}

func (s *bookingService) ListBookingsByPatient(ctx context.Context, patientID string, status Status) ([]Booking, error) {
	return s.repo.ListBookingsByPatient(ctx, patientID, status)
}

func (s *bookingService) UpdateBookingStatus(ctx context.Context, input UpdateBookingStatusInput) (Booking, error) {
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = s.clock().UTC()
	}
	return s.repo.UpdateBookingStatus(ctx, input)
}

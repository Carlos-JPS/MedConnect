package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const defaultReservationTTL = 15 * time.Minute
const defaultExternalCallTimeout = 3 * time.Second

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

type CancelBookingInput struct {
	BookingID string
	Reason    string
}

type ConfirmBookingInput struct {
	BookingID string
	PaymentID string
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

type HoldSlotInput struct {
	SlotID    string
	BookingID string
	HeldUntil time.Time
}

type ReleaseHeldSlotInput struct {
	SlotID    string
	BookingID string
}

type ConfirmSlotBookingInput struct {
	SlotID    string
	BookingID string
}

type PaymentStatus string

const (
	PaymentStatusUnspecified PaymentStatus = ""
	PaymentStatusPending     PaymentStatus = "PENDING"
	PaymentStatusApproved    PaymentStatus = "APPROVED"
	PaymentStatusCompleted   PaymentStatus = "COMPLETED"
	PaymentStatusRejected    PaymentStatus = "REJECTED"
	PaymentStatusRefunded    PaymentStatus = "REFUNDED"
)

type PaymentDetails struct {
	PaymentID string
	BookingID string
	Status    PaymentStatus
}

var (
	ErrExternalDependency  = errors.New("fallo en dependencia externa")
	ErrInvalidBookingState = errors.New("estado de reserva invalido")
)

type AvailabilityClient interface {
	HoldSlot(ctx context.Context, input HoldSlotInput) error
	ReleaseHeldSlot(ctx context.Context, input ReleaseHeldSlotInput) error
	ConfirmSlotBooking(ctx context.Context, input ConfirmSlotBookingInput) error
}

type PaymentClient interface {
	GetPayment(ctx context.Context, paymentID string) (PaymentDetails, error)
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
	CancelBooking(ctx context.Context, input CancelBookingInput) (Booking, error)
	ConfirmBooking(ctx context.Context, input ConfirmBookingInput) (Booking, error)
	GetBooking(ctx context.Context, bookingID string) (BookingDetails, error)
	ListBookingsByPatient(ctx context.Context, patientID string, status Status) ([]Booking, error)
	UpdateBookingStatus(ctx context.Context, input UpdateBookingStatusInput) (Booking, error)
}

type bookingService struct {
	repo                Repository
	clock               func() time.Time
	availability        AvailabilityClient
	payment             PaymentClient
	externalCallTimeout time.Duration
}

type Option func(*bookingService)

func WithClock(clock func() time.Time) Option {
	return func(s *bookingService) {
		s.clock = clock
	}
}

func WithAvailabilityClient(client AvailabilityClient) Option {
	return func(s *bookingService) {
		s.availability = client
	}
}

func WithPaymentClient(client PaymentClient) Option {
	return func(s *bookingService) {
		s.payment = client
	}
}

func WithExternalCallTimeout(timeout time.Duration) Option {
	return func(s *bookingService) {
		s.externalCallTimeout = timeout
	}
}

func NewBookingService(repo Repository, opts ...Option) BookingService {
	service := &bookingService{
		repo:                repo,
		clock:               time.Now,
		availability:        noopAvailabilityClient{},
		payment:             noopPaymentClient{},
		externalCallTimeout: defaultExternalCallTimeout,
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

	externalCtx, cancel := s.externalContext(ctx)
	defer cancel()
	if err := s.availability.HoldSlot(externalCtx, HoldSlotInput{
		SlotID:    input.SlotID,
		BookingID: bookingID,
		HeldUntil: booking.ReservedUntil,
	}); err != nil {
		return Booking{}, fmt.Errorf("%w: hold slot: %v", ErrExternalDependency, err)
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

	createdBooking, err := s.repo.CreateBooking(ctx, booking, event)
	if err != nil {
		releaseCtx, releaseCancel := s.externalContext(ctx)
		defer releaseCancel()
		_ = s.availability.ReleaseHeldSlot(releaseCtx, ReleaseHeldSlotInput{
			SlotID:    input.SlotID,
			BookingID: bookingID,
		})
		return Booking{}, err
	}

	return createdBooking, nil
}

func (s *bookingService) CancelBooking(ctx context.Context, input CancelBookingInput) (Booking, error) {
	booking, err := s.repo.GetBooking(ctx, input.BookingID)
	if err != nil {
		return Booking{}, err
	}
	if booking.Status == StatusCancelled {
		return booking, nil
	}
	if booking.Status != StatusPendingPayment {
		return Booking{}, fmt.Errorf("%w: no se puede cancelar una reserva %s", ErrInvalidBookingState, booking.Status)
	}

	externalCtx, cancel := s.externalContext(ctx)
	defer cancel()
	if err := s.availability.ReleaseHeldSlot(externalCtx, ReleaseHeldSlotInput{
		SlotID:    booking.SlotID,
		BookingID: booking.BookingID,
	}); err != nil {
		return Booking{}, fmt.Errorf("%w: release held slot: %v", ErrExternalDependency, err)
	}

	now := s.clock().UTC()
	payload, err := json.Marshal(map[string]string{
		"reason": input.Reason,
		"status": "CANCELLED",
	})
	if err != nil {
		return Booking{}, err
	}

	return s.repo.UpdateBookingStatus(ctx, UpdateBookingStatusInput{
		BookingID:   booking.BookingID,
		Status:      StatusCancelled,
		UpdatedAt:   now,
		CancelledAt: &now,
		Event: BookingEvent{
			EventID:   uuid.NewString(),
			BookingID: booking.BookingID,
			EventType: EventCancelled,
			Payload:   payload,
			CreatedAt: now,
		},
	})
}

func (s *bookingService) ConfirmBooking(ctx context.Context, input ConfirmBookingInput) (Booking, error) {
	booking, err := s.repo.GetBooking(ctx, input.BookingID)
	if err != nil {
		return Booking{}, err
	}
	switch booking.Status {
	case StatusConfirmed:
		return booking, nil
	case StatusCancelled, StatusExpired:
		return Booking{}, fmt.Errorf("%w: no se puede confirmar una reserva %s", ErrInvalidBookingState, booking.Status)
	}

	externalCtx, cancel := s.externalContext(ctx)
	defer cancel()
	payment, err := s.payment.GetPayment(externalCtx, input.PaymentID)
	if err != nil {
		return Booking{}, fmt.Errorf("%w: validar pago: %v", ErrExternalDependency, err)
	}
	if payment.BookingID == "" {
		return Booking{}, fmt.Errorf("%w: pago %s no informa reserva asociada", ErrInvalidBookingState, input.PaymentID)
	}
	if payment.BookingID != booking.BookingID {
		return Booking{}, fmt.Errorf("%w: pago %s pertenece a la reserva %s", ErrInvalidBookingState, input.PaymentID, payment.BookingID)
	}
	if !payment.Status.IsApproved() {
		return Booking{}, fmt.Errorf("pago %s no aprobado: %s", input.PaymentID, payment.Status)
	}

	externalCtx, cancel = s.externalContext(ctx)
	defer cancel()
	if err := s.availability.ConfirmSlotBooking(externalCtx, ConfirmSlotBookingInput{
		SlotID:    booking.SlotID,
		BookingID: booking.BookingID,
	}); err != nil {
		return Booking{}, fmt.Errorf("%w: confirmar slot: %v", ErrExternalDependency, err)
	}

	now := s.clock().UTC()
	confirmationCode := "MED-" + booking.BookingID[:min(len(booking.BookingID), 8)]
	payload, err := json.Marshal(map[string]string{
		"payment_id": input.PaymentID,
		"status":     "CONFIRMED",
	})
	if err != nil {
		return Booking{}, err
	}

	return s.repo.UpdateBookingStatus(ctx, UpdateBookingStatusInput{
		BookingID:        booking.BookingID,
		Status:           StatusConfirmed,
		PaymentID:        input.PaymentID,
		ConfirmationCode: confirmationCode,
		UpdatedAt:        now,
		ConfirmedAt:      &now,
		Event: BookingEvent{
			EventID:   uuid.NewString(),
			BookingID: booking.BookingID,
			EventType: EventConfirmed,
			Payload:   payload,
			CreatedAt: now,
		},
	})
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

func (s *bookingService) externalContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.externalCallTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, s.externalCallTimeout)
}

func (status Status) String() string {
	switch status {
	case StatusPendingPayment:
		return "PENDING_PAYMENT"
	case StatusConfirmed:
		return "CONFIRMED"
	case StatusCancelled:
		return "CANCELLED"
	case StatusExpired:
		return "EXPIRED"
	default:
		return "UNSPECIFIED"
	}
}

func (status PaymentStatus) IsApproved() bool {
	return status == PaymentStatusApproved || status == PaymentStatusCompleted
}

type noopAvailabilityClient struct{}

func (noopAvailabilityClient) HoldSlot(context.Context, HoldSlotInput) error {
	return nil
}

func (noopAvailabilityClient) ReleaseHeldSlot(context.Context, ReleaseHeldSlotInput) error {
	return nil
}

func (noopAvailabilityClient) ConfirmSlotBooking(context.Context, ConfirmSlotBookingInput) error {
	return nil
}

type noopPaymentClient struct{}

func (noopPaymentClient) GetPayment(_ context.Context, paymentID string) (PaymentDetails, error) {
	return PaymentDetails{PaymentID: paymentID, Status: PaymentStatusApproved}, nil
}

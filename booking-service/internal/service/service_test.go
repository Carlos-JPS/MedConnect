package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	createdBooking Booking
	createdEvent   BookingEvent
	booking        Booking
	updatedInput   UpdateBookingStatusInput
	createCalls    int
	updateCalls    int
}

func (r *fakeRepository) CreateBooking(_ context.Context, booking Booking, event BookingEvent) (Booking, error) {
	r.createCalls++
	r.createdBooking = booking
	r.createdEvent = event
	return booking, nil
}

func (r *fakeRepository) GetBooking(_ context.Context, bookingID string) (Booking, error) {
	if r.booking.BookingID != "" {
		return r.booking, nil
	}
	return Booking{BookingID: bookingID}, nil
}

func (r *fakeRepository) ListBookingEvents(_ context.Context, bookingID string) ([]BookingEvent, error) {
	return []BookingEvent{{BookingID: bookingID, EventType: EventCreated}}, nil
}

func (r *fakeRepository) ListBookingsByPatient(_ context.Context, patientID string, status Status) ([]Booking, error) {
	return []Booking{{PatientID: patientID, Status: status}}, nil
}

func (r *fakeRepository) UpdateBookingStatus(_ context.Context, input UpdateBookingStatusInput) (Booking, error) {
	r.updateCalls++
	r.updatedInput = input
	return Booking{
		BookingID:        input.BookingID,
		Status:           input.Status,
		PaymentID:        input.PaymentID,
		ConfirmationCode: input.ConfirmationCode,
		UpdatedAt:        input.UpdatedAt,
		ConfirmedAt:      input.ConfirmedAt,
		CancelledAt:      input.CancelledAt,
	}, nil
}

type fakeAvailabilityClient struct {
	holdInput    HoldSlotInput
	releaseInput ReleaseHeldSlotInput
	confirmInput ConfirmSlotBookingInput
	holdErr      error
	releaseErr   error
	confirmErr   error
	blockHold    bool
}

func (c *fakeAvailabilityClient) HoldSlot(ctx context.Context, input HoldSlotInput) error {
	c.holdInput = input
	if c.blockHold {
		<-ctx.Done()
		return ctx.Err()
	}
	return c.holdErr
}

func (c *fakeAvailabilityClient) ReleaseHeldSlot(_ context.Context, input ReleaseHeldSlotInput) error {
	c.releaseInput = input
	return c.releaseErr
}

func (c *fakeAvailabilityClient) ConfirmSlotBooking(_ context.Context, input ConfirmSlotBookingInput) error {
	c.confirmInput = input
	return c.confirmErr
}

type fakePaymentClient struct {
	status PaymentStatus
	err    error
	id     string
}

func (c *fakePaymentClient) GetPaymentStatus(_ context.Context, paymentID string) (PaymentStatus, error) {
	c.id = paymentID
	return c.status, c.err
}

func TestCreateBookingPersistsPendingAppointmentWithCreatedEvent(t *testing.T) {
	now := time.Date(2026, time.May, 3, 20, 0, 0, 0, time.UTC)
	repo := &fakeRepository{}
	availability := &fakeAvailabilityClient{}
	svc := NewBookingService(
		repo,
		WithClock(func() time.Time { return now }),
		WithAvailabilityClient(availability),
	)

	booking, err := svc.CreateBooking(context.Background(), CreateBookingInput{
		PatientID: "patient-1",
		DoctorID:  "doctor-1",
		SlotID:    "slot-1",
		Notes:     "Control anual",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if booking.BookingID == "" {
		t.Fatal("expected booking id to be generated")
	}
	if booking.Status != StatusPendingPayment {
		t.Fatalf("expected pending payment status, got %v", booking.Status)
	}
	if booking.ReservedUntil != now.Add(defaultReservationTTL) {
		t.Fatalf("expected reserved_until to use default ttl")
	}
	if repo.createdEvent.EventType != EventCreated {
		t.Fatalf("expected CREATED event, got %v", repo.createdEvent.EventType)
	}
	if repo.createdEvent.BookingID != booking.BookingID {
		t.Fatalf("expected event booking id to match created booking")
	}
	if availability.holdInput.SlotID != "slot-1" {
		t.Fatalf("expected availability hold for slot-1, got %q", availability.holdInput.SlotID)
	}
	if availability.holdInput.BookingID != booking.BookingID {
		t.Fatalf("expected availability hold booking id to match created booking")
	}
	if availability.holdInput.HeldUntil != booking.ReservedUntil {
		t.Fatalf("expected availability hold until booking reserved_until")
	}
}

func TestCreateBookingReturnsErrorWhenAvailabilityTimesOut(t *testing.T) {
	repo := &fakeRepository{}
	availability := &fakeAvailabilityClient{blockHold: true}
	svc := NewBookingService(
		repo,
		WithAvailabilityClient(availability),
		WithExternalCallTimeout(time.Nanosecond),
	)

	_, err := svc.CreateBooking(context.Background(), CreateBookingInput{
		PatientID: "patient-1",
		DoctorID:  "doctor-1",
		SlotID:    "slot-1",
	})

	if err == nil {
		t.Fatal("expected availability timeout error")
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected booking not to be persisted after availability timeout")
	}
}

func TestCancelBookingReleasesSlotAndMarksAppointmentCancelled(t *testing.T) {
	now := time.Date(2026, time.May, 3, 21, 0, 0, 0, time.UTC)
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusPendingPayment},
	}
	availability := &fakeAvailabilityClient{}
	svc := NewBookingService(
		repo,
		WithClock(func() time.Time { return now }),
		WithAvailabilityClient(availability),
	)

	booking, err := svc.CancelBooking(context.Background(), CancelBookingInput{
		BookingID: "booking-1",
		Reason:    "patient request",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if availability.releaseInput.SlotID != "slot-1" {
		t.Fatalf("expected held slot release for slot-1, got %q", availability.releaseInput.SlotID)
	}
	if booking.Status != StatusCancelled {
		t.Fatalf("expected cancelled booking, got %v", booking.Status)
	}
	if repo.updatedInput.Event.EventType != EventCancelled {
		t.Fatalf("expected CANCELLED event, got %v", repo.updatedInput.Event.EventType)
	}
	if repo.updatedInput.CancelledAt == nil || !repo.updatedInput.CancelledAt.Equal(now) {
		t.Fatalf("expected cancelled_at to use clock")
	}
}

func TestConfirmBookingValidatesPaymentAndConfirmsSlot(t *testing.T) {
	now := time.Date(2026, time.May, 3, 21, 30, 0, 0, time.UTC)
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusPendingPayment},
	}
	availability := &fakeAvailabilityClient{}
	payment := &fakePaymentClient{status: PaymentStatusApproved}
	svc := NewBookingService(
		repo,
		WithClock(func() time.Time { return now }),
		WithAvailabilityClient(availability),
		WithPaymentClient(payment),
	)

	booking, err := svc.ConfirmBooking(context.Background(), ConfirmBookingInput{
		BookingID: "booking-1",
		PaymentID: "payment-1",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if payment.id != "payment-1" {
		t.Fatalf("expected payment validation for payment-1, got %q", payment.id)
	}
	if availability.confirmInput.SlotID != "slot-1" {
		t.Fatalf("expected slot confirmation for slot-1, got %q", availability.confirmInput.SlotID)
	}
	if booking.Status != StatusConfirmed {
		t.Fatalf("expected confirmed booking, got %v", booking.Status)
	}
	if repo.updatedInput.Event.EventType != EventConfirmed {
		t.Fatalf("expected CONFIRMED event, got %v", repo.updatedInput.Event.EventType)
	}
	if repo.updatedInput.PaymentID != "payment-1" {
		t.Fatalf("expected payment id to be persisted")
	}
	if repo.updatedInput.ConfirmedAt == nil || !repo.updatedInput.ConfirmedAt.Equal(now) {
		t.Fatalf("expected confirmed_at to use clock")
	}
}

func TestConfirmBookingRejectsUnapprovedPaymentWithoutUpdatingAppointment(t *testing.T) {
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusPendingPayment},
	}
	payment := &fakePaymentClient{status: PaymentStatusRejected}
	svc := NewBookingService(repo, WithPaymentClient(payment))

	_, err := svc.ConfirmBooking(context.Background(), ConfirmBookingInput{
		BookingID: "booking-1",
		PaymentID: "payment-1",
	})

	if err == nil {
		t.Fatal("expected rejected payment error")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected appointment not to be updated after rejected payment")
	}
}

func TestCancelBookingReturnsControlledErrorWhenAvailabilityFails(t *testing.T) {
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusPendingPayment},
	}
	availability := &fakeAvailabilityClient{releaseErr: errors.New("availability down")}
	svc := NewBookingService(repo, WithAvailabilityClient(availability))

	_, err := svc.CancelBooking(context.Background(), CancelBookingInput{BookingID: "booking-1"})

	if err == nil {
		t.Fatal("expected availability error")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected appointment not to be updated when availability release fails")
	}
}

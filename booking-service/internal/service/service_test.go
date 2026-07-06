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
	createErr      error
	updateErr      error
	createCalls    int
	updateCalls    int
}

func (r *fakeRepository) CreateBooking(_ context.Context, booking Booking, event BookingEvent) (Booking, error) {
	r.createCalls++
	r.createdBooking = booking
	r.createdEvent = event
	if r.createErr != nil {
		return Booking{}, r.createErr
	}
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
	if r.updateErr != nil {
		return Booking{}, r.updateErr
	}
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
	releaseCalls int
	confirmCalls int
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
	c.releaseCalls++
	c.releaseInput = input
	return c.releaseErr
}

func (c *fakeAvailabilityClient) ConfirmSlotBooking(_ context.Context, input ConfirmSlotBookingInput) error {
	c.confirmCalls++
	c.confirmInput = input
	return c.confirmErr
}

type fakePaymentClient struct {
	status    PaymentStatus
	bookingID string
	err       error
	id        string
	calls     int
}

func (c *fakePaymentClient) CreatePayment(_ context.Context, input CreatePaymentInput) (PaymentDetails, error) {
	if c.err != nil {
		return PaymentDetails{}, c.err
	}
	return PaymentDetails{
		PaymentID: "payment-1",
		BookingID: input.BookingID,
		UserID:    input.UserID,
		Amount:    input.Amount,
		Currency:  input.Currency,
		Status:    PaymentStatusPending,
	}, nil
}

func (c *fakePaymentClient) ProcessPayment(_ context.Context, input ProcessPaymentInput) (ProcessPaymentResult, error) {
	if c.err != nil {
		return ProcessPaymentResult{}, c.err
	}
	return ProcessPaymentResult{PaymentID: input.PaymentID, TransactionID: "txn-1", Status: c.status}, nil
}

func (c *fakePaymentClient) GetPayment(_ context.Context, paymentID string) (PaymentDetails, error) {
	c.calls++
	c.id = paymentID
	if c.err != nil {
		return PaymentDetails{}, c.err
	}
	return PaymentDetails{
		PaymentID: paymentID,
		BookingID: c.bookingID,
		Status:    c.status,
	}, nil
}

func (c *fakePaymentClient) GetPaymentByBooking(_ context.Context, bookingID string) (PaymentDetails, error) {
	if c.err != nil {
		return PaymentDetails{}, c.err
	}
	return PaymentDetails{PaymentID: "payment-1", BookingID: bookingID, Status: c.status}, nil
}

func (c *fakePaymentClient) RefundPayment(_ context.Context, input RefundPaymentInput) (RefundDetails, error) {
	if c.err != nil {
		return RefundDetails{}, c.err
	}
	return RefundDetails{RefundID: "refund-1", PaymentID: input.PaymentID, Amount: input.Amount, Reason: input.Reason, Status: PaymentStatusRefunded}, nil
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

func TestCreateBookingReleasesHeldSlotWhenRepositoryCreateFails(t *testing.T) {
	createErr := errors.New("active booking already exists for slot")
	repo := &fakeRepository{createErr: createErr}
	availability := &fakeAvailabilityClient{}
	svc := NewBookingService(repo, WithAvailabilityClient(availability))

	_, err := svc.CreateBooking(context.Background(), CreateBookingInput{
		PatientID: "patient-1",
		DoctorID:  "doctor-1",
		SlotID:    "slot-1",
	})

	if !errors.Is(err, createErr) {
		t.Fatalf("expected original repository error, got %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected repository create to be called once, got %d", repo.createCalls)
	}
	if availability.releaseCalls != 1 {
		t.Fatalf("expected held slot compensation release, got %d calls", availability.releaseCalls)
	}
	if availability.releaseInput.SlotID != "slot-1" {
		t.Fatalf("expected release for slot-1, got %q", availability.releaseInput.SlotID)
	}
	if availability.releaseInput.BookingID == "" {
		t.Fatal("expected release booking id to be generated")
	}
	if availability.releaseInput.BookingID != availability.holdInput.BookingID {
		t.Fatalf("expected release booking id %q to match held booking id %q", availability.releaseInput.BookingID, availability.holdInput.BookingID)
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
	payment := &fakePaymentClient{status: PaymentStatusApproved, bookingID: "booking-1"}
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
	payment := &fakePaymentClient{status: PaymentStatusRejected, bookingID: "booking-1"}
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

func TestConfirmBookingRejectsPaymentForDifferentBooking(t *testing.T) {
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusPendingPayment},
	}
	availability := &fakeAvailabilityClient{}
	payment := &fakePaymentClient{status: PaymentStatusApproved, bookingID: "booking-2"}
	svc := NewBookingService(
		repo,
		WithAvailabilityClient(availability),
		WithPaymentClient(payment),
	)

	_, err := svc.ConfirmBooking(context.Background(), ConfirmBookingInput{
		BookingID: "booking-1",
		PaymentID: "payment-1",
	})

	if err == nil {
		t.Fatal("expected mismatched payment error")
	}
	if availability.confirmCalls != 0 {
		t.Fatalf("expected no slot confirmation for mismatched payment")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected appointment not to be updated for mismatched payment")
	}
}

func TestConfirmBookingRejectsPaymentWithoutBookingReference(t *testing.T) {
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusPendingPayment},
	}
	availability := &fakeAvailabilityClient{}
	payment := &fakePaymentClient{status: PaymentStatusApproved}
	svc := NewBookingService(
		repo,
		WithAvailabilityClient(availability),
		WithPaymentClient(payment),
	)

	_, err := svc.ConfirmBooking(context.Background(), ConfirmBookingInput{
		BookingID: "booking-1",
		PaymentID: "payment-1",
	})

	if err == nil {
		t.Fatal("expected payment booking reference error")
	}
	if availability.confirmCalls != 0 {
		t.Fatalf("expected no slot confirmation for payment without booking reference")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected appointment not to be updated for payment without booking reference")
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

func TestCancelBookingReturnsExistingBookingWhenAlreadyCancelled(t *testing.T) {
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusCancelled},
	}
	availability := &fakeAvailabilityClient{releaseErr: errors.New("should not release")}
	svc := NewBookingService(repo, WithAvailabilityClient(availability))

	booking, err := svc.CancelBooking(context.Background(), CancelBookingInput{BookingID: "booking-1"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if booking.Status != StatusCancelled {
		t.Fatalf("expected cancelled booking, got %v", booking.Status)
	}
	if availability.releaseCalls != 0 {
		t.Fatalf("expected no availability release for already cancelled booking")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected no repository update for already cancelled booking")
	}
}

func TestCancelBookingRejectsConfirmedBookingWithoutExternalCalls(t *testing.T) {
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusConfirmed},
	}
	availability := &fakeAvailabilityClient{}
	svc := NewBookingService(repo, WithAvailabilityClient(availability))

	_, err := svc.CancelBooking(context.Background(), CancelBookingInput{BookingID: "booking-1"})

	if err == nil {
		t.Fatal("expected invalid state error")
	}
	if availability.releaseCalls != 0 {
		t.Fatalf("expected no availability release for confirmed booking cancellation")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected no repository update for confirmed booking cancellation")
	}
}

func TestConfirmBookingReturnsExistingBookingWhenAlreadyConfirmed(t *testing.T) {
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusConfirmed},
	}
	availability := &fakeAvailabilityClient{confirmErr: errors.New("should not confirm")}
	payment := &fakePaymentClient{err: errors.New("should not validate")}
	svc := NewBookingService(
		repo,
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
	if booking.Status != StatusConfirmed {
		t.Fatalf("expected confirmed booking, got %v", booking.Status)
	}
	if payment.calls != 0 {
		t.Fatalf("expected no payment validation for already confirmed booking")
	}
	if availability.confirmCalls != 0 {
		t.Fatalf("expected no availability confirmation for already confirmed booking")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected no repository update for already confirmed booking")
	}
}

func TestConfirmBookingRejectsCancelledBookingWithoutExternalCalls(t *testing.T) {
	repo := &fakeRepository{
		booking: Booking{BookingID: "booking-1", SlotID: "slot-1", Status: StatusCancelled},
	}
	availability := &fakeAvailabilityClient{confirmErr: errors.New("should not confirm")}
	payment := &fakePaymentClient{err: errors.New("should not validate")}
	svc := NewBookingService(
		repo,
		WithAvailabilityClient(availability),
		WithPaymentClient(payment),
	)

	_, err := svc.ConfirmBooking(context.Background(), ConfirmBookingInput{
		BookingID: "booking-1",
		PaymentID: "payment-1",
	})

	if err == nil {
		t.Fatal("expected invalid state error")
	}
	if payment.calls != 0 {
		t.Fatalf("expected no payment validation for cancelled booking")
	}
	if availability.confirmCalls != 0 {
		t.Fatalf("expected no availability confirmation for cancelled booking")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected no repository update for cancelled booking")
	}
}

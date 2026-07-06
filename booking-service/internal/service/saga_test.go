package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeSagaRepository struct {
	saga    BookingSaga
	events  []BookingSagaEvent
	created int
	updated int
}

func (r *fakeSagaRepository) CreateSaga(_ context.Context, saga BookingSaga, event BookingSagaEvent) (BookingSaga, error) {
	r.created++
	r.saga = saga
	r.events = append(r.events, event)
	return saga, nil
}

func (r *fakeSagaRepository) UpdateSaga(_ context.Context, saga BookingSaga, event BookingSagaEvent) (BookingSaga, error) {
	r.updated++
	r.saga = saga
	r.events = append(r.events, event)
	return saga, nil
}

func (r *fakeSagaRepository) GetSaga(context.Context, string) (BookingSaga, error) {
	return r.saga, nil
}

func (r *fakeSagaRepository) ListSagaEvents(context.Context, string) ([]BookingSagaEvent, error) {
	return r.events, nil
}

func (r *fakeSagaRepository) hasStatus(status SagaStatus) bool {
	for _, event := range r.events {
		if event.Status == status {
			return true
		}
	}
	return false
}

type sagaPaymentClient struct {
	processStatus PaymentStatus
	createErr     error
	processErr    error
	refundErr     error
	existing      PaymentDetails
	createCalls   int
	processCalls  int
	refundCalls   int
	lookupCalls   int
	refundInput   RefundPaymentInput
}

func (c *sagaPaymentClient) CreatePayment(_ context.Context, input CreatePaymentInput) (PaymentDetails, error) {
	c.createCalls++
	if c.createErr != nil {
		return PaymentDetails{}, c.createErr
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

func (c *sagaPaymentClient) ProcessPayment(_ context.Context, input ProcessPaymentInput) (ProcessPaymentResult, error) {
	c.processCalls++
	if c.processErr != nil {
		return ProcessPaymentResult{}, c.processErr
	}
	status := c.processStatus
	if status == PaymentStatusUnspecified {
		status = PaymentStatusCompleted
	}
	return ProcessPaymentResult{PaymentID: input.PaymentID, TransactionID: "txn-1", Status: status}, nil
}

func (c *sagaPaymentClient) GetPayment(context.Context, string) (PaymentDetails, error) {
	return PaymentDetails{}, errors.New("not used by saga orchestrator")
}

func (c *sagaPaymentClient) GetPaymentByBooking(_ context.Context, bookingID string) (PaymentDetails, error) {
	c.lookupCalls++
	if c.existing.PaymentID != "" {
		if c.existing.BookingID == "" {
			c.existing.BookingID = bookingID
		}
		return c.existing, nil
	}
	return PaymentDetails{}, errors.New("not used by saga orchestrator")
}

func (c *sagaPaymentClient) RefundPayment(_ context.Context, input RefundPaymentInput) (RefundDetails, error) {
	c.refundCalls++
	c.refundInput = input
	if c.refundErr != nil {
		return RefundDetails{}, c.refundErr
	}
	return RefundDetails{RefundID: "refund-1", PaymentID: input.PaymentID, Amount: input.Amount, Reason: input.Reason, Status: PaymentStatusRefunded}, nil
}

func TestBookingSagaOrchestratorCompletesHappyPath(t *testing.T) {
	now := time.Date(2026, time.July, 6, 10, 0, 0, 0, time.UTC)
	bookingRepo := &fakeRepository{}
	sagaRepo := &fakeSagaRepository{}
	availability := &fakeAvailabilityClient{}
	payment := &sagaPaymentClient{processStatus: PaymentStatusCompleted}
	orchestrator := NewBookingSagaOrchestrator(
		bookingRepo,
		sagaRepo,
		WithSagaClock(func() time.Time { return now }),
		WithSagaAvailabilityClient(availability),
		WithSagaPaymentClient(payment),
		WithSagaRetryPolicy(0, 0),
	)

	result, err := orchestrator.StartBookingSaga(context.Background(), validStartSagaInput())
	if err != nil {
		t.Fatalf("expected saga success, got %v", err)
	}

	if result.Saga.Status != SagaStatusCompleted {
		t.Fatalf("expected saga completed, got %s", result.Saga.Status)
	}
	if result.Booking.Status != StatusConfirmed {
		t.Fatalf("expected confirmed booking, got %s", result.Booking.Status)
	}
	if bookingRepo.createCalls != 1 || bookingRepo.updateCalls != 1 {
		t.Fatalf("expected one booking create and one update, got create=%d update=%d", bookingRepo.createCalls, bookingRepo.updateCalls)
	}
	if availability.confirmCalls != 1 || availability.releaseCalls != 0 {
		t.Fatalf("expected slot confirmed without release, confirm=%d release=%d", availability.confirmCalls, availability.releaseCalls)
	}
	if payment.createCalls != 1 || payment.processCalls != 1 || payment.refundCalls != 0 {
		t.Fatalf("unexpected payment calls create=%d process=%d refund=%d", payment.createCalls, payment.processCalls, payment.refundCalls)
	}
	for _, status := range []SagaStatus{SagaStatusStarted, SagaStatusSlotHeld, SagaStatusBookingCreated, SagaStatusPaymentCreated, SagaStatusPaymentCompleted, SagaStatusSlotConfirmed, SagaStatusCompleted} {
		if !sagaRepo.hasStatus(status) {
			t.Fatalf("expected saga event with status %s", status)
		}
	}
}

func TestBookingSagaOrchestratorCompensatesRejectedPayment(t *testing.T) {
	bookingRepo := &fakeRepository{}
	sagaRepo := &fakeSagaRepository{}
	availability := &fakeAvailabilityClient{}
	payment := &sagaPaymentClient{processStatus: PaymentStatusRejected}
	orchestrator := NewBookingSagaOrchestrator(
		bookingRepo,
		sagaRepo,
		WithSagaAvailabilityClient(availability),
		WithSagaPaymentClient(payment),
		WithSagaRetryPolicy(0, 0),
	)

	result, err := orchestrator.StartBookingSaga(context.Background(), validStartSagaInput())
	if err == nil {
		t.Fatal("expected rejected payment error")
	}

	if result.Saga.Status != SagaStatusCompensated {
		t.Fatalf("expected compensated saga, got %s", result.Saga.Status)
	}
	if payment.refundCalls != 0 {
		t.Fatalf("expected no refund for rejected payment, got %d", payment.refundCalls)
	}
	if availability.releaseCalls != 1 {
		t.Fatalf("expected slot release compensation, got %d", availability.releaseCalls)
	}
	if bookingRepo.updatedInput.Status != StatusCancelled {
		t.Fatalf("expected booking cancellation, got %s", bookingRepo.updatedInput.Status)
	}
	if !sagaRepo.hasStatus(SagaStatusCompensating) || !sagaRepo.hasStatus(SagaStatusCompensated) {
		t.Fatalf("expected compensating and compensated saga events")
	}
}

func TestBookingSagaOrchestratorReusesExistingPaymentByBooking(t *testing.T) {
	bookingRepo := &fakeRepository{}
	sagaRepo := &fakeSagaRepository{}
	availability := &fakeAvailabilityClient{}
	payment := &sagaPaymentClient{
		processStatus: PaymentStatusCompleted,
		existing: PaymentDetails{
			PaymentID: "payment-existing",
			Amount:    15000,
			Currency:  "CLP",
			Status:    PaymentStatusPending,
		},
	}
	orchestrator := NewBookingSagaOrchestrator(
		bookingRepo,
		sagaRepo,
		WithSagaAvailabilityClient(availability),
		WithSagaPaymentClient(payment),
		WithSagaRetryPolicy(0, 0),
	)

	result, err := orchestrator.StartBookingSaga(context.Background(), validStartSagaInput())
	if err != nil {
		t.Fatalf("expected saga success, got %v", err)
	}

	if payment.lookupCalls != 1 {
		t.Fatalf("expected payment lookup by booking, got %d", payment.lookupCalls)
	}
	if payment.createCalls != 0 {
		t.Fatalf("expected existing payment to avoid create, got %d creates", payment.createCalls)
	}
	if result.Payment.PaymentID != "payment-existing" {
		t.Fatalf("expected existing payment in result, got %q", result.Payment.PaymentID)
	}
}

func TestBookingSagaOrchestratorRefundsWhenSlotConfirmationFails(t *testing.T) {
	bookingRepo := &fakeRepository{}
	sagaRepo := &fakeSagaRepository{}
	availability := &fakeAvailabilityClient{confirmErr: errors.New("availability down")}
	payment := &sagaPaymentClient{processStatus: PaymentStatusCompleted}
	orchestrator := NewBookingSagaOrchestrator(
		bookingRepo,
		sagaRepo,
		WithSagaAvailabilityClient(availability),
		WithSagaPaymentClient(payment),
		WithSagaRetryPolicy(0, 0),
	)

	result, err := orchestrator.StartBookingSaga(context.Background(), validStartSagaInput())
	if err == nil {
		t.Fatal("expected slot confirmation error")
	}

	if result.Saga.Status != SagaStatusCompensated {
		t.Fatalf("expected compensated saga, got %s", result.Saga.Status)
	}
	if payment.refundCalls != 1 {
		t.Fatalf("expected payment refund, got %d", payment.refundCalls)
	}
	if payment.refundInput.Amount != 15000 {
		t.Fatalf("expected refund amount 15000, got %.2f", payment.refundInput.Amount)
	}
	if availability.releaseCalls != 1 {
		t.Fatalf("expected slot release, got %d", availability.releaseCalls)
	}
	if bookingRepo.updatedInput.Status != StatusCancelled {
		t.Fatalf("expected booking cancellation, got %s", bookingRepo.updatedInput.Status)
	}
}

func TestBookingSagaOrchestratorMarksCompensationFailedWhenReleaseFails(t *testing.T) {
	createErr := errors.New("booking insert failed")
	bookingRepo := &fakeRepository{createErr: createErr}
	sagaRepo := &fakeSagaRepository{}
	availability := &fakeAvailabilityClient{releaseErr: errors.New("release failed")}
	payment := &sagaPaymentClient{}
	orchestrator := NewBookingSagaOrchestrator(
		bookingRepo,
		sagaRepo,
		WithSagaAvailabilityClient(availability),
		WithSagaPaymentClient(payment),
		WithSagaRetryPolicy(0, 0),
	)

	result, err := orchestrator.StartBookingSaga(context.Background(), validStartSagaInput())
	if err == nil {
		t.Fatal("expected create booking error")
	}

	if result.Saga.Status != SagaStatusCompensationFailed {
		t.Fatalf("expected compensation failed saga, got %s", result.Saga.Status)
	}
	if availability.releaseCalls != 1 {
		t.Fatalf("expected one release attempt, got %d", availability.releaseCalls)
	}
	if payment.createCalls != 0 {
		t.Fatalf("expected payment not to be created after booking insert failure")
	}
	if !sagaRepo.hasStatus(SagaStatusCompensationFailed) {
		t.Fatal("expected compensation failed saga event")
	}
}

func validStartSagaInput() StartBookingSagaInput {
	return StartBookingSagaInput{
		PatientID:       "46bd4a6f-6a4d-4e81-ae7c-c9d7ac05b235",
		DoctorID:        "7e0d2ab1-164e-4a28-8b95-f24293dd0e91",
		SlotID:          "0f5c2b6a-1a87-4b7e-ae2c-37ef2f9f1c21",
		Notes:           "Control de rutina",
		Amount:          15000,
		Currency:        "CLP",
		PaymentMethodID: "method-demo",
	}
}

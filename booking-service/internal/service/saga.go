package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	grpcmeta "github.com/MedConnect/booking-service/internal/clients/metadata"
	"github.com/google/uuid"
)

const (
	defaultSagaMaxRetries = 3
	defaultSagaRetryDelay = 200 * time.Millisecond
)

type SagaStatus string

const (
	SagaStatusStarted            SagaStatus = "STARTED"
	SagaStatusSlotHeld           SagaStatus = "SLOT_HELD"
	SagaStatusBookingCreated     SagaStatus = "BOOKING_CREATED"
	SagaStatusPaymentCreated     SagaStatus = "PAYMENT_CREATED"
	SagaStatusPaymentCompleted   SagaStatus = "PAYMENT_COMPLETED"
	SagaStatusSlotConfirmed      SagaStatus = "SLOT_CONFIRMED"
	SagaStatusCompleted          SagaStatus = "COMPLETED"
	SagaStatusCompensating       SagaStatus = "COMPENSATING"
	SagaStatusCompensated        SagaStatus = "COMPENSATED"
	SagaStatusFailed             SagaStatus = "FAILED"
	SagaStatusCompensationFailed SagaStatus = "COMPENSATION_FAILED"
)

type SagaCompensationStatus string

const (
	SagaCompensationNotRequired SagaCompensationStatus = "NOT_REQUIRED"
	SagaCompensationPending     SagaCompensationStatus = "PENDING"
	SagaCompensationInProgress  SagaCompensationStatus = "IN_PROGRESS"
	SagaCompensationCompleted   SagaCompensationStatus = "COMPLETED"
	SagaCompensationFailed      SagaCompensationStatus = "FAILED"
)

type SagaStep string

const (
	SagaStepStart          SagaStep = "START"
	SagaStepHoldSlot       SagaStep = "HOLD_SLOT"
	SagaStepCreateBooking  SagaStep = "CREATE_BOOKING"
	SagaStepCreatePayment  SagaStep = "CREATE_PAYMENT"
	SagaStepProcessPayment SagaStep = "PROCESS_PAYMENT"
	SagaStepConfirmSlot    SagaStep = "CONFIRM_SLOT"
	SagaStepConfirmBooking SagaStep = "CONFIRM_BOOKING"
	SagaStepCompensate     SagaStep = "COMPENSATE"
)

type BookingSaga struct {
	SagaID             string
	BookingID          string
	PaymentID          string
	PatientID          string
	DoctorID           string
	SlotID             string
	Status             SagaStatus
	CurrentStep        SagaStep
	CompensationStatus SagaCompensationStatus
	RetryCount         int
	LastError          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CompletedAt        *time.Time
}

type BookingSagaEvent struct {
	EventID      string
	SagaID       string
	EventType    string
	Step         SagaStep
	Status       SagaStatus
	Payload      []byte
	ErrorMessage string
	CreatedAt    time.Time
}

type BookingSagaDetails struct {
	Saga   BookingSaga
	Events []BookingSagaEvent
}

type StartBookingSagaInput struct {
	PatientID       string
	DoctorID        string
	SlotID          string
	Notes           string
	Amount          float64
	Currency        string
	PaymentMethodID string
}

type BookingSagaResult struct {
	Saga    BookingSaga
	Booking Booking
	Payment PaymentDetails
}

type SagaRepository interface {
	CreateSaga(ctx context.Context, saga BookingSaga, event BookingSagaEvent) (BookingSaga, error)
	UpdateSaga(ctx context.Context, saga BookingSaga, event BookingSagaEvent) (BookingSaga, error)
	GetSaga(ctx context.Context, sagaID string) (BookingSaga, error)
	ListSagaEvents(ctx context.Context, sagaID string) ([]BookingSagaEvent, error)
}

type BookingSagaService interface {
	StartBookingSaga(ctx context.Context, input StartBookingSagaInput) (BookingSagaResult, error)
	GetBookingSaga(ctx context.Context, sagaID string) (BookingSagaDetails, error)
}

type bookingSagaOrchestrator struct {
	bookingRepo         Repository
	sagaRepo            SagaRepository
	availability        AvailabilityClient
	payment             PaymentClient
	clock               func() time.Time
	externalCallTimeout time.Duration
	sagaMaxRetries      int
	sagaRetryDelay      time.Duration
}

type SagaOption func(*bookingSagaOrchestrator)

func WithSagaClock(clock func() time.Time) SagaOption {
	return func(o *bookingSagaOrchestrator) {
		o.clock = clock
	}
}

func WithSagaAvailabilityClient(client AvailabilityClient) SagaOption {
	return func(o *bookingSagaOrchestrator) {
		o.availability = client
	}
}

func WithSagaPaymentClient(client PaymentClient) SagaOption {
	return func(o *bookingSagaOrchestrator) {
		o.payment = client
	}
}

func WithSagaExternalCallTimeout(timeout time.Duration) SagaOption {
	return func(o *bookingSagaOrchestrator) {
		o.externalCallTimeout = timeout
	}
}

func WithSagaRetryPolicy(maxRetries int, retryDelay time.Duration) SagaOption {
	return func(o *bookingSagaOrchestrator) {
		o.sagaMaxRetries = maxRetries
		o.sagaRetryDelay = retryDelay
	}
}

func NewBookingSagaOrchestrator(bookingRepo Repository, sagaRepo SagaRepository, opts ...SagaOption) BookingSagaService {
	orchestrator := &bookingSagaOrchestrator{
		bookingRepo:         bookingRepo,
		sagaRepo:            sagaRepo,
		availability:        noopAvailabilityClient{},
		payment:             noopPaymentClient{},
		clock:               time.Now,
		externalCallTimeout: defaultExternalCallTimeout,
		sagaMaxRetries:      defaultSagaMaxRetries,
		sagaRetryDelay:      defaultSagaRetryDelay,
	}
	for _, opt := range opts {
		opt(orchestrator)
	}
	if orchestrator.sagaMaxRetries < 0 {
		orchestrator.sagaMaxRetries = defaultSagaMaxRetries
	}
	if orchestrator.sagaRetryDelay < 0 {
		orchestrator.sagaRetryDelay = defaultSagaRetryDelay
	}
	return orchestrator
}

func (o *bookingSagaOrchestrator) StartBookingSaga(ctx context.Context, input StartBookingSagaInput) (BookingSagaResult, error) {
	if err := validateStartBookingSagaInput(input); err != nil {
		return BookingSagaResult{}, err
	}

	now := o.now()
	bookingID := uuid.NewString()
	saga := BookingSaga{
		SagaID:             uuid.NewString(),
		BookingID:          bookingID,
		PatientID:          input.PatientID,
		DoctorID:           input.DoctorID,
		SlotID:             input.SlotID,
		Status:             SagaStatusStarted,
		CurrentStep:        SagaStepStart,
		CompensationStatus: SagaCompensationNotRequired,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	createdSaga, err := o.sagaRepo.CreateSaga(ctx, saga, o.sagaEvent(saga, "SAGA_STARTED", nil, ""))
	if err != nil {
		return BookingSagaResult{}, fmt.Errorf("crear saga booking: %w", err)
	}
	saga = createdSaga
	logSagaTransition(ctx, saga, "saga_started", "")

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

	if err := o.withRetry(ctx, func(attemptCtx context.Context) error {
		return o.availability.HoldSlot(attemptCtx, HoldSlotInput{
			SlotID:    input.SlotID,
			BookingID: bookingID,
			HeldUntil: booking.ReservedUntil,
		})
	}); err != nil {
		saga, _ = o.markSagaFailed(ctx, saga, SagaStepHoldSlot, err)
		return BookingSagaResult{Saga: saga}, fmt.Errorf("%w: saga hold slot: %v", ErrExternalDependency, err)
	}
	saga, err = o.updateSaga(ctx, saga, SagaStatusSlotHeld, SagaStepHoldSlot, SagaCompensationNotRequired, "", map[string]any{"slot_id": input.SlotID})
	if err != nil {
		return BookingSagaResult{Saga: saga}, err
	}

	booking, err = o.createSagaBooking(ctx, booking, saga.SagaID)
	if err != nil {
		saga, compensationErr := o.compensate(ctx, saga, sagaCompensationPlan{releaseSlot: true, reason: "booking create failed"}, err)
		return BookingSagaResult{Saga: saga}, joinSagaError(err, compensationErr)
	}
	saga, err = o.updateSaga(ctx, saga, SagaStatusBookingCreated, SagaStepCreateBooking, SagaCompensationNotRequired, "", map[string]any{"booking_id": booking.BookingID})
	if err != nil {
		return BookingSagaResult{Saga: saga, Booking: booking}, err
	}

	payment, err := o.createPayment(ctx, input, booking.BookingID)
	if err != nil {
		saga, compensationErr := o.compensate(ctx, saga, sagaCompensationPlan{cancelBooking: true, releaseSlot: true, reason: "payment create failed"}, err)
		return BookingSagaResult{Saga: saga, Booking: booking}, joinSagaError(err, compensationErr)
	}
	saga.PaymentID = payment.PaymentID
	saga, err = o.updateSaga(ctx, saga, SagaStatusPaymentCreated, SagaStepCreatePayment, SagaCompensationNotRequired, "", map[string]any{"payment_id": payment.PaymentID})
	if err != nil {
		return BookingSagaResult{Saga: saga, Booking: booking, Payment: payment}, err
	}

	processResult, err := o.processPayment(ctx, payment.PaymentID, input.PaymentMethodID)
	if err != nil {
		saga, compensationErr := o.compensate(ctx, saga, sagaCompensationPlan{cancelBooking: true, releaseSlot: true, reason: "payment process failed"}, err)
		return BookingSagaResult{Saga: saga, Booking: booking, Payment: payment}, joinSagaError(err, compensationErr)
	}
	if !processResult.Status.IsApproved() {
		err = fmt.Errorf("%w: pago %s no aprobado: %s", ErrInvalidBookingState, payment.PaymentID, processResult.Status)
		saga, compensationErr := o.compensate(ctx, saga, sagaCompensationPlan{cancelBooking: true, releaseSlot: true, reason: "payment not approved"}, err)
		return BookingSagaResult{Saga: saga, Booking: booking, Payment: payment}, joinSagaError(err, compensationErr)
	}
	payment.Status = processResult.Status
	saga, err = o.updateSaga(ctx, saga, SagaStatusPaymentCompleted, SagaStepProcessPayment, SagaCompensationNotRequired, "", map[string]any{"payment_id": payment.PaymentID, "transaction_id": processResult.TransactionID})
	if err != nil {
		return BookingSagaResult{Saga: saga, Booking: booking, Payment: payment}, err
	}

	if err := o.withRetry(ctx, func(attemptCtx context.Context) error {
		return o.availability.ConfirmSlotBooking(attemptCtx, ConfirmSlotBookingInput{SlotID: booking.SlotID, BookingID: booking.BookingID})
	}); err != nil {
		saga, compensationErr := o.compensate(ctx, saga, sagaCompensationPlan{refundPayment: true, refundAmount: payment.Amount, cancelBooking: true, releaseSlot: true, reason: "slot confirmation failed"}, err)
		return BookingSagaResult{Saga: saga, Booking: booking, Payment: payment}, joinSagaError(err, compensationErr)
	}
	saga, err = o.updateSaga(ctx, saga, SagaStatusSlotConfirmed, SagaStepConfirmSlot, SagaCompensationNotRequired, "", map[string]any{"slot_id": booking.SlotID})
	if err != nil {
		return BookingSagaResult{Saga: saga, Booking: booking, Payment: payment}, err
	}

	confirmedBooking, err := o.confirmSagaBooking(ctx, booking, payment.PaymentID, saga.SagaID)
	if err != nil {
		saga, compensationErr := o.compensate(ctx, saga, sagaCompensationPlan{refundPayment: true, refundAmount: payment.Amount, cancelBooking: true, releaseSlot: true, reason: "booking confirmation failed"}, err)
		return BookingSagaResult{Saga: saga, Booking: booking, Payment: payment}, joinSagaError(err, compensationErr)
	}
	booking = confirmedBooking

	completedAt := o.now()
	saga.CompletedAt = &completedAt
	saga, err = o.updateSaga(ctx, saga, SagaStatusCompleted, SagaStepConfirmBooking, SagaCompensationNotRequired, "", map[string]any{"booking_id": booking.BookingID, "payment_id": payment.PaymentID})
	if err != nil {
		return BookingSagaResult{Saga: saga, Booking: booking, Payment: payment}, err
	}

	return BookingSagaResult{Saga: saga, Booking: booking, Payment: payment}, nil
}

func (o *bookingSagaOrchestrator) GetBookingSaga(ctx context.Context, sagaID string) (BookingSagaDetails, error) {
	saga, err := o.sagaRepo.GetSaga(ctx, sagaID)
	if err != nil {
		return BookingSagaDetails{}, err
	}
	events, err := o.sagaRepo.ListSagaEvents(ctx, sagaID)
	if err != nil {
		return BookingSagaDetails{}, err
	}
	return BookingSagaDetails{Saga: saga, Events: events}, nil
}

func validateStartBookingSagaInput(input StartBookingSagaInput) error {
	missing := make([]string, 0, 5)
	if input.PatientID == "" {
		missing = append(missing, "patient_id")
	}
	if input.DoctorID == "" {
		missing = append(missing, "doctor_id")
	}
	if input.SlotID == "" {
		missing = append(missing, "slot_id")
	}
	if input.Currency == "" {
		missing = append(missing, "currency")
	}
	if input.PaymentMethodID == "" {
		missing = append(missing, "payment_method_id")
	}
	if len(missing) > 0 {
		return fmt.Errorf("campos obligatorios faltantes para saga: %s", strings.Join(missing, ", "))
	}
	if input.Amount <= 0 {
		return fmt.Errorf("amount debe ser mayor a cero")
	}
	return nil
}

func (o *bookingSagaOrchestrator) createSagaBooking(ctx context.Context, booking Booking, sagaID string) (Booking, error) {
	payload, err := json.Marshal(map[string]string{"status": "PENDING_PAYMENT", "slot_id": booking.SlotID, "saga_id": sagaID})
	if err != nil {
		return Booking{}, err
	}
	return o.bookingRepo.CreateBooking(ctx, booking, BookingEvent{
		EventID:   uuid.NewString(),
		BookingID: booking.BookingID,
		EventType: EventCreated,
		Payload:   payload,
		CreatedAt: o.now(),
	})
}

func (o *bookingSagaOrchestrator) createPayment(ctx context.Context, input StartBookingSagaInput, bookingID string) (PaymentDetails, error) {
	var payment PaymentDetails
	err := o.withRetry(ctx, func(attemptCtx context.Context) error {
		if existing, err := o.payment.GetPaymentByBooking(attemptCtx, bookingID); err == nil && existing.PaymentID != "" {
			payment = existing
			return nil
		}

		created, err := o.payment.CreatePayment(attemptCtx, CreatePaymentInput{BookingID: bookingID, UserID: input.PatientID, Amount: input.Amount, Currency: input.Currency})
		if err != nil {
			return err
		}
		payment = created
		return nil
	})
	return payment, err
}

func (o *bookingSagaOrchestrator) processPayment(ctx context.Context, paymentID string, paymentMethodID string) (ProcessPaymentResult, error) {
	var result ProcessPaymentResult
	err := o.withRetry(ctx, func(attemptCtx context.Context) error {
		processed, err := o.payment.ProcessPayment(attemptCtx, ProcessPaymentInput{PaymentID: paymentID, PaymentMethodID: paymentMethodID})
		if err != nil {
			return err
		}
		result = processed
		return nil
	})
	return result, err
}

func (o *bookingSagaOrchestrator) confirmSagaBooking(ctx context.Context, booking Booking, paymentID string, sagaID string) (Booking, error) {
	now := o.now()
	confirmationCode := "MED-" + booking.BookingID[:min(len(booking.BookingID), 8)]
	payload, err := json.Marshal(map[string]string{"payment_id": paymentID, "status": "CONFIRMED", "saga_id": sagaID})
	if err != nil {
		return Booking{}, err
	}
	return o.bookingRepo.UpdateBookingStatus(ctx, UpdateBookingStatusInput{
		BookingID:        booking.BookingID,
		Status:           StatusConfirmed,
		PaymentID:        paymentID,
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

func (o *bookingSagaOrchestrator) cancelSagaBooking(ctx context.Context, saga BookingSaga, reason string) error {
	now := o.now()
	payload, err := json.Marshal(map[string]string{"reason": reason, "status": "CANCELLED", "saga_id": saga.SagaID})
	if err != nil {
		return err
	}
	_, err = o.bookingRepo.UpdateBookingStatus(ctx, UpdateBookingStatusInput{
		BookingID:   saga.BookingID,
		Status:      StatusCancelled,
		UpdatedAt:   now,
		CancelledAt: &now,
		Event: BookingEvent{
			EventID:   uuid.NewString(),
			BookingID: saga.BookingID,
			EventType: EventCancelled,
			Payload:   payload,
			CreatedAt: now,
		},
	})
	return err
}

type sagaCompensationPlan struct {
	refundPayment bool
	refundAmount  float64
	cancelBooking bool
	releaseSlot   bool
	reason        string
}

func (o *bookingSagaOrchestrator) compensate(ctx context.Context, saga BookingSaga, plan sagaCompensationPlan, cause error) (BookingSaga, error) {
	updated, err := o.updateSaga(ctx, saga, SagaStatusCompensating, SagaStepCompensate, SagaCompensationInProgress, cause.Error(), map[string]any{"reason": plan.reason})
	if err == nil {
		saga = updated
	}

	var compensationErrs []error
	if plan.refundPayment && saga.PaymentID != "" {
		if err := o.withRetry(ctx, func(attemptCtx context.Context) error {
			_, err := o.payment.RefundPayment(attemptCtx, RefundPaymentInput{PaymentID: saga.PaymentID, Amount: plan.refundAmount, Reason: plan.reason})
			return err
		}); err != nil {
			compensationErrs = append(compensationErrs, fmt.Errorf("refund payment: %w", err))
		}
	}
	if plan.releaseSlot {
		if err := o.withRetry(ctx, func(attemptCtx context.Context) error {
			return o.availability.ReleaseHeldSlot(attemptCtx, ReleaseHeldSlotInput{SlotID: saga.SlotID, BookingID: saga.BookingID})
		}); err != nil {
			compensationErrs = append(compensationErrs, fmt.Errorf("release slot: %w", err))
		}
	}
	if plan.cancelBooking {
		if err := o.cancelSagaBooking(ctx, saga, plan.reason); err != nil {
			compensationErrs = append(compensationErrs, fmt.Errorf("cancel booking: %w", err))
		}
	}

	if err := errors.Join(compensationErrs...); err != nil {
		updated, updateErr := o.updateSaga(ctx, saga, SagaStatusCompensationFailed, SagaStepCompensate, SagaCompensationFailed, err.Error(), map[string]any{"reason": plan.reason})
		if updateErr == nil {
			saga = updated
		}
		return saga, err
	}

	updated, err = o.updateSaga(ctx, saga, SagaStatusCompensated, SagaStepCompensate, SagaCompensationCompleted, cause.Error(), map[string]any{"reason": plan.reason})
	if err == nil {
		saga = updated
	}
	return saga, err
}

func (o *bookingSagaOrchestrator) markSagaFailed(ctx context.Context, saga BookingSaga, step SagaStep, cause error) (BookingSaga, error) {
	completedAt := o.now()
	saga.CompletedAt = &completedAt
	return o.updateSaga(ctx, saga, SagaStatusFailed, step, SagaCompensationNotRequired, cause.Error(), nil)
}

func (o *bookingSagaOrchestrator) updateSaga(ctx context.Context, saga BookingSaga, status SagaStatus, step SagaStep, compensationStatus SagaCompensationStatus, lastError string, payload map[string]any) (BookingSaga, error) {
	saga.Status = status
	saga.CurrentStep = step
	saga.CompensationStatus = compensationStatus
	saga.LastError = lastError
	saga.UpdatedAt = o.now()
	if status == SagaStatusCompleted || status == SagaStatusCompensated || status == SagaStatusCompensationFailed || status == SagaStatusFailed {
		completedAt := saga.UpdatedAt
		saga.CompletedAt = &completedAt
	}
	updated, err := o.sagaRepo.UpdateSaga(ctx, saga, o.sagaEvent(saga, string(status), payload, lastError))
	if err != nil {
		return BookingSaga{}, err
	}
	logSagaTransition(ctx, updated, "saga_transition", lastError)
	return updated, nil
}

func (o *bookingSagaOrchestrator) sagaEvent(saga BookingSaga, eventType string, payload map[string]any, errorMessage string) BookingSagaEvent {
	encodedPayload := []byte("{}")
	if payload != nil {
		if marshaled, err := json.Marshal(payload); err == nil {
			encodedPayload = marshaled
		}
	}
	return BookingSagaEvent{
		EventID:      uuid.NewString(),
		SagaID:       saga.SagaID,
		EventType:    eventType,
		Step:         saga.CurrentStep,
		Status:       saga.Status,
		Payload:      encodedPayload,
		ErrorMessage: errorMessage,
		CreatedAt:    o.now(),
	}
}

func (o *bookingSagaOrchestrator) withRetry(ctx context.Context, operation func(context.Context) error) error {
	var lastErr error
	attempts := o.sagaMaxRetries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		attemptCtx, cancel := o.externalContext(ctx)
		err := operation(attemptCtx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < attempts-1 && o.sagaRetryDelay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(o.sagaRetryDelay):
			}
		}
	}
	return lastErr
}

func (o *bookingSagaOrchestrator) externalContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if o.externalCallTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, o.externalCallTimeout)
}

func (o *bookingSagaOrchestrator) now() time.Time {
	return o.clock().UTC()
}

func logSagaTransition(ctx context.Context, saga BookingSaga, event string, errorMessage string) {
	requestID := grpcmeta.RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = "unknown"
	}

	if errorMessage == "" {
		log.Printf(
			"service=booking-service component=saga event=%s request_id=%s saga_id=%s booking_id=%s payment_id=%s step=%s status=%s compensation_status=%s",
			event,
			requestID,
			saga.SagaID,
			saga.BookingID,
			saga.PaymentID,
			saga.CurrentStep,
			saga.Status,
			saga.CompensationStatus,
		)
		return
	}

	log.Printf(
		"service=booking-service component=saga event=%s request_id=%s saga_id=%s booking_id=%s payment_id=%s step=%s status=%s compensation_status=%s error=%q",
		event,
		requestID,
		saga.SagaID,
		saga.BookingID,
		saga.PaymentID,
		saga.CurrentStep,
		saga.Status,
		saga.CompensationStatus,
		errorMessage,
	)
}

func joinSagaError(primary error, compensation error) error {
	if compensation == nil {
		return primary
	}
	return fmt.Errorf("%w; compensacion fallida: %v", primary, compensation)
}

package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MedConnect/booking-service/internal/service"
	pb "github.com/MedConnect/booking-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeBookingService struct {
	booking   service.Booking
	createErr error
	cancelErr error
	lastID    string
}

func (f *fakeBookingService) CreateBooking(_ context.Context, input service.CreateBookingInput) (service.Booking, error) {
	if f.createErr != nil {
		return service.Booking{}, f.createErr
	}
	return service.Booking{
		PatientID: input.PatientID,
		DoctorID:  input.DoctorID,
		SlotID:    input.SlotID,
		Status:    service.StatusPendingPayment,
	}, nil
}

func (f *fakeBookingService) CancelBooking(_ context.Context, input service.CancelBookingInput) (service.Booking, error) {
	if f.cancelErr != nil {
		return service.Booking{}, f.cancelErr
	}
	return service.Booking{BookingID: input.BookingID, Status: service.StatusCancelled}, nil
}

func (f *fakeBookingService) ConfirmBooking(_ context.Context, input service.ConfirmBookingInput) (service.Booking, error) {
	return service.Booking{BookingID: input.BookingID, PaymentID: input.PaymentID, Status: service.StatusConfirmed}, nil
}

func (f *fakeBookingService) GetBooking(_ context.Context, bookingID string) (service.BookingDetails, error) {
	f.lastID = bookingID
	return service.BookingDetails{
		Booking: f.booking,
		Events: []service.BookingEvent{
			{BookingID: bookingID, EventType: service.EventCreated, CreatedAt: f.booking.CreatedAt},
		},
	}, nil
}

func (f *fakeBookingService) ListBookingsByPatient(_ context.Context, patientID string, status service.Status) ([]service.Booking, error) {
	return []service.Booking{{PatientID: patientID, Status: status}}, nil
}

func (f *fakeBookingService) UpdateBookingStatus(_ context.Context, input service.UpdateBookingStatusInput) (service.Booking, error) {
	return service.Booking{BookingID: input.BookingID, Status: input.Status}, nil
}

type fakeSagaService struct {
	result  service.BookingSagaResult
	details service.BookingSagaDetails
	err     error
	input   service.StartBookingSagaInput
	lastID  string
}

func (f *fakeSagaService) StartBookingSaga(_ context.Context, input service.StartBookingSagaInput) (service.BookingSagaResult, error) {
	f.input = input
	if f.err != nil {
		return service.BookingSagaResult{}, f.err
	}
	return f.result, nil
}

func (f *fakeSagaService) GetBookingSaga(_ context.Context, sagaID string) (service.BookingSagaDetails, error) {
	f.lastID = sagaID
	if f.err != nil {
		return service.BookingSagaDetails{}, f.err
	}
	return f.details, nil
}

func TestGetBookingDelegatesToServiceAndMapsResponse(t *testing.T) {
	createdAt := time.Date(2026, time.May, 3, 14, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(10 * time.Minute)
	reservedUntil := createdAt.Add(30 * time.Minute)
	confirmedAt := createdAt.Add(15 * time.Minute)

	svc := &fakeBookingService{
		booking: service.Booking{
			BookingID:        "booking-123",
			PatientID:        "patient-1",
			DoctorID:         "doctor-1",
			SlotID:           "slot-1",
			Status:           service.StatusConfirmed,
			PaymentID:        "payment-1",
			ConfirmationCode: "MED-001",
			Notes:            "Control anual",
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
			ReservedUntil:    reservedUntil,
			ConfirmedAt:      &confirmedAt,
		},
	}

	server := NewServer(svc, &fakeSagaService{})

	resp, err := server.GetBooking(context.Background(), &pb.GetBookingRequest{BookingId: "booking-123"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if svc.lastID != "booking-123" {
		t.Fatalf("expected service to receive booking id booking-123, got %q", svc.lastID)
	}
	if resp.GetBooking().GetStatus() != pb.BookingStatus_BOOKING_STATUS_CONFIRMED {
		t.Fatalf("expected confirmed status, got %s", resp.GetBooking().GetStatus())
	}
	if resp.GetBooking().GetReservedUntil().AsTime() != reservedUntil {
		t.Fatalf("expected reserved_until to be mapped")
	}
	if resp.GetBooking().GetConfirmedAt().AsTime() != confirmedAt {
		t.Fatalf("expected confirmed_at to be mapped")
	}
	if resp.GetBooking().GetNotes() != "Control anual" {
		t.Fatalf("expected notes to be mapped, got %q", resp.GetBooking().GetNotes())
	}
	if len(resp.GetEvents()) != 1 {
		t.Fatalf("expected one event, got %d", len(resp.GetEvents()))
	}
}

func TestCancelBookingMapsInvalidStateToFailedPrecondition(t *testing.T) {
	svc := &fakeBookingService{
		cancelErr: fmt.Errorf("%w: no se puede cancelar una reserva CONFIRMED", service.ErrInvalidBookingState),
	}
	server := NewServer(svc, &fakeSagaService{})

	_, err := server.CancelBooking(context.Background(), &pb.CancelBookingRequest{BookingId: "booking-1"})

	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %s: %v", status.Code(err), err)
	}
}

func TestCreateBookingMapsActiveSlotBookingExistsToFailedPrecondition(t *testing.T) {
	svc := &fakeBookingService{
		createErr: fmt.Errorf("%w: slot-1", service.ErrActiveSlotBookingExists),
	}
	server := NewServer(svc, &fakeSagaService{})

	_, err := server.CreateBooking(context.Background(), &pb.CreateBookingRequest{PatientId: "patient-1", DoctorId: "doctor-1", SlotId: "slot-1"})

	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %s: %v", status.Code(err), err)
	}
}

func TestStartBookingSagaDelegatesToServiceAndMapsResponse(t *testing.T) {
	now := time.Date(2026, time.July, 6, 10, 0, 0, 0, time.UTC)
	sagaSvc := &fakeSagaService{
		result: service.BookingSagaResult{
			Saga: service.BookingSaga{
				SagaID:             "saga-1",
				BookingID:          "booking-1",
				PaymentID:          "payment-1",
				PatientID:          "patient-1",
				DoctorID:           "doctor-1",
				SlotID:             "slot-1",
				Status:             service.SagaStatusCompleted,
				CurrentStep:        service.SagaStepConfirmBooking,
				CompensationStatus: service.SagaCompensationNotRequired,
				CreatedAt:          now,
				UpdatedAt:          now,
				CompletedAt:        &now,
			},
			Booking: service.Booking{BookingID: "booking-1", Status: service.StatusConfirmed, CreatedAt: now, UpdatedAt: now},
			Payment: service.PaymentDetails{PaymentID: "payment-1", Status: service.PaymentStatusCompleted},
		},
	}
	server := NewServer(&fakeBookingService{}, sagaSvc)

	resp, err := server.StartBookingSaga(context.Background(), &pb.StartBookingSagaRequest{
		PatientId:       "patient-1",
		DoctorId:        "doctor-1",
		SlotId:          "slot-1",
		Notes:           "Control",
		Amount:          15000,
		Currency:        "CLP",
		PaymentMethodId: "method-demo",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sagaSvc.input.PaymentMethodID != "method-demo" {
		t.Fatalf("expected payment method to be delegated, got %q", sagaSvc.input.PaymentMethodID)
	}
	if resp.GetSaga().GetStatus() != pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPLETED {
		t.Fatalf("expected completed saga status, got %s", resp.GetSaga().GetStatus())
	}
	if resp.GetPaymentId() != "payment-1" || resp.GetPaymentStatus() != "COMPLETED" {
		t.Fatalf("expected payment response to be mapped, got id=%q status=%q", resp.GetPaymentId(), resp.GetPaymentStatus())
	}
}

func TestGetBookingSagaDelegatesToServiceAndMapsEvents(t *testing.T) {
	now := time.Date(2026, time.July, 6, 10, 0, 0, 0, time.UTC)
	sagaSvc := &fakeSagaService{
		details: service.BookingSagaDetails{
			Saga: service.BookingSaga{SagaID: "saga-1", Status: service.SagaStatusCompleted, CreatedAt: now, UpdatedAt: now},
			Events: []service.BookingSagaEvent{
				{EventID: "event-1", SagaID: "saga-1", EventType: "COMPLETED", Step: service.SagaStepConfirmBooking, Status: service.SagaStatusCompleted, Payload: []byte(`{"ok":true}`), CreatedAt: now},
			},
		},
	}
	server := NewServer(&fakeBookingService{}, sagaSvc)

	resp, err := server.GetBookingSaga(context.Background(), &pb.GetBookingSagaRequest{SagaId: "saga-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sagaSvc.lastID != "saga-1" {
		t.Fatalf("expected saga id to be delegated, got %q", sagaSvc.lastID)
	}
	if len(resp.GetEvents()) != 1 {
		t.Fatalf("expected one saga event, got %d", len(resp.GetEvents()))
	}
	if resp.GetEvents()[0].GetStatus() != pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPLETED {
		t.Fatalf("expected completed event status, got %s", resp.GetEvents()[0].GetStatus())
	}
}

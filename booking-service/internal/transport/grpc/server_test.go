package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/MedConnect/booking-service/internal/service"
	pb "github.com/MedConnect/booking-service/pb"
)

type fakeBookingService struct {
	booking service.Booking
	lastID  string
}

func (f *fakeBookingService) CreateBooking(_ context.Context, input service.CreateBookingInput) (service.Booking, error) {
	return service.Booking{
		PatientID: input.PatientID,
		DoctorID:  input.DoctorID,
		SlotID:    input.SlotID,
		Status:    service.StatusPendingPayment,
	}, nil
}

func (f *fakeBookingService) CancelBooking(_ context.Context, input service.CancelBookingInput) (service.Booking, error) {
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

	server := NewServer(svc)

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

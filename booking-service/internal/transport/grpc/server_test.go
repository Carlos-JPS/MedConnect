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

func (f *fakeBookingService) GetBooking(_ context.Context, bookingID string) (service.Booking, error) {
	f.lastID = bookingID
	return f.booking, nil
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
}

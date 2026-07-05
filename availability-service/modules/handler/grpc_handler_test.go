package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Carlos-JPS/medconnect/availability-service/modules/repository"
	pb "github.com/Carlos-JPS/medconnect/availability-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeAvailabilityService struct {
	getSlotsErr  error
	getAgendaErr error
	holdErr      error
}

func (f fakeAvailabilityService) GetAvailableSlots(context.Context, string, time.Time, time.Time) ([]*repository.Slot, error) {
	return nil, f.getSlotsErr
}

func (f fakeAvailabilityService) HoldSlot(context.Context, string, string, time.Time) (*repository.Slot, error) {
	if f.holdErr != nil {
		return nil, f.holdErr
	}
	return &repository.Slot{ID: "slot-1", Status: repository.StatusHeld}, nil
}

func (f fakeAvailabilityService) ConfirmSlotBooking(context.Context, string, string) (*repository.Slot, error) {
	return &repository.Slot{ID: "slot-1", Status: repository.StatusBooked}, nil
}

func (f fakeAvailabilityService) ReleaseHeldSlot(context.Context, string, string) (*repository.Slot, error) {
	return &repository.Slot{ID: "slot-1", Status: repository.StatusAvailable}, nil
}

func (f fakeAvailabilityService) GetDoctorAgenda(context.Context, string, time.Time, time.Time) ([]*repository.Slot, error) {
	return nil, f.getAgendaErr
}

func TestGetAvailableSlotsMapsRepositoryFailureToUnavailable(t *testing.T) {
	handler := NewGRPCHandler(fakeAvailabilityService{getSlotsErr: errors.New("database unavailable")})

	_, err := handler.GetAvailableSlots(context.Background(), &pb.GetAvailableSlotsRequest{
		Specialty: "Cardiología",
		FromDate:  "2026-01-01T00:00:00Z",
		ToDate:    "2027-01-01T00:00:00Z",
	})

	if status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable, got %s: %v", status.Code(err), err)
	}
}

func TestHoldSlotMapsMissingDirectoryEntryToNotFound(t *testing.T) {
	handler := NewGRPCHandler(fakeAvailabilityService{holdErr: repository.ErrSlotShardNotFound})

	_, err := handler.HoldSlot(context.Background(), &pb.HoldSlotRequest{SlotId: "slot-1", BookingId: "booking-1"})

	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %s: %v", status.Code(err), err)
	}
}

func TestGetDoctorAgendaRejectsInvalidDateRange(t *testing.T) {
	handler := NewGRPCHandler(fakeAvailabilityService{})

	_, err := handler.GetDoctorAgenda(context.Background(), &pb.GetDoctorAgendaRequest{
		DoctorId: "doctor-1",
		FromDate: "2027-01-01T00:00:00Z",
		ToDate:   "2026-01-01T00:00:00Z",
	})

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s: %v", status.Code(err), err)
	}
}

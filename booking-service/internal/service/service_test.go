package service

import (
	"context"
	"testing"
	"time"
)

type fakeRepository struct {
	createdBooking Booking
	createdEvent   BookingEvent
}

func (r *fakeRepository) CreateBooking(_ context.Context, booking Booking, event BookingEvent) (Booking, error) {
	r.createdBooking = booking
	r.createdEvent = event
	return booking, nil
}

func (r *fakeRepository) GetBooking(_ context.Context, bookingID string) (Booking, error) {
	return Booking{BookingID: bookingID}, nil
}

func (r *fakeRepository) ListBookingEvents(_ context.Context, bookingID string) ([]BookingEvent, error) {
	return []BookingEvent{{BookingID: bookingID, EventType: EventCreated}}, nil
}

func (r *fakeRepository) ListBookingsByPatient(_ context.Context, patientID string, status Status) ([]Booking, error) {
	return []Booking{{PatientID: patientID, Status: status}}, nil
}

func (r *fakeRepository) UpdateBookingStatus(_ context.Context, input UpdateBookingStatusInput) (Booking, error) {
	return Booking{BookingID: input.BookingID, Status: input.Status, UpdatedAt: input.UpdatedAt}, nil
}

func TestCreateBookingPersistsPendingAppointmentWithCreatedEvent(t *testing.T) {
	now := time.Date(2026, time.May, 3, 20, 0, 0, 0, time.UTC)
	repo := &fakeRepository{}
	svc := NewBookingService(repo, WithClock(func() time.Time { return now }))

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
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Carlos-JPS/medconnect/availability-service/modules/repository"
)

type AvailabilityService interface {
	GetAvailableSlots(ctx context.Context, specialty string, startDate, endDate time.Time) ([]*repository.Slot, error)
	HoldSlot(ctx context.Context, slotID string, bookingID string, heldUntil time.Time) (*repository.Slot, error)
	ConfirmSlotBooking(ctx context.Context, slotID string, bookingID string) (*repository.Slot, error)
	ReleaseHeldSlot(ctx context.Context, slotID string, bookingID string) (*repository.Slot, error)
	GetDoctorAgenda(ctx context.Context, doctorID string, startDate, endDate time.Time) ([]*repository.Slot, error)
}

type availabilityService struct {
	repo repository.AvailabilityRepository
}

func NewAvailabilityService(repo repository.AvailabilityRepository) AvailabilityService {
	return &availabilityService{
		repo: repo,
	}
}

func (s *availabilityService) GetAvailableSlots(ctx context.Context, specialty string, startDate, endDate time.Time) ([]*repository.Slot, error) {
	if startDate.After(endDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}
	return s.repo.GetAvailableSlots(ctx, specialty, startDate, endDate)
}

func (s *availabilityService) HoldSlot(ctx context.Context, slotID string, bookingID string, heldUntil time.Time) (*repository.Slot, error) {
	return s.repo.HoldSlot(ctx, slotID, bookingID, heldUntil)
}

func (s *availabilityService) ConfirmSlotBooking(ctx context.Context, slotID string, bookingID string) (*repository.Slot, error) {
	return s.repo.ConfirmSlotBooking(ctx, slotID, bookingID)
}

func (s *availabilityService) ReleaseHeldSlot(ctx context.Context, slotID string, bookingID string) (*repository.Slot, error) {
	return s.repo.ReleaseHeldSlot(ctx, slotID, bookingID)
}

func (s *availabilityService) GetDoctorAgenda(ctx context.Context, doctorID string, startDate, endDate time.Time) ([]*repository.Slot, error) {
	if startDate.After(endDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}
	return s.repo.GetDoctorAgenda(ctx, doctorID, startDate, endDate)
}

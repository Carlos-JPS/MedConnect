package service

import (
	"fmt"
	"time"

	"github.com/Carlos-JPS/medconnect/availability-service/modules/repository"
)

type AvailabilityService interface {
	GetAvailableSlots(specialty string, startDate, endDate time.Time) ([]*repository.Slot, error)
	HoldSlot(slotID string) (*repository.Slot, error)
	ConfirmSlotBooking(slotID string) (*repository.Slot, error)
	ReleaseHeldSlot(slotID string) (*repository.Slot, error)
	GetDoctorAgenda(doctorID string, startDate, endDate time.Time) ([]*repository.Slot, error)
}

type availabilityService struct {
	repo repository.AvailabilityRepository
}

func NewAvailabilityService(repo repository.AvailabilityRepository) AvailabilityService {
	return &availabilityService{
		repo: repo,
	}
}

func (s *availabilityService) GetAvailableSlots(specialty string, startDate, endDate time.Time) ([]*repository.Slot, error) {
	if startDate.After(endDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}
	return s.repo.GetAvailableSlots(specialty, startDate, endDate)
}

func (s *availabilityService) HoldSlot(slotID string) (*repository.Slot, error) {
	return s.repo.HoldSlot(slotID)
}

func (s *availabilityService) ConfirmSlotBooking(slotID string) (*repository.Slot, error) {
	return s.repo.ConfirmSlotBooking(slotID)
}

func (s *availabilityService) ReleaseHeldSlot(slotID string) (*repository.Slot, error) {
	return s.repo.ReleaseHeldSlot(slotID)
}

func (s *availabilityService) GetDoctorAgenda(doctorID string, startDate, endDate time.Time) ([]*repository.Slot, error) {
	if startDate.After(endDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}
	return s.repo.GetDoctorAgenda(doctorID, startDate, endDate)
}

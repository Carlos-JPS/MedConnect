package handler

import (
	"context"
	"errors"
	"time"

	"github.com/Carlos-JPS/medconnect/availability-service/modules/repository"
	"github.com/Carlos-JPS/medconnect/availability-service/modules/service"
	pb "github.com/Carlos-JPS/medconnect/availability-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	pb.UnimplementedAvailabilityServiceServer
	svc service.AvailabilityService
}

func NewGRPCHandler(svc service.AvailabilityService) *GRPCHandler {
	return &GRPCHandler{
		svc: svc,
	}
}

func mapSlotToProto(s *repository.Slot) *pb.Slot {
	return &pb.Slot{
		SlotId:    s.ID,
		DoctorId:  s.DoctorID,
		Specialty: "", // We might not have this depending on the query, or fetch it
		StartTime: s.StartTime.Format(time.RFC3339),
		EndTime:   s.EndTime.Format(time.RFC3339),
		Status:    string(s.Status),
	}
}

func availabilityError(message string, err error) error {
	switch {
	case errors.Is(err, repository.ErrSlotNotAvailable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, repository.ErrSlotShardNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Errorf(codes.Unavailable, "%s: %v", message, err)
	}
}

func (h *GRPCHandler) GetAvailableSlots(ctx context.Context, req *pb.GetAvailableSlotsRequest) (*pb.GetAvailableSlotsResponse, error) {
	if req.Specialty == "" || req.FromDate == "" || req.ToDate == "" {
		return nil, status.Error(codes.InvalidArgument, "specialty, from_date and to_date are required")
	}

	startDate, err := time.Parse(time.RFC3339, req.FromDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid from_date format, use RFC3339")
	}

	endDate, err := time.Parse(time.RFC3339, req.ToDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid to_date format, use RFC3339")
	}
	if startDate.After(endDate) {
		return nil, status.Error(codes.InvalidArgument, "from_date cannot be after to_date")
	}

	slots, err := h.svc.GetAvailableSlots(ctx, req.Specialty, startDate, endDate)
	if err != nil {
		return nil, availabilityError("failed to get slots", err)
	}

	var pbSlots []*pb.Slot
	for _, s := range slots {
		pbSlots = append(pbSlots, mapSlotToProto(s))
	}

	return &pb.GetAvailableSlotsResponse{Slots: pbSlots}, nil
}

func (h *GRPCHandler) HoldSlot(ctx context.Context, req *pb.HoldSlotRequest) (*pb.HoldSlotResponse, error) {
	if req.SlotId == "" {
		return nil, status.Error(codes.InvalidArgument, "slot_id is required")
	}

	heldUntil := time.Now().Add(15 * time.Minute)
	if req.HeldUntil != "" {
		parsed, err := time.Parse(time.RFC3339, req.HeldUntil)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid held_until format, use RFC3339")
		}
		heldUntil = parsed
	}

	slot, err := h.svc.HoldSlot(ctx, req.SlotId, req.BookingId, heldUntil)
	if err != nil {
		return nil, availabilityError("failed to hold slot", err)
	}

	return &pb.HoldSlotResponse{
		SlotId:    slot.ID,
		Status:    string(slot.Status),
		HeldUntil: heldUntil.Format(time.RFC3339),
		BookingId: req.BookingId,
	}, nil
}

func (h *GRPCHandler) ConfirmSlotBooking(ctx context.Context, req *pb.ConfirmSlotBookingRequest) (*pb.ConfirmSlotBookingResponse, error) {
	if req.SlotId == "" {
		return nil, status.Error(codes.InvalidArgument, "slot_id is required")
	}

	slot, err := h.svc.ConfirmSlotBooking(ctx, req.SlotId, req.BookingId)
	if err != nil {
		return nil, availabilityError("failed to confirm slot", err)
	}

	return &pb.ConfirmSlotBookingResponse{
		SlotId:    slot.ID,
		Status:    string(slot.Status),
		BookingId: req.BookingId,
	}, nil
}

func (h *GRPCHandler) ReleaseHeldSlot(ctx context.Context, req *pb.ReleaseHeldSlotRequest) (*pb.ReleaseHeldSlotResponse, error) {
	if req.SlotId == "" {
		return nil, status.Error(codes.InvalidArgument, "slot_id is required")
	}

	slot, err := h.svc.ReleaseHeldSlot(ctx, req.SlotId, req.BookingId)
	if err != nil {
		return nil, availabilityError("failed to release slot", err)
	}

	return &pb.ReleaseHeldSlotResponse{
		SlotId: slot.ID,
		Status: string(slot.Status),
	}, nil
}

func (h *GRPCHandler) GetDoctorAgenda(ctx context.Context, req *pb.GetDoctorAgendaRequest) (*pb.GetDoctorAgendaResponse, error) {
	if req.DoctorId == "" || req.FromDate == "" || req.ToDate == "" {
		return nil, status.Error(codes.InvalidArgument, "doctor_id, from_date and to_date are required")
	}

	startDate, err := time.Parse(time.RFC3339, req.FromDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid from_date format, use RFC3339")
	}

	endDate, err := time.Parse(time.RFC3339, req.ToDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid to_date format, use RFC3339")
	}
	if startDate.After(endDate) {
		return nil, status.Error(codes.InvalidArgument, "from_date cannot be after to_date")
	}

	slots, err := h.svc.GetDoctorAgenda(ctx, req.DoctorId, startDate, endDate)
	if err != nil {
		return nil, availabilityError("failed to get agenda", err)
	}

	var pbSlots []*pb.Slot
	for _, s := range slots {
		pbSlots = append(pbSlots, mapSlotToProto(s))
	}

	return &pb.GetDoctorAgendaResponse{
		DoctorId: req.DoctorId,
		Slots:    pbSlots,
	}, nil
}

package handler

import (
	"context"
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

	slots, err := h.svc.GetAvailableSlots(req.Specialty, startDate, endDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get slots: %v", err)
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

	slot, err := h.svc.HoldSlot(req.SlotId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hold slot: %v", err)
	}

	return &pb.HoldSlotResponse{
		SlotId:     slot.ID,
		Status:     string(slot.Status),
		HeldUntil:  time.Now().Add(15 * time.Minute).Format(time.RFC3339), // Demo logic
		BookingId:  req.BookingId,
	}, nil
}

func (h *GRPCHandler) ConfirmSlotBooking(ctx context.Context, req *pb.ConfirmSlotBookingRequest) (*pb.ConfirmSlotBookingResponse, error) {
	if req.SlotId == "" {
		return nil, status.Error(codes.InvalidArgument, "slot_id is required")
	}

	slot, err := h.svc.ConfirmSlotBooking(req.SlotId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to confirm slot: %v", err)
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

	slot, err := h.svc.ReleaseHeldSlot(req.SlotId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to release slot: %v", err)
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

	slots, err := h.svc.GetDoctorAgenda(req.DoctorId, startDate, endDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get agenda: %v", err)
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

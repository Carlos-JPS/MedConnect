package grpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MedConnect/booking-service/internal/service"
	pb "github.com/MedConnect/booking-service/pb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedBookingServiceServer
	bookingService service.BookingService
}

func NewServer(bookingService service.BookingService) *Server {
	return &Server{bookingService: bookingService}
}

func (s *Server) CreateBooking(ctx context.Context, req *pb.CreateBookingRequest) (*pb.CreateBookingResponse, error) {
	booking, err := s.bookingService.CreateBooking(ctx, service.CreateBookingInput{
		PatientID: req.GetPatientId(),
		DoctorID:  req.GetDoctorId(),
		SlotID:    req.GetSlotId(),
		Notes:     req.GetNotes(),
	})
	if err != nil {
		return nil, err
	}

	return &pb.CreateBookingResponse{
		BookingId:     booking.BookingID,
		Status:        mapStatus(booking.Status),
		ReservedUntil: timestamppb.New(booking.ReservedUntil),
	}, nil
}

func (s *Server) CancelBooking(ctx context.Context, req *pb.CancelBookingRequest) (*pb.CancelBookingResponse, error) {
	booking, err := s.bookingService.CancelBooking(ctx, service.CancelBookingInput{
		BookingID: req.GetBookingId(),
		Reason:    req.GetReason(),
	})
	if err != nil {
		return nil, err
	}

	return &pb.CancelBookingResponse{
		BookingId: booking.BookingID,
		Status:    mapStatus(booking.Status),
		UpdatedAt: timestamppb.New(booking.UpdatedAt),
	}, nil
}

func (s *Server) GetBooking(ctx context.Context, req *pb.GetBookingRequest) (*pb.GetBookingResponse, error) {
	details, err := s.bookingService.GetBooking(ctx, req.GetBookingId())
	if err != nil {
		return nil, err
	}

	return &pb.GetBookingResponse{
		Booking: mapBooking(details.Booking),
		Events:  mapEvents(details.Events),
	}, nil
}

func (s *Server) ConfirmBooking(ctx context.Context, req *pb.ConfirmBookingRequest) (*pb.ConfirmBookingResponse, error) {
	booking, err := s.bookingService.ConfirmBooking(ctx, service.ConfirmBookingInput{
		BookingID: req.GetBookingId(),
		PaymentID: req.GetPaymentId(),
	})
	if err != nil {
		return nil, err
	}

	return &pb.ConfirmBookingResponse{
		BookingId: booking.BookingID,
		Status:    mapStatus(booking.Status),
		UpdatedAt: timestamppb.New(booking.UpdatedAt),
	}, nil
}

func (s *Server) ListBookingsByPatient(ctx context.Context, req *pb.ListBookingsByPatientRequest) (*pb.ListBookingsByPatientResponse, error) {
	bookings, err := s.bookingService.ListBookingsByPatient(ctx, req.GetPatientId(), mapPBStatus(req.GetStatus()))
	if err != nil {
		return nil, err
	}

	return &pb.ListBookingsByPatientResponse{
		Bookings: mapBookings(bookings),
	}, nil
}

func mapBookings(bookings []service.Booking) []*pb.Booking {
	result := make([]*pb.Booking, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, mapBooking(booking))
	}
	return result
}

func mapBooking(booking service.Booking) *pb.Booking {
	return &pb.Booking{
		BookingId:        booking.BookingID,
		PatientId:        booking.PatientID,
		DoctorId:         booking.DoctorID,
		SlotId:           booking.SlotID,
		Status:           mapStatus(booking.Status),
		PaymentId:        booking.PaymentID,
		ConfirmationCode: booking.ConfirmationCode,
		Notes:            booking.Notes,
		CreatedAt:        timestamppb.New(booking.CreatedAt),
		UpdatedAt:        timestamppb.New(booking.UpdatedAt),
		ReservedUntil:    timestamppb.New(booking.ReservedUntil),
		ConfirmedAt:      timestampPtr(booking.ConfirmedAt),
		CancelledAt:      timestampPtr(booking.CancelledAt),
	}
}

func mapEvents(events []service.BookingEvent) []*pb.BookingEvent {
	result := make([]*pb.BookingEvent, 0, len(events))
	for _, event := range events {
		result = append(result, mapEvent(event))
	}
	return result
}

func mapEvent(event service.BookingEvent) *pb.BookingEvent {
	return &pb.BookingEvent{
		EventId:   event.EventID,
		BookingId: event.BookingID,
		EventType: mapEventType(event.EventType),
		Payload:   mapPayload(event.Payload),
		CreatedAt: timestamppb.New(event.CreatedAt),
	}
}

func mapStatus(status service.Status) pb.BookingStatus {
	switch status {
	case service.StatusPendingPayment:
		return pb.BookingStatus_BOOKING_STATUS_PENDING_PAYMENT
	case service.StatusConfirmed:
		return pb.BookingStatus_BOOKING_STATUS_CONFIRMED
	case service.StatusCancelled:
		return pb.BookingStatus_BOOKING_STATUS_CANCELLED
	case service.StatusExpired:
		return pb.BookingStatus_BOOKING_STATUS_EXPIRED
	default:
		return pb.BookingStatus_BOOKING_STATUS_UNSPECIFIED
	}
}

func mapPBStatus(status pb.BookingStatus) service.Status {
	switch status {
	case pb.BookingStatus_BOOKING_STATUS_PENDING_PAYMENT:
		return service.StatusPendingPayment
	case pb.BookingStatus_BOOKING_STATUS_CONFIRMED:
		return service.StatusConfirmed
	case pb.BookingStatus_BOOKING_STATUS_CANCELLED:
		return service.StatusCancelled
	case pb.BookingStatus_BOOKING_STATUS_EXPIRED:
		return service.StatusExpired
	default:
		return service.StatusUnspecified
	}
}

func mapEventType(eventType service.EventType) pb.BookingEventType {
	switch eventType {
	case service.EventCreated:
		return pb.BookingEventType_BOOKING_EVENT_TYPE_CREATED
	case service.EventPaymentApproved:
		return pb.BookingEventType_BOOKING_EVENT_TYPE_PAYMENT_APPROVED
	case service.EventConfirmed:
		return pb.BookingEventType_BOOKING_EVENT_TYPE_CONFIRMED
	case service.EventCancelled:
		return pb.BookingEventType_BOOKING_EVENT_TYPE_CANCELLED
	case service.EventExpired:
		return pb.BookingEventType_BOOKING_EVENT_TYPE_EXPIRED
	default:
		return pb.BookingEventType_BOOKING_EVENT_TYPE_UNSPECIFIED
	}
}

func mapPayload(payload []byte) *structpb.Struct {
	if len(payload) == 0 {
		return nil
	}

	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil
	}
	structValue, err := structpb.NewStruct(values)
	if err != nil {
		return nil
	}
	return structValue
}

func timestampPtr(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

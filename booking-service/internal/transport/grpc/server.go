package grpc

import (
	"context"
	"time"

	"github.com/MedConnect/booking-service/internal/service"
	pb "github.com/MedConnect/booking-service/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedBookingServiceServer
	bookingService service.BookingService
}

func NewServer(bookingService service.BookingService) *Server {
	return &Server{bookingService: bookingService}
}

func (s *Server) GetBooking(ctx context.Context, req *pb.GetBookingRequest) (*pb.GetBookingResponse, error) {
	booking, err := s.bookingService.GetBooking(ctx, req.GetBookingId())
	if err != nil {
		return nil, err
	}

	return &pb.GetBookingResponse{
		Booking: mapBooking(booking),
	}, nil
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

func timestampPtr(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

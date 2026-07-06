package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/MedConnect/booking-service/internal/service"
	pb "github.com/MedConnect/booking-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedBookingServiceServer
	bookingService service.BookingService
	sagaService    service.BookingSagaService
}

func NewServer(bookingService service.BookingService, sagaService service.BookingSagaService) *Server {
	return &Server{bookingService: bookingService, sagaService: sagaService}
}

func (s *Server) CreateBooking(ctx context.Context, req *pb.CreateBookingRequest) (*pb.CreateBookingResponse, error) {
	booking, err := s.bookingService.CreateBooking(ctx, service.CreateBookingInput{
		PatientID: req.GetPatientId(),
		DoctorID:  req.GetDoctorId(),
		SlotID:    req.GetSlotId(),
		Notes:     req.GetNotes(),
	})
	if err != nil {
		return nil, mapServiceError(err)
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
		return nil, mapServiceError(err)
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
		return nil, mapServiceError(err)
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
		return nil, mapServiceError(err)
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
		return nil, mapServiceError(err)
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

func (s *Server) StartBookingSaga(ctx context.Context, req *pb.StartBookingSagaRequest) (*pb.StartBookingSagaResponse, error) {
	result, err := s.sagaService.StartBookingSaga(ctx, service.StartBookingSagaInput{
		PatientID:       req.GetPatientId(),
		DoctorID:        req.GetDoctorId(),
		SlotID:          req.GetSlotId(),
		Notes:           req.GetNotes(),
		Amount:          req.GetAmount(),
		Currency:        req.GetCurrency(),
		PaymentMethodID: req.GetPaymentMethodId(),
	})
	if err != nil {
		return nil, mapServiceError(err)
	}

	return &pb.StartBookingSagaResponse{
		Saga:          mapSaga(result.Saga),
		Booking:       mapBooking(result.Booking),
		PaymentId:     result.Payment.PaymentID,
		PaymentStatus: string(result.Payment.Status),
	}, nil
}

func (s *Server) GetBookingSaga(ctx context.Context, req *pb.GetBookingSagaRequest) (*pb.GetBookingSagaResponse, error) {
	details, err := s.sagaService.GetBookingSaga(ctx, req.GetSagaId())
	if err != nil {
		return nil, mapServiceError(err)
	}

	return &pb.GetBookingSagaResponse{
		Saga:   mapSaga(details.Saga),
		Events: mapSagaEvents(details.Events),
	}, nil
}

func mapEvents(events []service.BookingEvent) []*pb.BookingEvent {
	result := make([]*pb.BookingEvent, 0, len(events))
	for _, event := range events {
		result = append(result, mapEvent(event))
	}
	return result
}

func mapSaga(saga service.BookingSaga) *pb.BookingSaga {
	return &pb.BookingSaga{
		SagaId:             saga.SagaID,
		BookingId:          saga.BookingID,
		PaymentId:          saga.PaymentID,
		PatientId:          saga.PatientID,
		DoctorId:           saga.DoctorID,
		SlotId:             saga.SlotID,
		Status:             mapSagaStatus(saga.Status),
		CurrentStep:        string(saga.CurrentStep),
		CompensationStatus: mapSagaCompensationStatus(saga.CompensationStatus),
		RetryCount:         int32(saga.RetryCount),
		LastError:          saga.LastError,
		CreatedAt:          timestamppb.New(saga.CreatedAt),
		UpdatedAt:          timestamppb.New(saga.UpdatedAt),
		CompletedAt:        timestampPtr(saga.CompletedAt),
	}
}

func mapSagaEvents(events []service.BookingSagaEvent) []*pb.BookingSagaEvent {
	result := make([]*pb.BookingSagaEvent, 0, len(events))
	for _, event := range events {
		result = append(result, mapSagaEvent(event))
	}
	return result
}

func mapSagaEvent(event service.BookingSagaEvent) *pb.BookingSagaEvent {
	return &pb.BookingSagaEvent{
		EventId:      event.EventID,
		SagaId:       event.SagaID,
		EventType:    event.EventType,
		Step:         string(event.Step),
		Status:       mapSagaStatus(event.Status),
		Payload:      mapPayload(event.Payload),
		ErrorMessage: event.ErrorMessage,
		CreatedAt:    timestamppb.New(event.CreatedAt),
	}
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

func mapSagaStatus(status service.SagaStatus) pb.BookingSagaStatus {
	switch status {
	case service.SagaStatusStarted:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_STARTED
	case service.SagaStatusSlotHeld:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_SLOT_HELD
	case service.SagaStatusBookingCreated:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_BOOKING_CREATED
	case service.SagaStatusPaymentCreated:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_PAYMENT_CREATED
	case service.SagaStatusPaymentCompleted:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_PAYMENT_COMPLETED
	case service.SagaStatusSlotConfirmed:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_SLOT_CONFIRMED
	case service.SagaStatusCompleted:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPLETED
	case service.SagaStatusCompensating:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPENSATING
	case service.SagaStatusCompensated:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPENSATED
	case service.SagaStatusFailed:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_FAILED
	case service.SagaStatusCompensationFailed:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPENSATION_FAILED
	default:
		return pb.BookingSagaStatus_BOOKING_SAGA_STATUS_UNSPECIFIED
	}
}

func mapSagaCompensationStatus(status service.SagaCompensationStatus) pb.BookingSagaCompensationStatus {
	switch status {
	case service.SagaCompensationNotRequired:
		return pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_NOT_REQUIRED
	case service.SagaCompensationPending:
		return pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_PENDING
	case service.SagaCompensationInProgress:
		return pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_IN_PROGRESS
	case service.SagaCompensationCompleted:
		return pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_COMPLETED
	case service.SagaCompensationFailed:
		return pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_FAILED
	default:
		return pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_UNSPECIFIED
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

func mapServiceError(err error) error {
	switch {
	case errors.Is(err, service.ErrActiveSlotBookingExists):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, service.ErrInvalidBookingState):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, service.ErrExternalDependency):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return err
	}
}

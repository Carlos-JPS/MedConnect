package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	pb "github.com/MedConnect/booking-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BookingClient interface {
	CreateBooking(ctx context.Context, req *pb.CreateBookingRequest) (*pb.CreateBookingResponse, error)
	CancelBooking(ctx context.Context, req *pb.CancelBookingRequest) (*pb.CancelBookingResponse, error)
	GetBooking(ctx context.Context, req *pb.GetBookingRequest) (*pb.GetBookingResponse, error)
	ListBookingsByPatient(ctx context.Context, req *pb.ListBookingsByPatientRequest) (*pb.ListBookingsByPatientResponse, error)
	ConfirmBooking(ctx context.Context, req *pb.ConfirmBookingRequest) (*pb.ConfirmBookingResponse, error)
}

type Handler struct {
	booking BookingClient
}

func NewHandler(booking BookingClient) *Handler {
	return &Handler{booking: booking}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/bookings":
		h.createBooking(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/bookings":
		h.listBookings(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/bookings/"):
		h.getBooking(w, r)
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/bookings/") && strings.HasSuffix(r.URL.Path, "/cancel"):
		h.cancelBooking(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/bookings/") && strings.HasSuffix(r.URL.Path, "/confirm"):
		h.confirmBooking(w, r)
	default:
		writeError(w, http.StatusNotFound, "ruta no encontrada")
	}
}

type createBookingRequest struct {
	PatientID string `json:"patient_id"`
	DoctorID  string `json:"doctor_id"`
	SlotID    string `json:"slot_id"`
	Notes     string `json:"notes"`
}

func (h *Handler) createBooking(w http.ResponseWriter, r *http.Request) {
	var req createBookingRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.PatientID == "" || req.DoctorID == "" || req.SlotID == "" {
		writeError(w, http.StatusBadRequest, "patient_id, doctor_id y slot_id son obligatorios")
		return
	}

	resp, err := h.booking.CreateBooking(r.Context(), &pb.CreateBookingRequest{
		PatientId: req.PatientID,
		DoctorId:  req.DoctorID,
		SlotId:    req.SlotID,
		Notes:     req.Notes,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"booking_id":     resp.GetBookingId(),
		"status":         statusToJSON(resp.GetStatus()),
		"reserved_until": timestampToJSON(resp.GetReservedUntil()),
	})
}

func (h *Handler) listBookings(w http.ResponseWriter, r *http.Request) {
	patientID := r.URL.Query().Get("patient_id")
	if patientID == "" {
		writeError(w, http.StatusBadRequest, "patient_id es obligatorio")
		return
	}

	resp, err := h.booking.ListBookingsByPatient(r.Context(), &pb.ListBookingsByPatientRequest{
		PatientId: patientID,
		Status:    statusFromQuery(r.URL.Query().Get("status")),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	bookings := make([]bookingResponse, 0, len(resp.GetBookings()))
	for _, booking := range resp.GetBookings() {
		bookings = append(bookings, mapBooking(booking))
	}
	writeJSON(w, http.StatusOK, map[string]any{"bookings": bookings})
}

func (h *Handler) getBooking(w http.ResponseWriter, r *http.Request) {
	bookingID := bookingIDFromPath(r.URL.Path, "")
	if bookingID == "" {
		writeError(w, http.StatusNotFound, "booking no encontrado")
		return
	}

	resp, err := h.booking.GetBooking(r.Context(), &pb.GetBookingRequest{BookingId: bookingID})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	events := make([]bookingEventResponse, 0, len(resp.GetEvents()))
	for _, event := range resp.GetEvents() {
		events = append(events, mapEvent(event))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"booking": mapBooking(resp.GetBooking()),
		"events":  events,
	})
}

type cancelBookingRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) cancelBooking(w http.ResponseWriter, r *http.Request) {
	bookingID := bookingIDFromPath(r.URL.Path, "cancel")
	if bookingID == "" {
		writeError(w, http.StatusNotFound, "booking no encontrado")
		return
	}

	var req cancelBookingRequest
	if err := decodeJSONAllowEmpty(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}

	resp, err := h.booking.CancelBooking(r.Context(), &pb.CancelBookingRequest{
		BookingId: bookingID,
		Reason:    req.Reason,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"booking_id": resp.GetBookingId(),
		"status":     statusToJSON(resp.GetStatus()),
		"updated_at": timestampToJSON(resp.GetUpdatedAt()),
	})
}

type confirmBookingRequest struct {
	PaymentID string `json:"payment_id"`
}

func (h *Handler) confirmBooking(w http.ResponseWriter, r *http.Request) {
	bookingID := bookingIDFromPath(r.URL.Path, "confirm")
	if bookingID == "" {
		writeError(w, http.StatusNotFound, "booking no encontrado")
		return
	}

	var req confirmBookingRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.PaymentID == "" {
		writeError(w, http.StatusBadRequest, "payment_id es obligatorio")
		return
	}

	resp, err := h.booking.ConfirmBooking(r.Context(), &pb.ConfirmBookingRequest{
		BookingId: bookingID,
		PaymentId: req.PaymentID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"booking_id": resp.GetBookingId(),
		"status":     statusToJSON(resp.GetStatus()),
		"updated_at": timestampToJSON(resp.GetUpdatedAt()),
	})
}

func bookingIDFromPath(path string, suffix string) string {
	trimmed := strings.TrimPrefix(path, "/bookings/")
	if trimmed == path || trimmed == "" {
		return ""
	}
	if suffix != "" {
		trimmed = strings.TrimSuffix(trimmed, "/"+suffix)
	}
	if strings.Contains(trimmed, "/") {
		return ""
	}
	return trimmed
}

type bookingResponse struct {
	BookingID        string `json:"booking_id"`
	PatientID        string `json:"patient_id,omitempty"`
	DoctorID         string `json:"doctor_id,omitempty"`
	SlotID           string `json:"slot_id,omitempty"`
	Status           string `json:"status"`
	PaymentID        string `json:"payment_id,omitempty"`
	ConfirmationCode string `json:"confirmation_code,omitempty"`
	Notes            string `json:"notes,omitempty"`
	CreatedAt        string `json:"created_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
	ReservedUntil    string `json:"reserved_until,omitempty"`
	ConfirmedAt      string `json:"confirmed_at,omitempty"`
	CancelledAt      string `json:"cancelled_at,omitempty"`
}

func mapBooking(booking *pb.Booking) bookingResponse {
	if booking == nil {
		return bookingResponse{}
	}
	return bookingResponse{
		BookingID:        booking.GetBookingId(),
		PatientID:        booking.GetPatientId(),
		DoctorID:         booking.GetDoctorId(),
		SlotID:           booking.GetSlotId(),
		Status:           statusToJSON(booking.GetStatus()),
		PaymentID:        booking.GetPaymentId(),
		ConfirmationCode: booking.GetConfirmationCode(),
		Notes:            booking.GetNotes(),
		CreatedAt:        timestampToJSON(booking.GetCreatedAt()),
		UpdatedAt:        timestampToJSON(booking.GetUpdatedAt()),
		ReservedUntil:    timestampToJSON(booking.GetReservedUntil()),
		ConfirmedAt:      timestampToJSON(booking.GetConfirmedAt()),
		CancelledAt:      timestampToJSON(booking.GetCancelledAt()),
	}
}

type bookingEventResponse struct {
	EventID   string         `json:"event_id"`
	BookingID string         `json:"booking_id"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt string         `json:"created_at,omitempty"`
}

func mapEvent(event *pb.BookingEvent) bookingEventResponse {
	if event == nil {
		return bookingEventResponse{}
	}
	payload := map[string]any(nil)
	if event.GetPayload() != nil {
		payload = event.GetPayload().AsMap()
	}
	return bookingEventResponse{
		EventID:   event.GetEventId(),
		BookingID: event.GetBookingId(),
		EventType: eventTypeToJSON(event.GetEventType()),
		Payload:   payload,
		CreatedAt: timestampToJSON(event.GetCreatedAt()),
	}
}

func statusFromQuery(value string) pb.BookingStatus {
	switch strings.ToUpper(value) {
	case "PENDING_PAYMENT":
		return pb.BookingStatus_BOOKING_STATUS_PENDING_PAYMENT
	case "CONFIRMED":
		return pb.BookingStatus_BOOKING_STATUS_CONFIRMED
	case "CANCELLED":
		return pb.BookingStatus_BOOKING_STATUS_CANCELLED
	case "EXPIRED":
		return pb.BookingStatus_BOOKING_STATUS_EXPIRED
	default:
		return pb.BookingStatus_BOOKING_STATUS_UNSPECIFIED
	}
}

func statusToJSON(value pb.BookingStatus) string {
	switch value {
	case pb.BookingStatus_BOOKING_STATUS_PENDING_PAYMENT:
		return "PENDING_PAYMENT"
	case pb.BookingStatus_BOOKING_STATUS_CONFIRMED:
		return "CONFIRMED"
	case pb.BookingStatus_BOOKING_STATUS_CANCELLED:
		return "CANCELLED"
	case pb.BookingStatus_BOOKING_STATUS_EXPIRED:
		return "EXPIRED"
	default:
		return "UNSPECIFIED"
	}
}

func eventTypeToJSON(value pb.BookingEventType) string {
	switch value {
	case pb.BookingEventType_BOOKING_EVENT_TYPE_CREATED:
		return "CREATED"
	case pb.BookingEventType_BOOKING_EVENT_TYPE_PAYMENT_APPROVED:
		return "PAYMENT_APPROVED"
	case pb.BookingEventType_BOOKING_EVENT_TYPE_CONFIRMED:
		return "CONFIRMED"
	case pb.BookingEventType_BOOKING_EVENT_TYPE_CANCELLED:
		return "CANCELLED"
	case pb.BookingEventType_BOOKING_EVENT_TYPE_EXPIRED:
		return "EXPIRED"
	default:
		return "UNSPECIFIED"
	}
}

func timestampToJSON(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return value.AsTime().Format(time.RFC3339)
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func decodeJSONAllowEmpty(r *http.Request, target any) error {
	defer r.Body.Close()
	if r.Body == nil {
		return nil
	}
	err := json.NewDecoder(r.Body).Decode(target)
	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func writeGRPCError(w http.ResponseWriter, err error) {
	code := status.Code(err)
	switch code {
	case codes.NotFound:
		writeError(w, http.StatusNotFound, err.Error())
	case codes.InvalidArgument:
		writeError(w, http.StatusBadRequest, err.Error())
	case codes.DeadlineExceeded, codes.Unavailable, codes.Unknown:
		writeError(w, http.StatusBadGateway, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

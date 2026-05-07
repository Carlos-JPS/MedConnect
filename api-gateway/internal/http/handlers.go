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
	paymentpb "github.com/sllanoscaro/payment-service/pb"
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

type PaymentClient interface {
	CreatePayment(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error)
	ProcessPayment(ctx context.Context, req *paymentpb.ProcessPaymentRequest) (*paymentpb.ProcessPaymentResponse, error)
	GetPayment(ctx context.Context, req *paymentpb.GetPaymentRequest) (*paymentpb.GetPaymentResponse, error)
	GetPaymentsByUser(ctx context.Context, req *paymentpb.GetPaymentsByUserRequest) (*paymentpb.GetPaymentsByUserResponse, error)
	GetPaymentByBooking(ctx context.Context, req *paymentpb.GetPaymentByBookingRequest) (*paymentpb.GetPaymentByBookingResponse, error)
	RefundPayment(ctx context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error)
}

type Handler struct {
	booking BookingClient
	payment PaymentClient
}

func NewHandler(booking BookingClient, payment ...PaymentClient) *Handler {
	handler := &Handler{booking: booking}
	if len(payment) > 0 {
		handler.payment = payment[0]
	}
	return handler
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
	case r.Method == http.MethodPost && r.URL.Path == "/payments":
		h.createPayment(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/payments/") && strings.HasSuffix(r.URL.Path, "/process"):
		h.processPayment(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/payments/") && strings.HasSuffix(r.URL.Path, "/refund"):
		h.refundPayment(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/payments/user/"):
		h.getPaymentsByUser(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/payments/booking/"):
		h.getPaymentByBooking(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/payments/"):
		h.getPayment(w, r)
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

type createPaymentRequest struct {
	BookingID string  `json:"booking_id"`
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request) {
	if !h.requirePaymentClient(w) {
		return
	}

	var req createPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.BookingID == "" || req.UserID == "" || req.Amount <= 0 || req.Currency == "" {
		writeError(w, http.StatusBadRequest, "booking_id, user_id, amount y currency son obligatorios")
		return
	}

	resp, err := h.payment.CreatePayment(r.Context(), &paymentpb.CreatePaymentRequest{
		BookingId: req.BookingID,
		UserId:    req.UserID,
		Amount:    req.Amount,
		Currency:  req.Currency,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapPayment(resp.GetPayment()))
}

type processPaymentRequest struct {
	PaymentMethodID string `json:"payment_method_id"`
}

func (h *Handler) processPayment(w http.ResponseWriter, r *http.Request) {
	if !h.requirePaymentClient(w) {
		return
	}

	paymentID := paymentIDFromPath(r.URL.Path, "process")
	if paymentID == "" {
		writeError(w, http.StatusNotFound, "pago no encontrado")
		return
	}

	var req processPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.PaymentMethodID == "" {
		writeError(w, http.StatusBadRequest, "payment_method_id es obligatorio")
		return
	}

	resp, err := h.payment.ProcessPayment(r.Context(), &paymentpb.ProcessPaymentRequest{
		PaymentId:       paymentID,
		PaymentMethodId: req.PaymentMethodID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"payment_id":     resp.GetPaymentId(),
		"transaction_id": resp.GetTransactionId(),
		"status":         resp.GetStatus(),
	})
}

func (h *Handler) getPayment(w http.ResponseWriter, r *http.Request) {
	if !h.requirePaymentClient(w) {
		return
	}

	paymentID := paymentIDFromPath(r.URL.Path, "")
	if paymentID == "" {
		writeError(w, http.StatusNotFound, "pago no encontrado")
		return
	}

	resp, err := h.payment.GetPayment(r.Context(), &paymentpb.GetPaymentRequest{PaymentId: paymentID})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPayment(resp.GetPayment()))
}

func (h *Handler) getPaymentsByUser(w http.ResponseWriter, r *http.Request) {
	if !h.requirePaymentClient(w) {
		return
	}

	userID := valueAfterPrefix(r.URL.Path, "/payments/user/")
	if userID == "" {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	resp, err := h.payment.GetPaymentsByUser(r.Context(), &paymentpb.GetPaymentsByUserRequest{UserId: userID})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	payments := make([]paymentResponse, 0, len(resp.GetPayments()))
	for _, payment := range resp.GetPayments() {
		payments = append(payments, mapPayment(payment))
	}
	writeJSON(w, http.StatusOK, map[string]any{"payments": payments})
}

func (h *Handler) getPaymentByBooking(w http.ResponseWriter, r *http.Request) {
	if !h.requirePaymentClient(w) {
		return
	}

	bookingID := valueAfterPrefix(r.URL.Path, "/payments/booking/")
	if bookingID == "" {
		writeError(w, http.StatusNotFound, "booking no encontrado")
		return
	}

	resp, err := h.payment.GetPaymentByBooking(r.Context(), &paymentpb.GetPaymentByBookingRequest{BookingId: bookingID})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPayment(resp.GetPayment()))
}

type refundPaymentRequest struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
}

func (h *Handler) refundPayment(w http.ResponseWriter, r *http.Request) {
	if !h.requirePaymentClient(w) {
		return
	}

	paymentID := paymentIDFromPath(r.URL.Path, "refund")
	if paymentID == "" {
		writeError(w, http.StatusNotFound, "pago no encontrado")
		return
	}

	var req refundPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.Amount <= 0 || req.Reason == "" {
		writeError(w, http.StatusBadRequest, "amount y reason son obligatorios")
		return
	}

	resp, err := h.payment.RefundPayment(r.Context(), &paymentpb.RefundPaymentRequest{
		PaymentId: paymentID,
		Amount:    req.Amount,
		Reason:    req.Reason,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapRefund(resp.GetRefund()))
}

func (h *Handler) requirePaymentClient(w http.ResponseWriter) bool {
	if h.payment == nil {
		writeError(w, http.StatusServiceUnavailable, "payment-service no configurado")
		return false
	}
	return true
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

func paymentIDFromPath(path string, suffix string) string {
	trimmed := strings.TrimPrefix(path, "/payments/")
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

func valueAfterPrefix(path string, prefix string) string {
	value := strings.TrimPrefix(path, prefix)
	if value == path || value == "" || strings.Contains(value, "/") {
		return ""
	}
	return value
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

type paymentResponse struct {
	PaymentID string  `json:"payment_id"`
	BookingID string  `json:"booking_id,omitempty"`
	UserID    string  `json:"user_id,omitempty"`
	Amount    float64 `json:"amount,omitempty"`
	Currency  string  `json:"currency,omitempty"`
	Status    string  `json:"status,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
}

type refundResponse struct {
	RefundID  string  `json:"refund_id"`
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount,omitempty"`
	Reason    string  `json:"reason,omitempty"`
	Status    string  `json:"status,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
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

func mapPayment(payment *paymentpb.Payment) paymentResponse {
	if payment == nil {
		return paymentResponse{}
	}
	return paymentResponse{
		PaymentID: payment.GetPaymentId(),
		BookingID: payment.GetBookingId(),
		UserID:    payment.GetUserId(),
		Amount:    payment.GetAmount(),
		Currency:  payment.GetCurrency(),
		Status:    payment.GetStatus(),
		CreatedAt: payment.GetCreatedAt(),
	}
}

func mapRefund(refund *paymentpb.Refund) refundResponse {
	if refund == nil {
		return refundResponse{}
	}
	return refundResponse{
		RefundID:  refund.GetRefundId(),
		PaymentID: refund.GetPaymentId(),
		Amount:    refund.GetAmount(),
		Reason:    refund.GetReason(),
		Status:    refund.GetStatus(),
		CreatedAt: refund.GetCreatedAt(),
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

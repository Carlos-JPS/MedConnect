package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	availabilitypb "github.com/Carlos-JPS/medconnect/availability-service/pb"
	notificationclient "github.com/MedConnect/api-gateway/internal/clients/notification"
	authpb "github.com/MedConnect/auth-service/pb"
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
	StartBookingSaga(ctx context.Context, req *pb.StartBookingSagaRequest) (*pb.StartBookingSagaResponse, error)
	GetBookingSaga(ctx context.Context, req *pb.GetBookingSagaRequest) (*pb.GetBookingSagaResponse, error)
}

type PaymentClient interface {
	CreatePayment(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error)
	ProcessPayment(ctx context.Context, req *paymentpb.ProcessPaymentRequest) (*paymentpb.ProcessPaymentResponse, error)
	GetPayment(ctx context.Context, req *paymentpb.GetPaymentRequest) (*paymentpb.GetPaymentResponse, error)
	GetPaymentsByUser(ctx context.Context, req *paymentpb.GetPaymentsByUserRequest) (*paymentpb.GetPaymentsByUserResponse, error)
	GetPaymentByBooking(ctx context.Context, req *paymentpb.GetPaymentByBookingRequest) (*paymentpb.GetPaymentByBookingResponse, error)
	RefundPayment(ctx context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error)
}

type AvailabilityClient interface {
	GetAvailableSlots(ctx context.Context, req *availabilitypb.GetAvailableSlotsRequest) (*availabilitypb.GetAvailableSlotsResponse, error)
	GetDoctorAgenda(ctx context.Context, req *availabilitypb.GetDoctorAgendaRequest) (*availabilitypb.GetDoctorAgendaResponse, error)
	HoldSlot(ctx context.Context, req *availabilitypb.HoldSlotRequest) (*availabilitypb.HoldSlotResponse, error)
	ConfirmSlotBooking(ctx context.Context, req *availabilitypb.ConfirmSlotBookingRequest) (*availabilitypb.ConfirmSlotBookingResponse, error)
	ReleaseHeldSlot(ctx context.Context, req *availabilitypb.ReleaseHeldSlotRequest) (*availabilitypb.ReleaseHeldSlotResponse, error)
}

type AuthClient interface {
	RegisterUser(ctx context.Context, req *authpb.RegisterUserRequest) (*authpb.RegisterUserResponse, error)
	Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error)
	ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error)
	GetUserById(ctx context.Context, req *authpb.GetUserByIdRequest) (*authpb.GetUserByIdResponse, error)
}

type NotificationClient interface {
	ListNotifications(ctx context.Context, recipientID string, limit int) (*notificationclient.ListResponse, error)
	GetUnreadCount(ctx context.Context, recipientID string) (*notificationclient.UnreadCountResponse, error)
	MarkNotificationRead(ctx context.Context, notificationID int64, recipientID string) (*notificationclient.MarkReadResponse, error)
	GetDevStatus(ctx context.Context, recipientID string, limit int) (*notificationclient.DevStatusResponse, error)
}

type Handler struct {
	booking      BookingClient
	payment      PaymentClient
	availability AvailabilityClient
	auth         AuthClient
	notification NotificationClient
}

func NewHandler(booking BookingClient, payment PaymentClient, availability AvailabilityClient, auth AuthClient, notification NotificationClient) *Handler {
	return &Handler{booking: booking, payment: payment, availability: availability, auth: auth, notification: notification}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/auth/register":
		h.registerUser(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/auth/login":
		h.loginUser(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/auth/validate":
		h.validateToken(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/auth/users/"):
		h.getUser(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/notifications":
		h.withAuth(h.listNotifications)(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/notifications/unread-count":
		h.withAuth(h.getUnreadNotifications)(w, r)
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/notifications/") && strings.HasSuffix(r.URL.Path, "/read"):
		h.withAuth(h.markNotificationRead)(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/notifications/dev/status":
		h.withAuth(h.getNotificationDevStatus)(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/bookings":
		h.withAuth(h.createBooking)(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/booking-sagas":
		h.withAuth(h.startBookingSaga)(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/booking-sagas/"):
		h.withAuth(h.getBookingSaga)(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/bookings":
		h.withAuth(h.listBookings)(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/bookings/"):
		h.withAuth(h.getBooking)(w, r)
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/bookings/") && strings.HasSuffix(r.URL.Path, "/cancel"):
		h.withAuth(h.cancelBooking)(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/bookings/") && strings.HasSuffix(r.URL.Path, "/confirm"):
		h.withAuth(h.confirmBooking)(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/payments":
		h.withAuth(h.createPayment)(w, r)
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
	case r.Method == http.MethodGet && r.URL.Path == "/availability/slots":
		h.getAvailableSlots(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/availability/doctors/"):
		h.getDoctorAgenda(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/availability/hold":
		h.holdSlot(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/availability/confirm":
		h.confirmSlotBooking(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/availability/release":
		h.releaseHeldSlot(w, r)
	default:
		writeError(w, http.StatusNotFound, "ruta no encontrada")
	}
}

type startBookingSagaRequest struct {
	DoctorID        string  `json:"doctor_id"`
	SlotID          string  `json:"slot_id"`
	Notes           string  `json:"notes"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	PaymentMethodID string  `json:"payment_method_id"`
}

func (h *Handler) listNotifications(w http.ResponseWriter, r *http.Request) {
	if !h.requireNotificationClient(w) {
		return
	}
	recipientID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	resp, err := h.notification.ListNotifications(r.Context(), recipientID, limitFromQuery(r, 10))
	if err != nil {
		writeNotificationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) getUnreadNotifications(w http.ResponseWriter, r *http.Request) {
	if !h.requireNotificationClient(w) {
		return
	}
	recipientID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	resp, err := h.notification.GetUnreadCount(r.Context(), recipientID)
	if err != nil {
		writeNotificationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	if !h.requireNotificationClient(w) {
		return
	}
	recipientID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	notificationID, ok := notificationIDFromReadPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "notificacion no encontrada")
		return
	}

	resp, err := h.notification.MarkNotificationRead(r.Context(), notificationID, recipientID)
	if err != nil {
		writeNotificationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) getNotificationDevStatus(w http.ResponseWriter, r *http.Request) {
	if !h.requireNotificationClient(w) {
		return
	}
	recipientID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	resp, err := h.notification.GetDevStatus(r.Context(), recipientID, limitFromQuery(r, 5))
	if err != nil {
		writeNotificationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) requireNotificationClient(w http.ResponseWriter) bool {
	if h.notification == nil {
		writeError(w, http.StatusServiceUnavailable, "notification-service no configurado")
		return false
	}
	return true
}

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "identidad de usuario no encontrada")
		return "", false
	}
	return userID, true
}

func (h *Handler) startBookingSaga(w http.ResponseWriter, r *http.Request) {
	patientID, ok := r.Context().Value(userIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "identidad de usuario no encontrada")
		return
	}

	var req startBookingSagaRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.DoctorID == "" || req.SlotID == "" || req.Amount <= 0 || req.Currency == "" || req.PaymentMethodID == "" {
		writeError(w, http.StatusBadRequest, "doctor_id, slot_id, amount, currency y payment_method_id son obligatorios")
		return
	}

	resp, err := h.booking.StartBookingSaga(r.Context(), &pb.StartBookingSagaRequest{
		PatientId:       patientID,
		DoctorId:        req.DoctorID,
		SlotId:          req.SlotID,
		Notes:           req.Notes,
		Amount:          req.Amount,
		Currency:        req.Currency,
		PaymentMethodId: req.PaymentMethodID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"saga":           mapBookingSaga(resp.GetSaga()),
		"booking":        mapBooking(resp.GetBooking()),
		"payment_id":     resp.GetPaymentId(),
		"payment_status": resp.GetPaymentStatus(),
	})
}

func (h *Handler) getBookingSaga(w http.ResponseWriter, r *http.Request) {
	sagaID := valueAfterPrefix(r.URL.Path, "/booking-sagas/")
	if sagaID == "" {
		writeError(w, http.StatusNotFound, "saga no encontrada")
		return
	}

	resp, err := h.booking.GetBookingSaga(r.Context(), &pb.GetBookingSagaRequest{SagaId: sagaID})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	events := make([]bookingSagaEventResponse, 0, len(resp.GetEvents()))
	for _, event := range resp.GetEvents() {
		events = append(events, mapBookingSagaEvent(event))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"saga":   mapBookingSaga(resp.GetSaga()),
		"events": events,
	})
}

type createBookingRequest struct {
	PatientID string `json:"patient_id"`
	DoctorID  string `json:"doctor_id"`
	SlotID    string `json:"slot_id"`
	Notes     string `json:"notes"`
}

func (h *Handler) createBooking(w http.ResponseWriter, r *http.Request) {
	// Obtener el ID del paciente directamente del contexto (inyectado por el middleware)
	patientID, ok := r.Context().Value(userIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "identidad de usuario no encontrada")
		return
	}

	var req createBookingRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}

	// Forzamos que el PatientId sea el del usuario autenticado
	if req.DoctorID == "" || req.SlotID == "" {
		writeError(w, http.StatusBadRequest, "doctor_id y slot_id son obligatorios")
		return
	}

	resp, err := h.booking.CreateBooking(r.Context(), &pb.CreateBookingRequest{
		PatientId: patientID,
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
	patientID, ok := r.Context().Value(userIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "identidad de usuario no encontrada")
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

	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "identidad de usuario no encontrada")
		return
	}

	var req createPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.BookingID == "" || req.Amount <= 0 || req.Currency == "" {
		writeError(w, http.StatusBadRequest, "booking_id, amount y currency son obligatorios")
		return
	}

	resp, err := h.payment.CreatePayment(r.Context(), &paymentpb.CreatePaymentRequest{
		BookingId: req.BookingID,
		UserId:    userID,
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

// ── Availability handlers ──────────────────────────────────────────────

func (h *Handler) getAvailableSlots(w http.ResponseWriter, r *http.Request) {
	specialty := r.URL.Query().Get("specialty")
	fromDate := r.URL.Query().Get("from_date")
	toDate := r.URL.Query().Get("to_date")
	doctorID := r.URL.Query().Get("doctor_id")

	if specialty == "" || fromDate == "" || toDate == "" {
		writeError(w, http.StatusBadRequest, "specialty, from_date y to_date son obligatorios")
		return
	}

	resp, err := h.availability.GetAvailableSlots(r.Context(), &availabilitypb.GetAvailableSlotsRequest{
		Specialty: specialty,
		FromDate:  fromDate,
		ToDate:    toDate,
		DoctorId:  doctorID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	slots := make([]map[string]any, 0, len(resp.GetSlots()))
	for _, s := range resp.GetSlots() {
		slots = append(slots, mapSlot(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"slots": slots})
}

func (h *Handler) getDoctorAgenda(w http.ResponseWriter, r *http.Request) {
	doctorID := valueAfterPrefix(r.URL.Path, "/availability/doctors/")
	if doctorID == "" {
		writeError(w, http.StatusBadRequest, "doctor_id es obligatorio")
		return
	}

	fromDate := r.URL.Query().Get("from_date")
	toDate := r.URL.Query().Get("to_date")
	if fromDate == "" || toDate == "" {
		writeError(w, http.StatusBadRequest, "from_date y to_date son obligatorios")
		return
	}

	resp, err := h.availability.GetDoctorAgenda(r.Context(), &availabilitypb.GetDoctorAgendaRequest{
		DoctorId: doctorID,
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	slots := make([]map[string]any, 0, len(resp.GetSlots()))
	for _, s := range resp.GetSlots() {
		slots = append(slots, mapSlot(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"doctor_id": resp.GetDoctorId(),
		"slots":     slots,
	})
}

type holdSlotRequest struct {
	SlotID    string `json:"slot_id"`
	BookingID string `json:"booking_id"`
	HeldUntil string `json:"held_until"`
}

func (h *Handler) holdSlot(w http.ResponseWriter, r *http.Request) {
	var req holdSlotRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.SlotID == "" || req.BookingID == "" {
		writeError(w, http.StatusBadRequest, "slot_id y booking_id son obligatorios")
		return
	}

	resp, err := h.availability.HoldSlot(r.Context(), &availabilitypb.HoldSlotRequest{
		SlotId:    req.SlotID,
		BookingId: req.BookingID,
		HeldUntil: req.HeldUntil,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"slot_id":    resp.GetSlotId(),
		"status":     resp.GetStatus(),
		"held_until": resp.GetHeldUntil(),
		"booking_id": resp.GetBookingId(),
	})
}

type confirmSlotRequest struct {
	SlotID    string `json:"slot_id"`
	BookingID string `json:"booking_id"`
}

func (h *Handler) confirmSlotBooking(w http.ResponseWriter, r *http.Request) {
	var req confirmSlotRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.SlotID == "" || req.BookingID == "" {
		writeError(w, http.StatusBadRequest, "slot_id y booking_id son obligatorios")
		return
	}

	resp, err := h.availability.ConfirmSlotBooking(r.Context(), &availabilitypb.ConfirmSlotBookingRequest{
		SlotId:    req.SlotID,
		BookingId: req.BookingID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"slot_id":    resp.GetSlotId(),
		"status":     resp.GetStatus(),
		"booking_id": resp.GetBookingId(),
	})
}

type releaseSlotRequest struct {
	SlotID    string `json:"slot_id"`
	BookingID string `json:"booking_id"`
}

func (h *Handler) releaseHeldSlot(w http.ResponseWriter, r *http.Request) {
	var req releaseSlotRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}
	if req.SlotID == "" || req.BookingID == "" {
		writeError(w, http.StatusBadRequest, "slot_id y booking_id son obligatorios")
		return
	}

	resp, err := h.availability.ReleaseHeldSlot(r.Context(), &availabilitypb.ReleaseHeldSlotRequest{
		SlotId:    req.SlotID,
		BookingId: req.BookingID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"slot_id": resp.GetSlotId(),
		"status":  resp.GetStatus(),
	})
}

func mapSlot(s *availabilitypb.Slot) map[string]any {
	if s == nil {
		return map[string]any{}
	}
	return map[string]any{
		"slot_id":    s.GetSlotId(),
		"doctor_id":  s.GetDoctorId(),
		"specialty":  s.GetSpecialty(),
		"start_time": s.GetStartTime(),
		"end_time":   s.GetEndTime(),
		"status":     s.GetStatus(),
	}
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

func notificationIDFromReadPath(path string) (int64, bool) {
	value := strings.TrimPrefix(path, "/notifications/")
	if value == path || value == "" {
		return 0, false
	}
	value = strings.TrimSuffix(value, "/read")
	if value == "" || strings.Contains(value, "/") {
		return 0, false
	}

	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
}

func limitFromQuery(r *http.Request, fallback int) int {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > 50 {
		return 50
	}
	return parsed
}

type bookingSagaResponse struct {
	SagaID             string `json:"saga_id"`
	BookingID          string `json:"booking_id,omitempty"`
	PaymentID          string `json:"payment_id,omitempty"`
	PatientID          string `json:"patient_id,omitempty"`
	DoctorID           string `json:"doctor_id,omitempty"`
	SlotID             string `json:"slot_id,omitempty"`
	Status             string `json:"status"`
	CurrentStep        string `json:"current_step,omitempty"`
	CompensationStatus string `json:"compensation_status,omitempty"`
	RetryCount         int32  `json:"retry_count,omitempty"`
	LastError          string `json:"last_error,omitempty"`
	CreatedAt          string `json:"created_at,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
	CompletedAt        string `json:"completed_at,omitempty"`
}

type bookingSagaEventResponse struct {
	EventID      string         `json:"event_id"`
	SagaID       string         `json:"saga_id"`
	EventType    string         `json:"event_type"`
	Step         string         `json:"step"`
	Status       string         `json:"status"`
	Payload      map[string]any `json:"payload,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	CreatedAt    string         `json:"created_at,omitempty"`
}

func mapBookingSaga(saga *pb.BookingSaga) bookingSagaResponse {
	if saga == nil {
		return bookingSagaResponse{}
	}
	return bookingSagaResponse{
		SagaID:             saga.GetSagaId(),
		BookingID:          saga.GetBookingId(),
		PaymentID:          saga.GetPaymentId(),
		PatientID:          saga.GetPatientId(),
		DoctorID:           saga.GetDoctorId(),
		SlotID:             saga.GetSlotId(),
		Status:             sagaStatusToJSON(saga.GetStatus()),
		CurrentStep:        saga.GetCurrentStep(),
		CompensationStatus: sagaCompensationStatusToJSON(saga.GetCompensationStatus()),
		RetryCount:         saga.GetRetryCount(),
		LastError:          saga.GetLastError(),
		CreatedAt:          timestampToJSON(saga.GetCreatedAt()),
		UpdatedAt:          timestampToJSON(saga.GetUpdatedAt()),
		CompletedAt:        timestampToJSON(saga.GetCompletedAt()),
	}
}

func mapBookingSagaEvent(event *pb.BookingSagaEvent) bookingSagaEventResponse {
	if event == nil {
		return bookingSagaEventResponse{}
	}
	payload := map[string]any(nil)
	if event.GetPayload() != nil {
		payload = event.GetPayload().AsMap()
	}
	return bookingSagaEventResponse{
		EventID:      event.GetEventId(),
		SagaID:       event.GetSagaId(),
		EventType:    event.GetEventType(),
		Step:         event.GetStep(),
		Status:       sagaStatusToJSON(event.GetStatus()),
		Payload:      payload,
		ErrorMessage: event.GetErrorMessage(),
		CreatedAt:    timestampToJSON(event.GetCreatedAt()),
	}
}

func sagaStatusToJSON(value pb.BookingSagaStatus) string {
	switch value {
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_STARTED:
		return "STARTED"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_SLOT_HELD:
		return "SLOT_HELD"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_BOOKING_CREATED:
		return "BOOKING_CREATED"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_PAYMENT_CREATED:
		return "PAYMENT_CREATED"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_PAYMENT_COMPLETED:
		return "PAYMENT_COMPLETED"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_SLOT_CONFIRMED:
		return "SLOT_CONFIRMED"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPLETED:
		return "COMPLETED"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPENSATING:
		return "COMPENSATING"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPENSATED:
		return "COMPENSATED"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_FAILED:
		return "FAILED"
	case pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPENSATION_FAILED:
		return "COMPENSATION_FAILED"
	default:
		return "UNSPECIFIED"
	}
}

func sagaCompensationStatusToJSON(value pb.BookingSagaCompensationStatus) string {
	switch value {
	case pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_NOT_REQUIRED:
		return "NOT_REQUIRED"
	case pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_PENDING:
		return "PENDING"
	case pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_IN_PROGRESS:
		return "IN_PROGRESS"
	case pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_COMPLETED:
		return "COMPLETED"
	case pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_FAILED:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
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
	case codes.FailedPrecondition:
		writeError(w, http.StatusConflict, err.Error())
	case codes.Unavailable:
		writeError(w, http.StatusServiceUnavailable, err.Error())
	case codes.DeadlineExceeded, codes.Unknown:
		writeError(w, http.StatusBadGateway, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeNotificationError(w http.ResponseWriter, err error) {
	var upstreamErr notificationclient.HTTPError
	if errors.As(err, &upstreamErr) {
		statusCode := upstreamErr.StatusCode
		if statusCode < http.StatusBadRequest || statusCode > http.StatusNetworkAuthenticationRequired {
			statusCode = http.StatusBadGateway
		}
		writeError(w, statusCode, upstreamErr.Message)
		return
	}
	writeError(w, http.StatusBadGateway, err.Error())
}

// ── Auth handlers ──────────────────────────────────────────────

type registerUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

func (h *Handler) registerUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuthClient(w) {
		return
	}

	var req registerUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}

	resp, err := h.auth.RegisterUser(r.Context(), &authpb.RegisterUserRequest{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Role:     req.Role,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id":    resp.GetUserId(),
		"role":       resp.GetRole(),
		"is_active":  resp.GetIsActive(),
		"created_at": timestampToJSON(resp.GetCreatedAt()),
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) loginUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuthClient(w) {
		return
	}

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}

	resp, err := h.auth.Login(r.Context(), &authpb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": resp.GetAccessToken(),
		"user_id":      resp.GetUserId(),
		"role":         resp.GetRole(),
		"expires_at":   timestampToJSON(resp.GetExpiresAt()),
	})
}

type validateTokenRequest struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) validateToken(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuthClient(w) {
		return
	}

	var req validateTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "json invalido")
		return
	}

	resp, err := h.auth.ValidateToken(r.Context(), &authpb.ValidateTokenRequest{
		AccessToken: req.AccessToken,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":   resp.GetValid(),
		"user_id": resp.GetUserId(),
		"role":    resp.GetRole(),
	})
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuthClient(w) {
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/auth/users/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "ID de usuario requerido")
		return
	}

	resp, err := h.auth.GetUserById(r.Context(), &authpb.GetUserByIdRequest{
		UserId: id,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":    resp.GetUserId(),
		"full_name":  resp.GetFullName(),
		"email":      resp.GetEmail(),
		"role":       resp.GetRole(),
		"is_active":  resp.GetIsActive(),
		"created_at": timestampToJSON(resp.GetCreatedAt()),
	})
}

func (h *Handler) requireAuthClient(w http.ResponseWriter) bool {
	if h.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "auth-service no configurado")
		return false
	}
	return true
}

type contextKey string

const userIDKey contextKey = "user_id"

func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if token == "" || authHeader == token {
			writeError(w, http.StatusUnauthorized, "token de acceso requerido")
			return
		}

		resp, err := h.auth.ValidateToken(r.Context(), &authpb.ValidateTokenRequest{
			AccessToken: token,
		})

		if err != nil || !resp.GetValid() {
			writeError(w, http.StatusUnauthorized, "token inválido o expirado")
			return
		}

		// Inyectar el ID del usuario verificado en el contexto
		ctx := context.WithValue(r.Context(), userIDKey, resp.GetUserId())
		next(w, r.WithContext(ctx))
	}
}

package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	notificationclient "github.com/MedConnect/api-gateway/internal/clients/notification"
	authpb "github.com/MedConnect/auth-service/pb"
	pb "github.com/MedConnect/booking-service/pb"
	paymentpb "github.com/sllanoscaro/payment-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeBookingClient struct {
	createReq  *pb.CreateBookingRequest
	createErr  error
	getReq     *pb.GetBookingRequest
	listReq    *pb.ListBookingsByPatientRequest
	cancelReq  *pb.CancelBookingRequest
	cancelErr  error
	confirmReq *pb.ConfirmBookingRequest
	sagaReq    *pb.StartBookingSagaRequest
	getSagaReq *pb.GetBookingSagaRequest
}

func (c *fakeBookingClient) CreateBooking(_ context.Context, req *pb.CreateBookingRequest) (*pb.CreateBookingResponse, error) {
	c.createReq = req
	if c.createErr != nil {
		return nil, c.createErr
	}
	return &pb.CreateBookingResponse{
		BookingId:     "booking-1",
		Status:        pb.BookingStatus_BOOKING_STATUS_PENDING_PAYMENT,
		ReservedUntil: timestamppb.New(time.Date(2026, time.May, 4, 10, 15, 0, 0, time.UTC)),
	}, nil
}

func (c *fakeBookingClient) GetBooking(_ context.Context, req *pb.GetBookingRequest) (*pb.GetBookingResponse, error) {
	c.getReq = req
	return &pb.GetBookingResponse{
		Booking: &pb.Booking{
			BookingId:     req.GetBookingId(),
			PatientId:     "patient-1",
			DoctorId:      "doctor-1",
			SlotId:        "slot-1",
			Status:        pb.BookingStatus_BOOKING_STATUS_CONFIRMED,
			PaymentId:     "payment-1",
			CreatedAt:     timestamppb.New(time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)),
			UpdatedAt:     timestamppb.New(time.Date(2026, time.May, 4, 10, 5, 0, 0, time.UTC)),
			ReservedUntil: timestamppb.New(time.Date(2026, time.May, 4, 10, 15, 0, 0, time.UTC)),
		},
	}, nil
}

func (c *fakeBookingClient) ListBookingsByPatient(_ context.Context, req *pb.ListBookingsByPatientRequest) (*pb.ListBookingsByPatientResponse, error) {
	c.listReq = req
	return &pb.ListBookingsByPatientResponse{
		Bookings: []*pb.Booking{
			{
				BookingId: "booking-1",
				PatientId: req.GetPatientId(),
				Status:    req.GetStatus(),
			},
		},
	}, nil
}

func (c *fakeBookingClient) CancelBooking(_ context.Context, req *pb.CancelBookingRequest) (*pb.CancelBookingResponse, error) {
	c.cancelReq = req
	if c.cancelErr != nil {
		return nil, c.cancelErr
	}
	return &pb.CancelBookingResponse{
		BookingId: req.GetBookingId(),
		Status:    pb.BookingStatus_BOOKING_STATUS_CANCELLED,
		UpdatedAt: timestamppb.New(time.Date(2026, time.May, 4, 10, 20, 0, 0, time.UTC)),
	}, nil
}

func (c *fakeBookingClient) ConfirmBooking(_ context.Context, req *pb.ConfirmBookingRequest) (*pb.ConfirmBookingResponse, error) {
	c.confirmReq = req
	return &pb.ConfirmBookingResponse{
		BookingId: req.GetBookingId(),
		Status:    pb.BookingStatus_BOOKING_STATUS_CONFIRMED,
		UpdatedAt: timestamppb.New(time.Date(2026, time.May, 4, 10, 30, 0, 0, time.UTC)),
	}, nil
}

func (c *fakeBookingClient) StartBookingSaga(_ context.Context, req *pb.StartBookingSagaRequest) (*pb.StartBookingSagaResponse, error) {
	c.sagaReq = req
	return &pb.StartBookingSagaResponse{
		Saga: &pb.BookingSaga{
			SagaId:             "saga-1",
			BookingId:          "booking-1",
			PaymentId:          "payment-1",
			PatientId:          req.GetPatientId(),
			DoctorId:           req.GetDoctorId(),
			SlotId:             req.GetSlotId(),
			Status:             pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPLETED,
			CurrentStep:        "CONFIRM_BOOKING",
			CompensationStatus: pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_NOT_REQUIRED,
			CreatedAt:          timestamppb.New(time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)),
			UpdatedAt:          timestamppb.New(time.Date(2026, time.May, 4, 10, 2, 0, 0, time.UTC)),
			CompletedAt:        timestamppb.New(time.Date(2026, time.May, 4, 10, 2, 0, 0, time.UTC)),
		},
		Booking: &pb.Booking{
			BookingId: "booking-1",
			PatientId: req.GetPatientId(),
			DoctorId:  req.GetDoctorId(),
			SlotId:    req.GetSlotId(),
			Status:    pb.BookingStatus_BOOKING_STATUS_CONFIRMED,
			PaymentId: "payment-1",
		},
		PaymentId:     "payment-1",
		PaymentStatus: "COMPLETED",
	}, nil
}

func (c *fakeBookingClient) GetBookingSaga(_ context.Context, req *pb.GetBookingSagaRequest) (*pb.GetBookingSagaResponse, error) {
	c.getSagaReq = req
	return &pb.GetBookingSagaResponse{
		Saga: &pb.BookingSaga{
			SagaId:             req.GetSagaId(),
			BookingId:          "booking-1",
			Status:             pb.BookingSagaStatus_BOOKING_SAGA_STATUS_COMPLETED,
			CurrentStep:        "CONFIRM_BOOKING",
			CompensationStatus: pb.BookingSagaCompensationStatus_BOOKING_SAGA_COMPENSATION_STATUS_NOT_REQUIRED,
		},
		Events: []*pb.BookingSagaEvent{
			{
				EventId:   "event-1",
				SagaId:    req.GetSagaId(),
				EventType: "SAGA_STARTED",
				Step:      "START",
				Status:    pb.BookingSagaStatus_BOOKING_SAGA_STATUS_STARTED,
				CreatedAt: timestamppb.New(time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)),
			},
		},
	}, nil
}

type fakeAuthClient struct{}

func (fakeAuthClient) RegisterUser(context.Context, *authpb.RegisterUserRequest) (*authpb.RegisterUserResponse, error) {
	return &authpb.RegisterUserResponse{}, nil
}

func (fakeAuthClient) Login(context.Context, *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	return &authpb.LoginResponse{}, nil
}

func (fakeAuthClient) ValidateToken(context.Context, *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	return &authpb.ValidateTokenResponse{Valid: true, UserId: "patient-1", Role: "PATIENT"}, nil
}

func (fakeAuthClient) GetUserById(context.Context, *authpb.GetUserByIdRequest) (*authpb.GetUserByIdResponse, error) {
	return &authpb.GetUserByIdResponse{}, nil
}

type fakePaymentClient struct {
	createReq       *paymentpb.CreatePaymentRequest
	processReq      *paymentpb.ProcessPaymentRequest
	getReq          *paymentpb.GetPaymentRequest
	listByUserReq   *paymentpb.GetPaymentsByUserRequest
	getByBookingReq *paymentpb.GetPaymentByBookingRequest
	refundReq       *paymentpb.RefundPaymentRequest
}

type fakeNotificationClient struct {
	listRecipient   string
	listLimit       int
	unreadRecipient string
	readID          int64
	readRecipient   string
	devRecipient    string
	devLimit        int
}

func (c *fakeNotificationClient) ListNotifications(_ context.Context, recipientID string, limit int) (*notificationclient.ListResponse, error) {
	c.listRecipient = recipientID
	c.listLimit = limit
	return &notificationclient.ListResponse{
		Notifications: []notificationclient.Notification{
			{
				ID:          7,
				EventID:     "event-1",
				BookingID:   "booking-1",
				EventType:   "booking.created",
				RecipientID: recipientID,
				Channel:     "IN_APP",
				Message:     "Reserva creada.",
				Status:      "SENT",
				CreatedAt:   "2026-05-04T13:00:00Z",
			},
		},
	}, nil
}

func (c *fakeNotificationClient) GetUnreadCount(_ context.Context, recipientID string) (*notificationclient.UnreadCountResponse, error) {
	c.unreadRecipient = recipientID
	return &notificationclient.UnreadCountResponse{UnreadCount: 1}, nil
}

func (c *fakeNotificationClient) MarkNotificationRead(_ context.Context, notificationID int64, recipientID string) (*notificationclient.MarkReadResponse, error) {
	c.readID = notificationID
	c.readRecipient = recipientID
	return &notificationclient.MarkReadResponse{ID: notificationID, ReadAt: "2026-05-04T13:05:00Z"}, nil
}

func (c *fakeNotificationClient) GetDevStatus(_ context.Context, recipientID string, limit int) (*notificationclient.DevStatusResponse, error) {
	c.devRecipient = recipientID
	c.devLimit = limit
	return &notificationclient.DevStatusResponse{
		Service:            "notification-service",
		Store:              "postgres",
		BookingTopic:       "medconnect.booking.events.v1",
		DLQTopic:           "medconnect.booking.events.dlq.v1",
		ConsumerGroup:      "medconnect-notification-service-v1",
		TotalNotifications: 2,
		RecipientID:        recipientID,
		RecipientTotal:     1,
		RecipientUnread:    1,
	}, nil
}

func (c *fakePaymentClient) CreatePayment(_ context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	c.createReq = req
	return &paymentpb.CreatePaymentResponse{
		Payment: &paymentpb.Payment{
			PaymentId: "payment-1",
			BookingId: req.GetBookingId(),
			UserId:    req.GetUserId(),
			Amount:    req.GetAmount(),
			Currency:  req.GetCurrency(),
			Status:    "PENDING",
		},
	}, nil
}

func (c *fakePaymentClient) ProcessPayment(_ context.Context, req *paymentpb.ProcessPaymentRequest) (*paymentpb.ProcessPaymentResponse, error) {
	c.processReq = req
	return &paymentpb.ProcessPaymentResponse{
		PaymentId:     req.GetPaymentId(),
		TransactionId: "txn-1",
		Status:        "COMPLETED",
	}, nil
}

func (c *fakePaymentClient) GetPayment(_ context.Context, req *paymentpb.GetPaymentRequest) (*paymentpb.GetPaymentResponse, error) {
	c.getReq = req
	return &paymentpb.GetPaymentResponse{
		Payment: &paymentpb.Payment{
			PaymentId: req.GetPaymentId(),
			BookingId: "booking-1",
			UserId:    "patient-1",
			Amount:    15000,
			Currency:  "CLP",
			Status:    "COMPLETED",
		},
	}, nil
}

func (c *fakePaymentClient) GetPaymentsByUser(_ context.Context, req *paymentpb.GetPaymentsByUserRequest) (*paymentpb.GetPaymentsByUserResponse, error) {
	c.listByUserReq = req
	return &paymentpb.GetPaymentsByUserResponse{
		Payments: []*paymentpb.Payment{
			{
				PaymentId: "payment-1",
				UserId:    req.GetUserId(),
				Status:    "PENDING",
			},
		},
	}, nil
}

func (c *fakePaymentClient) GetPaymentByBooking(_ context.Context, req *paymentpb.GetPaymentByBookingRequest) (*paymentpb.GetPaymentByBookingResponse, error) {
	c.getByBookingReq = req
	return &paymentpb.GetPaymentByBookingResponse{
		Payment: &paymentpb.Payment{
			PaymentId: "payment-1",
			BookingId: req.GetBookingId(),
			Status:    "PENDING",
		},
	}, nil
}

func (c *fakePaymentClient) RefundPayment(_ context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error) {
	c.refundReq = req
	return &paymentpb.RefundPaymentResponse{
		Refund: &paymentpb.Refund{
			RefundId:  "refund-1",
			PaymentId: req.GetPaymentId(),
			Amount:    req.GetAmount(),
			Reason:    req.GetReason(),
			Status:    "REFUNDED",
		},
	}, nil
}

func TestCreateBookingEndpointTranslatesHTTPToGRPC(t *testing.T) {
	client := &fakeBookingClient{}
	handler := NewHandler(client, nil, nil, fakeAuthClient{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/bookings", strings.NewReader(`{
		"patient_id":"patient-1",
		"doctor_id":"doctor-1",
		"slot_id":"slot-1",
		"notes":"Control anual"
	}`))
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if client.createReq.GetPatientId() != "patient-1" {
		t.Fatalf("expected patient_id to be sent to booking-service")
	}
	if !strings.Contains(rec.Body.String(), `"booking_id":"booking-1"`) {
		t.Fatalf("expected booking id response, got %s", rec.Body.String())
	}
}

func TestGetBookingEndpointUsesPathID(t *testing.T) {
	client := &fakeBookingClient{}
	handler := NewHandler(client, nil, nil, fakeAuthClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/bookings/booking-1", nil)
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if client.getReq.GetBookingId() != "booking-1" {
		t.Fatalf("expected booking id booking-1, got %q", client.getReq.GetBookingId())
	}
}

func TestListBookingsEndpointRequiresPatientIDAndMapsStatus(t *testing.T) {
	client := &fakeBookingClient{}
	handler := NewHandler(client, nil, nil, fakeAuthClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/bookings?patient_id=patient-1&status=PENDING_PAYMENT", nil)
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if client.listReq.GetPatientId() != "patient-1" {
		t.Fatalf("expected patient id patient-1")
	}
	if client.listReq.GetStatus() != pb.BookingStatus_BOOKING_STATUS_PENDING_PAYMENT {
		t.Fatalf("expected pending payment status, got %s", client.listReq.GetStatus())
	}
}

func TestCancelBookingEndpointUsesPatchSubresource(t *testing.T) {
	client := &fakeBookingClient{}
	handler := NewHandler(client, nil, nil, fakeAuthClient{}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/bookings/booking-1/cancel", strings.NewReader(`{"reason":"patient request"}`))
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if client.cancelReq.GetBookingId() != "booking-1" {
		t.Fatalf("expected cancel booking id booking-1")
	}
	if client.cancelReq.GetReason() != "patient request" {
		t.Fatalf("expected cancel reason to be forwarded")
	}
}

func TestCancelBookingEndpointMapsFailedPreconditionToConflict(t *testing.T) {
	client := &fakeBookingClient{
		cancelErr: status.Error(codes.FailedPrecondition, "estado de reserva invalido"),
	}
	handler := NewHandler(client, nil, nil, fakeAuthClient{}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/bookings/booking-1/cancel", strings.NewReader(`{"reason":"patient request"}`))
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestConfirmBookingEndpointUsesPostSubresource(t *testing.T) {
	client := &fakeBookingClient{}
	handler := NewHandler(client, nil, nil, fakeAuthClient{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/bookings/booking-1/confirm", strings.NewReader(`{"payment_id":"payment-1"}`))
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if client.confirmReq.GetBookingId() != "booking-1" {
		t.Fatalf("expected confirm booking id booking-1")
	}
	if client.confirmReq.GetPaymentId() != "payment-1" {
		t.Fatalf("expected payment id to be forwarded")
	}
}

func TestStartBookingSagaEndpointTranslatesHTTPToGRPC(t *testing.T) {
	client := &fakeBookingClient{}
	handler := NewHandler(client, nil, nil, fakeAuthClient{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/booking-sagas", strings.NewReader(`{
		"doctor_id":"doctor-1",
		"slot_id":"slot-1",
		"notes":"Control anual",
		"amount":15000,
		"currency":"CLP",
		"payment_method_id":"method-demo"
	}`))
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if client.sagaReq.GetPatientId() != "patient-1" {
		t.Fatalf("expected authenticated patient id, got %q", client.sagaReq.GetPatientId())
	}
	if client.sagaReq.GetPaymentMethodId() != "method-demo" {
		t.Fatalf("expected payment method to be forwarded")
	}
	if !strings.Contains(rec.Body.String(), `"saga_id":"saga-1"`) {
		t.Fatalf("expected saga id response, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"payment_status":"COMPLETED"`) {
		t.Fatalf("expected payment status response, got %s", rec.Body.String())
	}
}

func TestGetBookingSagaEndpointUsesPathID(t *testing.T) {
	client := &fakeBookingClient{}
	handler := NewHandler(client, nil, nil, fakeAuthClient{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/booking-sagas/saga-1", nil)
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if client.getSagaReq.GetSagaId() != "saga-1" {
		t.Fatalf("expected saga id saga-1, got %q", client.getSagaReq.GetSagaId())
	}
	if !strings.Contains(rec.Body.String(), `"event_type":"SAGA_STARTED"`) {
		t.Fatalf("expected saga events response, got %s", rec.Body.String())
	}
}

func TestListNotificationsUsesAuthenticatedUser(t *testing.T) {
	notificationClient := &fakeNotificationClient{}
	handler := NewHandler(nil, nil, nil, fakeAuthClient{}, notificationClient)
	req := httptest.NewRequest(http.MethodGet, "/notifications?limit=3", nil)
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if notificationClient.listRecipient != "patient-1" {
		t.Fatalf("expected authenticated patient id, got %q", notificationClient.listRecipient)
	}
	if notificationClient.listLimit != 3 {
		t.Fatalf("expected limit 3, got %d", notificationClient.listLimit)
	}
	if !strings.Contains(rec.Body.String(), `"message":"Reserva creada."`) {
		t.Fatalf("expected notification response, got %s", rec.Body.String())
	}
}

func TestNotificationUnreadAndReadEndpoints(t *testing.T) {
	notificationClient := &fakeNotificationClient{}
	handler := NewHandler(nil, nil, nil, fakeAuthClient{}, notificationClient)

	countReq := httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	addAuth(countReq)
	countRec := httptest.NewRecorder()
	handler.ServeHTTP(countRec, countReq)

	if countRec.Code != http.StatusOK {
		t.Fatalf("expected count status 200, got %d: %s", countRec.Code, countRec.Body.String())
	}
	if notificationClient.unreadRecipient != "patient-1" {
		t.Fatalf("expected unread recipient patient-1, got %q", notificationClient.unreadRecipient)
	}
	if !strings.Contains(countRec.Body.String(), `"unread_count":1`) {
		t.Fatalf("expected unread count response, got %s", countRec.Body.String())
	}

	readReq := httptest.NewRequest(http.MethodPatch, "/notifications/7/read", nil)
	addAuth(readReq)
	readRec := httptest.NewRecorder()
	handler.ServeHTTP(readRec, readReq)

	if readRec.Code != http.StatusOK {
		t.Fatalf("expected read status 200, got %d: %s", readRec.Code, readRec.Body.String())
	}
	if notificationClient.readID != 7 || notificationClient.readRecipient != "patient-1" {
		t.Fatalf("expected read id 7 for patient-1, got id=%d recipient=%q", notificationClient.readID, notificationClient.readRecipient)
	}
}

func TestNotificationDevStatusEndpointUsesAuthenticatedUser(t *testing.T) {
	notificationClient := &fakeNotificationClient{}
	handler := NewHandler(nil, nil, nil, fakeAuthClient{}, notificationClient)
	req := httptest.NewRequest(http.MethodGet, "/notifications/dev/status?limit=4", nil)
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if notificationClient.devRecipient != "patient-1" || notificationClient.devLimit != 4 {
		t.Fatalf("expected dev request for patient-1 with limit 4, got recipient=%q limit=%d", notificationClient.devRecipient, notificationClient.devLimit)
	}
	if !strings.Contains(rec.Body.String(), `"booking_topic":"medconnect.booking.events.v1"`) {
		t.Fatalf("expected dev status response, got %s", rec.Body.String())
	}
}

func TestWriteGRPCErrorMapsUnavailableToServiceUnavailable(t *testing.T) {
	rec := httptest.NewRecorder()

	writeGRPCError(rec, status.Error(codes.Unavailable, "availability-service no disponible"))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWriteGRPCErrorMapsFailedPreconditionToConflict(t *testing.T) {
	rec := httptest.NewRecorder()

	writeGRPCError(rec, status.Error(codes.FailedPrecondition, "ya existe una reserva activa para el slot"))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreatePaymentEndpointTranslatesHTTPToGRPC(t *testing.T) {
	bookingClient := &fakeBookingClient{}
	paymentClient := &fakePaymentClient{}
	handler := NewHandler(bookingClient, paymentClient, nil, fakeAuthClient{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(`{
		"booking_id":"booking-1",
		"user_id":"patient-1",
		"amount":15000,
		"currency":"CLP"
	}`))
	addAuth(req)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if paymentClient.createReq.GetBookingId() != "booking-1" {
		t.Fatalf("expected booking id to be sent to payment-service")
	}
	if !strings.Contains(rec.Body.String(), `"payment_id":"payment-1"`) {
		t.Fatalf("expected payment id response, got %s", rec.Body.String())
	}
}

func addAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer test-token")
}

func TestPaymentActionEndpointsUsePathIDs(t *testing.T) {
	bookingClient := &fakeBookingClient{}
	paymentClient := &fakePaymentClient{}
	handler := NewHandler(bookingClient, paymentClient, nil, nil, nil)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
		check  func(t *testing.T)
	}{
		{
			name:   "process",
			method: http.MethodPost,
			path:   "/payments/payment-1/process",
			body:   `{"payment_method_id":"method-1"}`,
			want:   http.StatusOK,
			check: func(t *testing.T) {
				if paymentClient.processReq.GetPaymentId() != "payment-1" {
					t.Fatalf("expected process payment id payment-1")
				}
			},
		},
		{
			name:   "get",
			method: http.MethodGet,
			path:   "/payments/payment-1",
			want:   http.StatusOK,
			check: func(t *testing.T) {
				if paymentClient.getReq.GetPaymentId() != "payment-1" {
					t.Fatalf("expected get payment id payment-1")
				}
			},
		},
		{
			name:   "list by user",
			method: http.MethodGet,
			path:   "/payments/user/patient-1",
			want:   http.StatusOK,
			check: func(t *testing.T) {
				if paymentClient.listByUserReq.GetUserId() != "patient-1" {
					t.Fatalf("expected user id patient-1")
				}
			},
		},
		{
			name:   "get by booking",
			method: http.MethodGet,
			path:   "/payments/booking/booking-1",
			want:   http.StatusOK,
			check: func(t *testing.T) {
				if paymentClient.getByBookingReq.GetBookingId() != "booking-1" {
					t.Fatalf("expected booking id booking-1")
				}
			},
		},
		{
			name:   "refund",
			method: http.MethodPost,
			path:   "/payments/payment-1/refund",
			body:   `{"amount":1000,"reason":"patient request"}`,
			want:   http.StatusOK,
			check: func(t *testing.T) {
				if paymentClient.refundReq.GetPaymentId() != "payment-1" {
					t.Fatalf("expected refund payment id payment-1")
				}
				if paymentClient.refundReq.GetReason() != "patient request" {
					t.Fatalf("expected refund reason to be forwarded")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("expected status %d, got %d: %s", tc.want, rec.Code, rec.Body.String())
			}
			tc.check(t)
		})
	}
}

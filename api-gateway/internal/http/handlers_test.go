package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pb "github.com/MedConnect/booking-service/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeBookingClient struct {
	createReq  *pb.CreateBookingRequest
	getReq     *pb.GetBookingRequest
	listReq    *pb.ListBookingsByPatientRequest
	cancelReq  *pb.CancelBookingRequest
	confirmReq *pb.ConfirmBookingRequest
}

func (c *fakeBookingClient) CreateBooking(_ context.Context, req *pb.CreateBookingRequest) (*pb.CreateBookingResponse, error) {
	c.createReq = req
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

func TestCreateBookingEndpointTranslatesHTTPToGRPC(t *testing.T) {
	client := &fakeBookingClient{}
	handler := NewHandler(client)
	req := httptest.NewRequest(http.MethodPost, "/bookings", strings.NewReader(`{
		"patient_id":"patient-1",
		"doctor_id":"doctor-1",
		"slot_id":"slot-1",
		"notes":"Control anual"
	}`))
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
	handler := NewHandler(client)
	req := httptest.NewRequest(http.MethodGet, "/bookings/booking-1", nil)
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
	handler := NewHandler(client)
	req := httptest.NewRequest(http.MethodGet, "/bookings?patient_id=patient-1&status=PENDING_PAYMENT", nil)
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
	handler := NewHandler(client)
	req := httptest.NewRequest(http.MethodPatch, "/bookings/booking-1/cancel", strings.NewReader(`{"reason":"patient request"}`))
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

func TestConfirmBookingEndpointUsesPostSubresource(t *testing.T) {
	client := &fakeBookingClient{}
	handler := NewHandler(client)
	req := httptest.NewRequest(http.MethodPost, "/bookings/booking-1/confirm", strings.NewReader(`{"payment_id":"payment-1"}`))
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

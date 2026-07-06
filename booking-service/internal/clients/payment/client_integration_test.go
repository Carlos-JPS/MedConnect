package payment

import (
	"context"
	"net"
	"testing"

	"github.com/MedConnect/booking-service/internal/service"
	paymentpb "github.com/sllanoscaro/payment-service/pb"
	"google.golang.org/grpc"
)

type fakePaymentServiceServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	createReq       *paymentpb.CreatePaymentRequest
	processReq      *paymentpb.ProcessPaymentRequest
	getByBookingReq *paymentpb.GetPaymentByBookingRequest
	refundReq       *paymentpb.RefundPaymentRequest
}

func (s *fakePaymentServiceServer) CreatePayment(_ context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	s.createReq = req
	return &paymentpb.CreatePaymentResponse{
		Payment: &paymentpb.Payment{
			PaymentId: "payment-created",
			BookingId: req.GetBookingId(),
			UserId:    req.GetUserId(),
			Amount:    req.GetAmount(),
			Currency:  req.GetCurrency(),
			Status:    "pending",
		},
	}, nil
}

func (s *fakePaymentServiceServer) ProcessPayment(_ context.Context, req *paymentpb.ProcessPaymentRequest) (*paymentpb.ProcessPaymentResponse, error) {
	s.processReq = req
	return &paymentpb.ProcessPaymentResponse{
		PaymentId:     req.GetPaymentId(),
		TransactionId: "txn-1",
		Status:        "completed",
	}, nil
}

func (s *fakePaymentServiceServer) GetPayment(context.Context, *paymentpb.GetPaymentRequest) (*paymentpb.GetPaymentResponse, error) {
	return &paymentpb.GetPaymentResponse{
		Payment: &paymentpb.Payment{
			PaymentId: "payment-1",
			BookingId: "booking-1",
			UserId:    "patient-1",
			Amount:    15000,
			Currency:  "CLP",
			Status:    "COMPLETED",
		},
	}, nil
}

func (s *fakePaymentServiceServer) GetPaymentByBooking(_ context.Context, req *paymentpb.GetPaymentByBookingRequest) (*paymentpb.GetPaymentByBookingResponse, error) {
	s.getByBookingReq = req
	return &paymentpb.GetPaymentByBookingResponse{
		Payment: &paymentpb.Payment{
			PaymentId: "payment-for-booking",
			BookingId: req.GetBookingId(),
			Status:    "refunded",
		},
	}, nil
}

func (s *fakePaymentServiceServer) RefundPayment(_ context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error) {
	s.refundReq = req
	return &paymentpb.RefundPaymentResponse{
		Refund: &paymentpb.Refund{
			RefundId:  "refund-1",
			PaymentId: req.GetPaymentId(),
			Amount:    req.GetAmount(),
			Reason:    req.GetReason(),
			Status:    "refunded",
		},
	}, nil
}

func TestGRPCClientUsesPaymentServiceContract(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	fakeServer := &fakePaymentServiceServer{}
	paymentpb.RegisterPaymentServiceServer(server, fakeServer)
	go func() {
		_ = server.Serve(lis)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = lis.Close()
	})

	client, err := NewGRPCClient(lis.Addr().String())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()

	created, err := client.CreatePayment(ctx, service.CreatePaymentInput{
		BookingID: "booking-1",
		UserID:    "patient-1",
		Amount:    15000,
		Currency:  "CLP",
	})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if fakeServer.createReq.GetBookingId() != "booking-1" || fakeServer.createReq.GetUserId() != "patient-1" {
		t.Fatalf("unexpected create request: %+v", fakeServer.createReq)
	}
	if created.PaymentID != "payment-created" || created.Status != service.PaymentStatusPending {
		t.Fatalf("unexpected created payment: %+v", created)
	}

	processed, err := client.ProcessPayment(ctx, service.ProcessPaymentInput{
		PaymentID:       "payment-created",
		PaymentMethodID: "method-demo",
	})
	if err != nil {
		t.Fatalf("process payment: %v", err)
	}
	if fakeServer.processReq.GetPaymentMethodId() != "method-demo" {
		t.Fatalf("unexpected process request: %+v", fakeServer.processReq)
	}
	if processed.TransactionID != "txn-1" || processed.Status != service.PaymentStatusCompleted {
		t.Fatalf("unexpected processed payment: %+v", processed)
	}

	payment, err := client.GetPayment(ctx, "payment-1")
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	if payment.Status != service.PaymentStatusCompleted {
		t.Fatalf("expected COMPLETED status, got %q", payment.Status)
	}
	if payment.BookingID != "booking-1" {
		t.Fatalf("expected booking-1, got %q", payment.BookingID)
	}
	if payment.UserID != "patient-1" || payment.Amount != 15000 || payment.Currency != "CLP" {
		t.Fatalf("unexpected payment details: %+v", payment)
	}

	byBooking, err := client.GetPaymentByBooking(ctx, "booking-2")
	if err != nil {
		t.Fatalf("get payment by booking: %v", err)
	}
	if fakeServer.getByBookingReq.GetBookingId() != "booking-2" {
		t.Fatalf("unexpected get by booking request: %+v", fakeServer.getByBookingReq)
	}
	if byBooking.PaymentID != "payment-for-booking" || byBooking.Status != service.PaymentStatusRefunded {
		t.Fatalf("unexpected payment by booking: %+v", byBooking)
	}

	refund, err := client.RefundPayment(ctx, service.RefundPaymentInput{
		PaymentID: "payment-created",
		Amount:    15000,
		Reason:    "saga compensation",
	})
	if err != nil {
		t.Fatalf("refund payment: %v", err)
	}
	if fakeServer.refundReq.GetReason() != "saga compensation" {
		t.Fatalf("unexpected refund request: %+v", fakeServer.refundReq)
	}
	if refund.RefundID != "refund-1" || refund.Status != service.PaymentStatusRefunded {
		t.Fatalf("unexpected refund: %+v", refund)
	}
}

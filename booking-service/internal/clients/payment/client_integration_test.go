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
}

func (fakePaymentServiceServer) GetPayment(context.Context, *paymentpb.GetPaymentRequest) (*paymentpb.GetPaymentResponse, error) {
	return &paymentpb.GetPaymentResponse{
		Payment: &paymentpb.Payment{
			PaymentId: "payment-1",
			BookingId: "booking-1",
			Status:    "COMPLETED",
		},
	}, nil
}

func TestGRPCClientUsesPaymentServiceContract(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(server, fakePaymentServiceServer{})
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

	payment, err := client.GetPayment(context.Background(), "payment-1")
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	if payment.Status != service.PaymentStatusCompleted {
		t.Fatalf("expected COMPLETED status, got %q", payment.Status)
	}
	if payment.BookingID != "booking-1" {
		t.Fatalf("expected booking-1, got %q", payment.BookingID)
	}
}

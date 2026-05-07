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

	status, err := client.GetPaymentStatus(context.Background(), "payment-1")
	if err != nil {
		t.Fatalf("get payment status: %v", err)
	}
	if status != service.PaymentStatusCompleted {
		t.Fatalf("expected COMPLETED status, got %q", status)
	}
}

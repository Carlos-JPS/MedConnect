package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "github.com/sllanoscaro/payment-service/pb"
)

type server struct {
	pb.UnimplementedPaymentServiceServer
}

func (s *server) GetPayment(
	ctx context.Context,
	req *pb.GetPaymentRequest,
) (*pb.GetPaymentResponse, error) {

	return &pb.GetPaymentResponse{
		Payment: &pb.Payment{
			PaymentId: req.PaymentId,
			BookingId: "booking-123",
			UserId:    "user-42",
			Amount:    25000,
			Currency:  "CLP",
			Status:    "completed",
			CreatedAt: "2026-04-30T10:00:00Z",
		},
	}, nil
}

func main() {

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("no se pudo escuchar: %v", err)
	}

	srv := grpc.NewServer()

	pb.RegisterPaymentServiceServer(srv, &server{})

	log.Println("servidor gRPC escuchando en :50051")

	if err := srv.Serve(lis); err != nil {
		log.Fatalf("error al servir: %v", err)
	}
}

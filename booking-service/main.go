package main

import (
	"context"
	"log"
	"net"

	pb "github.com/MedConnect/booking-service/pb"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedBookingServiceServer
}

// GetBooking: implementación mínima con datos hardcodeados
func (s *server) GetBooking(ctx context.Context, req *pb.GetBookingRequest) (*pb.GetBookingResponse, error) {
	log.Printf("Recibida petición GetBooking para el ID: %s", req.GetBookingId())
	
	return &pb.GetBookingResponse{
		Booking: &pb.Booking{
			BookingId:  req.GetBookingId(),
			PatientId:  "patient-123",
			DoctorId:   "doctor-456",
			SlotId:     "slot-789",
			Status:     "CONFIRMED",
			PaymentId:  "payment-001",
			CreatedAt:  "2026-04-30T10:00:00Z",
			ExpiresAt:  "2026-04-30T11:00:00Z",
			UpdatedAt:  "2026-04-30T10:05:00Z",
		},
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Error al escuchar en el puerto 50051: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterBookingServiceServer(srv, &server{})

	log.Println("Servidor gRPC de booking-service escuchando en :50051")
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("Error al arrancar el servidor: %v", err)
	}
}

package main

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/MedConnect/booking-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type server struct {
	pb.UnimplementedBookingServiceServer
}

// GetBooking: implementación mínima con datos hardcodeados
func (s *server) GetBooking(ctx context.Context, req *pb.GetBookingRequest) (*pb.GetBookingResponse, error) {
	log.Printf("Recibida petición GetBooking para el ID: %s", req.GetBookingId())

	createdAt := time.Date(2026, time.April, 30, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	reservedUntil := createdAt.Add(1 * time.Hour)

	return &pb.GetBookingResponse{
		Booking: &pb.Booking{
			BookingId:        req.GetBookingId(),
			PatientId:        "patient-123",
			DoctorId:         "doctor-456",
			SlotId:           "slot-789",
			Status:           pb.BookingStatus_BOOKING_STATUS_CONFIRMED,
			PaymentId:        "payment-001",
			ConfirmationCode: "MED-001",
			Notes:            "Control general",
			CreatedAt:        timestamppb.New(createdAt),
			UpdatedAt:        timestamppb.New(updatedAt),
			ReservedUntil:    timestamppb.New(reservedUntil),
			ConfirmedAt:      timestamppb.New(updatedAt),
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

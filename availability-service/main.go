package main

import (
	"context"
	"log"
	"net"

	pb "github.com/Carlos-JPS/medconnect/availability-service/pb"
	"google.golang.org/grpc"
)

// server implementa la interfaz AvailabilityServiceServer generada por protoc
type server struct {
	pb.UnimplementedAvailabilityServiceServer
}

// GetAvailableSlots: implementación mínima con datos hardcodeados
func (s *server) GetAvailableSlots(ctx context.Context, req *pb.GetAvailableSlotsRequest) (*pb.GetAvailableSlotsResponse, error) {
	log.Printf("Recibida petición de slots para especialidad: %s desde %s hasta %s", req.Specialty, req.FromDate, req.ToDate)

	// Datos hardcodeados para probar que el servidor responde
	return &pb.GetAvailableSlotsResponse{
		Slots: []*pb.Slot{
			{
				SlotId:    "123e4567-e89b-12d3-a456-426614174000",
				DoctorId:  "doc-001",
				Specialty: req.Specialty,
				StartTime: "2024-05-10T09:00:00Z",
				EndTime:   "2024-05-10T09:30:00Z",
				Status:    "available",
			},
			{
				SlotId:    "123e4567-e89b-12d3-a456-426614174001",
				DoctorId:  "doc-001",
				Specialty: req.Specialty,
				StartTime: "2024-05-10T10:00:00Z",
				EndTime:   "2024-05-10T10:30:00Z",
				Status:    "available",
			},
		},
	}, nil
}

func main() {
	// Escuchar en el puerto 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Error al escuchar en el puerto 50051: %v", err)
	}

	// Crear el servidor gRPC
	srv := grpc.NewServer()

	// Registrar nuestro servicio
	pb.RegisterAvailabilityServiceServer(srv, &server{})

	log.Println("Servidor gRPC de Availability escuchando en :50051")

	// Empezar a servir
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("Error al servir gRPC: %v", err)
	}
}

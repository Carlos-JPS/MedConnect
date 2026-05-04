package main

import (
	"log"
	"net"

	availabilityclient "github.com/MedConnect/booking-service/internal/clients/availability"
	paymentclient "github.com/MedConnect/booking-service/internal/clients/payment"
	"github.com/MedConnect/booking-service/internal/config"
	"github.com/MedConnect/booking-service/internal/repository/postgres"
	"github.com/MedConnect/booking-service/internal/service"
	grpcserver "github.com/MedConnect/booking-service/internal/transport/grpc"
	pb "github.com/MedConnect/booking-service/pb"
	googlegrpc "google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	lis, err := net.Listen("tcp", net.JoinHostPort(cfg.GRPCHost, cfg.GRPCPort))
	if err != nil {
		log.Fatalf("error al escuchar en %s:%s: %v", cfg.GRPCHost, cfg.GRPCPort, err)
	}

	repo, err := postgres.Open(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("error al inicializar repositorio booking: %v", err)
	}
	defer repo.Close()

	_ = availabilityclient.NewNoopClient()
	_ = paymentclient.NewNoopClient()

	bookingService := service.NewBookingService(repo)
	server := googlegrpc.NewServer()
	pb.RegisterBookingServiceServer(server, grpcserver.NewServer(bookingService))

	log.Printf("servidor gRPC de booking-service escuchando en %s:%s", cfg.GRPCHost, cfg.GRPCPort)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("error al arrancar el servidor: %v", err)
	}
}

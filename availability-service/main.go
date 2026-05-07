package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Carlos-JPS/medconnect/availability-service/modules/config"
	"github.com/Carlos-JPS/medconnect/availability-service/modules/handler"
	"github.com/Carlos-JPS/medconnect/availability-service/modules/repository"
	"github.com/Carlos-JPS/medconnect/availability-service/modules/service"
	pb "github.com/Carlos-JPS/medconnect/availability-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()

	repo, err := connectWithRetry(cfg.DSN(), 5, 3*time.Second)
	if err != nil {
		log.Fatalf("no se pudo conectar a la base de datos: %v", err)
	}
	log.Println("conexión a PostgreSQL establecida")

	svc := service.NewAvailabilityService(repo)
	h := handler.NewGRPCHandler(svc)

	addr := fmt.Sprintf(":%s", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("error abriendo puerto %s: %v", addr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAvailabilityServiceServer(grpcServer, h)
	reflection.Register(grpcServer)

	log.Printf("availability-service escuchando en %s (gRPC)", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("error en servidor gRPC: %v", err)
	}
}

func connectWithRetry(dsn string, maxAttempts int, delay time.Duration) (repository.AvailabilityRepository, error) {
	var repo repository.AvailabilityRepository
	var err error

	for i := 1; i <= maxAttempts; i++ {
		repo, err = repository.NewPostgresRepository(dsn)
		if err == nil {
			return repo, nil
		}
		log.Printf("intento %d/%d fallido al conectar a postgres: %v", i, maxAttempts, err)
		if i < maxAttempts {
			time.Sleep(delay)
		}
	}
	return nil, fmt.Errorf("todos los intentos de conexión fallaron: %w", err)
}

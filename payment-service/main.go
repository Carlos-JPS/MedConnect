package main

import (
	"fmt"
	"log"
	"net"
	"time"

	pb "github.com/sllanoscaro/payment-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/sllanoscaro/payment-service/modules/config"
	"github.com/sllanoscaro/payment-service/modules/handler"
	"github.com/sllanoscaro/payment-service/modules/repository"
	"github.com/sllanoscaro/payment-service/modules/service"
)

func main() {
	cfg := config.Load()

	repo, err := connectWithRetry(cfg.DSN(), 5, 3*time.Second)
	if err != nil {
		log.Fatalf("no se pudo conectar a la base de datos: %v", err)
	}
	log.Println("conexión a PostgreSQL establecida")

	processor := service.NewProcessor()
	svc := service.NewPaymentService(repo, processor)
	h := handler.NewPaymentHandler(svc)

	addr := fmt.Sprintf(":%s", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("error abriendo puerto %s: %v", addr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPaymentServiceServer(grpcServer, h)
	reflection.Register(grpcServer)

	log.Printf("payment-service escuchando en %s (gRPC)", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("error en servidor gRPC: %v", err)
	}
}

func connectWithRetry(dsn string, maxAttempts int, delay time.Duration) (repository.PaymentRepository, error) {
	var repo repository.PaymentRepository
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

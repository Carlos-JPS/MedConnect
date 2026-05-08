package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/MedConnect/auth-service/internal/repository/postgres"
	"github.com/MedConnect/auth-service/internal/service"
	transport "github.com/MedConnect/auth-service/internal/transport/grpc"
	"github.com/MedConnect/auth-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Read configuration
	dbDSN := os.Getenv("AUTH_DB_DSN")
	if dbDSN == "" {
		log.Fatal("AUTH_DB_DSN environment variable is not set")
	}

	port := os.Getenv("AUTH_SERVICE_PORT")
	if port == "" {
		port = "50051" // default port
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize layers
	repo := postgres.NewRepository(db)
	svc := service.NewAuthService(repo, jwtSecret)
	handler := transport.NewAuthHandler(svc)

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcServer, handler)
	
	// Register reflection service on gRPC server to allow tools like grpcurl/insomnia to inspect the server
	reflection.Register(grpcServer)

	log.Printf("Starting gRPC server on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

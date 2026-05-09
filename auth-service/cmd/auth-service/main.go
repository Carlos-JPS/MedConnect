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
	// 1. Carga de configuración desde variables de entorno
	dbDSN := os.Getenv("AUTH_DB_DSN")
	if dbDSN == "" {
		log.Fatal("La variable de entorno AUTH_DB_DSN no está configurada")
	}

	port := os.Getenv("AUTH_SERVICE_PORT")
	if port == "" {
		port = "50051" // Puerto por defecto para gRPC
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("La variable de entorno JWT_SECRET no está configurada")
	}

	// 2. Conexión a la base de datos PostgreSQL
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		log.Fatalf("Error al abrir la conexión a la base de datos: %v", err)
	}
	defer db.Close()

	// Verificar que la base de datos sea accesible
	if err := db.Ping(); err != nil {
		log.Fatalf("No se pudo conectar a la base de datos (Ping): %v", err)
	}

	// 3. Inicialización de las capas de la aplicación (Inyección de Dependencias)
	// Repositorio: Acceso a datos
	repo := postgres.NewRepository(db)
	// Servicio: Lógica de negocio (necesita el repositorio y la clave para JWT)
	svc := service.NewAuthService(repo, jwtSecret)
	// Handler: Capa de transporte gRPC
	handler := transport.NewAuthHandler(svc)

	// 4. Configuración del servidor gRPC
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Error al escuchar en el puerto %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	// Registrar el servicio de autenticación en el servidor gRPC
	pb.RegisterAuthServiceServer(grpcServer, handler)
	
	// Habilitar Reflection para que herramientas como grpcurl puedan inspeccionar el servidor
	reflection.Register(grpcServer)

	log.Printf("Iniciando servidor gRPC en el puerto %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Error al ejecutar el servidor gRPC: %v", err)
	}
}

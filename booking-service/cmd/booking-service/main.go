package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	availabilityclient "github.com/MedConnect/booking-service/internal/clients/availability"
	paymentclient "github.com/MedConnect/booking-service/internal/clients/payment"
	"github.com/MedConnect/booking-service/internal/config"
	kafkamessaging "github.com/MedConnect/booking-service/internal/messaging/kafka"
	"github.com/MedConnect/booking-service/internal/outbox"
	"github.com/MedConnect/booking-service/internal/repository/postgres"
	"github.com/MedConnect/booking-service/internal/service"
	grpcserver "github.com/MedConnect/booking-service/internal/transport/grpc"
	pb "github.com/MedConnect/booking-service/pb"
	googlegrpc "google.golang.org/grpc"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	lis, err := net.Listen("tcp", net.JoinHostPort(cfg.GRPCHost, cfg.GRPCPort))
	if err != nil {
		log.Fatalf("error al escuchar en %s:%s: %v", cfg.GRPCHost, cfg.GRPCPort, err)
	}

	repo, err := postgres.Open(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("error al inicializar repositorio booking: %v", err)
	}
	defer repo.Close()
	repo.SetBookingEventsTopic(cfg.BookingEventsTopic)

	availabilityClient, err := availabilityclient.NewGRPCClient(cfg.AvailabilityServiceTarget)
	if err != nil {
		log.Fatalf("error al inicializar cliente availability-service: %v", err)
	}
	defer availabilityClient.Close()

	paymentClient, err := paymentclient.NewGRPCClient(cfg.PaymentServiceTarget)
	if err != nil {
		log.Fatalf("error al inicializar cliente payment-service: %v", err)
	}
	defer paymentClient.Close()

	bookingService := service.NewBookingService(
		repo,
		service.WithAvailabilityClient(availabilityClient),
		service.WithPaymentClient(paymentClient),
		service.WithExternalCallTimeout(cfg.ExternalCallTimeout),
	)
	server := googlegrpc.NewServer()
	pb.RegisterBookingServiceServer(server, grpcserver.NewServer(bookingService))

	if cfg.OutboxDispatcherEnabled {
		publisher, err := kafkamessaging.NewPublisher(cfg.KafkaBrokers)
		if err != nil {
			log.Printf("outbox dispatcher deshabilitado: no se pudo inicializar productor Kafka: %v", err)
		} else {
			defer publisher.Close()
			dispatcher := outbox.NewDispatcher(
				repo,
				publisher,
				outbox.DispatcherConfig{
					BatchSize:         cfg.OutboxBatchSize,
					PollInterval:      cfg.OutboxPollInterval,
					InitialRetryDelay: cfg.OutboxInitialRetryDelay,
					MaxRetryDelay:     cfg.OutboxMaxRetryDelay,
				},
				log.Default(),
			)
			go dispatcher.Run(ctx)
		}
	}

	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	log.Printf("servidor gRPC de booking-service escuchando en %s:%s", cfg.GRPCHost, cfg.GRPCPort)
	if err := server.Serve(lis); err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Fatalf("error al arrancar el servidor: %v", err)
	}
}

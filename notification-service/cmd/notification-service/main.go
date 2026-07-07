package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MedConnect/notification-service/internal/config"
	"github.com/MedConnect/notification-service/internal/consumer"
	"github.com/MedConnect/notification-service/internal/httpapi"
	"github.com/MedConnect/notification-service/internal/observability"
	"github.com/MedConnect/notification-service/internal/processor"
	"github.com/MedConnect/notification-service/internal/repository/postgres"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo, err := postgres.Open(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("error al inicializar repositorio de notificaciones: %v", err)
	}
	defer repo.Close()
	observability.StartMetricsServer("notification-service")

	httpServer := &http.Server{
		Addr: net.JoinHostPort(cfg.HTTPHost, cfg.HTTPPort),
		Handler: httpapi.NewServer(
			repo,
			httpapi.Config{
				KafkaBrokers:  cfg.KafkaBrokers,
				BookingTopic:  cfg.BookingTopic,
				DLQTopic:      cfg.DLQTopic,
				ConsumerGroup: cfg.ConsumerGroup,
			},
			log.Default(),
		),
	}
	go func() {
		log.Printf("notification-service HTTP interno escuchando en %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("error al iniciar HTTP interno de notificaciones: %v", err)
		}
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("error al cerrar HTTP interno de notificaciones: %v", err)
		}
	}()

	dlqPublisher, err := consumer.NewDLQPublisher(cfg.KafkaBrokers, cfg.DLQTopic)
	if err != nil {
		log.Fatalf("error al inicializar productor DLQ: %v", err)
	}
	defer dlqPublisher.Close()

	eventProcessor := processor.New(
		repo,
		dlqPublisher,
		processor.Config{
			RetryLimit:      cfg.RetryLimit,
			NotificationLog: cfg.NotificationLog,
		},
		log.Default(),
	)

	bookingConsumer, err := consumer.NewBookingEventConsumer(
		cfg.KafkaBrokers,
		cfg.BookingTopic,
		cfg.ConsumerGroup,
		eventProcessor,
		log.Default(),
	)
	if err != nil {
		log.Fatalf("error al inicializar consumidor Kafka: %v", err)
	}
	defer bookingConsumer.Close()

	log.Printf("notification-service consumiendo topic=%s group=%s", cfg.BookingTopic, cfg.ConsumerGroup)
	if err := bookingConsumer.Run(ctx); err != nil {
		log.Fatalf("error al ejecutar consumidor de notificaciones: %v", err)
	}
}

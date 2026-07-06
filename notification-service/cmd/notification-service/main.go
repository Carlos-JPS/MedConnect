package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MedConnect/notification-service/internal/config"
	"github.com/MedConnect/notification-service/internal/consumer"
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

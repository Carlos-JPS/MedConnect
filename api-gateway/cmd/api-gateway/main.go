package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/MedConnect/api-gateway/internal/config"
	availabilityclient "github.com/MedConnect/api-gateway/internal/grpc/availability"
	authclient "github.com/MedConnect/api-gateway/internal/grpc/auth"
	bookingclient "github.com/MedConnect/api-gateway/internal/grpc/booking"
	paymentclient "github.com/MedConnect/api-gateway/internal/grpc/payment"
	httpapi "github.com/MedConnect/api-gateway/internal/http"
)

func main() {
	cfg := config.Load()

	bookingClient, err := bookingclient.NewClient(cfg.BookingServiceTarget)
	if err != nil {
		log.Fatalf("error al inicializar cliente booking-service: %v", err)
	}
	defer bookingClient.Close()

	paymentClient, err := paymentclient.NewClient(cfg.PaymentServiceTarget)
	if err != nil {
		log.Fatalf("error al inicializar cliente payment-service: %v", err)
	}
	defer paymentClient.Close()

	availabilityClient, err := availabilityclient.NewClient(cfg.AvailabilityServiceTarget)
	if err != nil {
		log.Fatalf("error al inicializar cliente availability-service: %v", err)
	}
	defer availabilityClient.Close()

	authClient, err := authclient.NewClient(cfg.AuthServiceTarget)
	if err != nil {
		log.Fatalf("error al inicializar cliente auth-service: %v", err)
	}
	defer authClient.Close()

	handler := withTimeout(httpapi.NewHandler(bookingClient, paymentClient, availabilityClient, authClient), cfg.RequestTimeout)
	server := &http.Server{
		Addr:    net.JoinHostPort(cfg.HTTPHost, cfg.HTTPPort),
		Handler: handler,
	}

	log.Printf("api-gateway escuchando en %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("error al iniciar api-gateway: %v", err)
	}
}

func withTimeout(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

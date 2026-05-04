package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/MedConnect/api-gateway/internal/config"
	bookingclient "github.com/MedConnect/api-gateway/internal/grpc/booking"
	httpapi "github.com/MedConnect/api-gateway/internal/http"
)

func main() {
	cfg := config.Load()

	bookingClient, err := bookingclient.NewClient(cfg.BookingServiceTarget)
	if err != nil {
		log.Fatalf("error al inicializar cliente booking-service: %v", err)
	}
	defer bookingClient.Close()

	handler := withTimeout(httpapi.NewHandler(bookingClient), cfg.RequestTimeout)
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

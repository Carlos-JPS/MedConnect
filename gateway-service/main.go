package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sllanoscaro/gateway-service/modules/config"
	"github.com/sllanoscaro/gateway-service/modules/handler"
	pb "github.com/sllanoscaro/gateway-service/pb/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()
	conn, err := grpc.NewClient(
		cfg.PaymentServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("error conectando a payment-service: %v", err)
	}
	defer conn.Close()

	paymentClient := pb.NewPaymentServiceClient(conn)
	paymentHandler := handler.NewPaymentHandler(paymentClient)

	r := gin.Default()
	payments := r.Group("/payments")
	{
		payments.POST("", paymentHandler.CreatePayment)
		payments.POST("/:id/process", paymentHandler.ProcessPayment)
		payments.GET("/:id", paymentHandler.GetPayment)
		payments.GET("/user/:userId", paymentHandler.GetPaymentsByUser)
		payments.GET("/booking/:bookingId", paymentHandler.GetPaymentByBooking)
		payments.POST("/:id/refund", paymentHandler.RefundPayment)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("gateway escuchando en %s (HTTP)", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("error iniciando servidor HTTP: %v", err)
	}
}

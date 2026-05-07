package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/sllanoscaro/gateway-service/pb/payment"
)

type PaymentHandler struct {
	client pb.PaymentServiceClient
}

func NewPaymentHandler(client pb.PaymentServiceClient) *PaymentHandler {
	return &PaymentHandler{client: client}
}

func newCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var body struct {
		BookingID string  `json:"booking_id" binding:"required"`
		UserID    string  `json:"user_id"    binding:"required"`
		Amount    float64 `json:"amount"     binding:"required"`
		Currency  string  `json:"currency"   binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	resp, err := h.client.CreatePayment(ctx, &pb.CreatePaymentRequest{
		BookingId: body.BookingID,
		UserId:    body.UserID,
		Amount:    body.Amount,
		Currency:  body.Currency,
	})
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment-service no disponible", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp.Payment)
}

func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	var body struct {
		PaymentMethodID string `json:"payment_method_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	resp, err := h.client.ProcessPayment(ctx, &pb.ProcessPaymentRequest{
		PaymentId:       c.Param("id"),
		PaymentMethodId: body.PaymentMethodID,
	})
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment-service no disponible", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	ctx, cancel := newCtx()
	defer cancel()

	resp, err := h.client.GetPayment(ctx, &pb.GetPaymentRequest{
		PaymentId: c.Param("id"),
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pago no encontrado", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Payment)
}

func (h *PaymentHandler) GetPaymentsByUser(c *gin.Context) {
	ctx, cancel := newCtx()
	defer cancel()

	resp, err := h.client.GetPaymentsByUser(ctx, &pb.GetPaymentsByUserRequest{
		UserId: c.Param("userId"),
	})
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment-service no disponible", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Payments)
}

func (h *PaymentHandler) GetPaymentByBooking(c *gin.Context) {
	ctx, cancel := newCtx()
	defer cancel()

	resp, err := h.client.GetPaymentByBooking(ctx, &pb.GetPaymentByBookingRequest{
		BookingId: c.Param("bookingId"),
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pago no encontrado para la reserva", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Payment)
}

func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	var body struct {
		Amount float64 `json:"amount" binding:"required"`
		Reason string  `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := newCtx()
	defer cancel()

	resp, err := h.client.RefundPayment(ctx, &pb.RefundPaymentRequest{
		PaymentId: c.Param("id"),
		Amount:    body.Amount,
		Reason:    body.Reason,
	})
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment-service no disponible", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Refund)
}

package payment

import (
	"context"
	"errors"
	"strings"

	grpcmeta "github.com/MedConnect/booking-service/internal/clients/metadata"
	"github.com/MedConnect/booking-service/internal/service"
	pb "github.com/sllanoscaro/payment-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.PaymentServiceClient
}

func NewGRPCClient(address string) (*GRPCClient, error) {
	if address == "" {
		return nil, errors.New("direccion de payment-service no configurada")
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		conn:   conn,
		client: pb.NewPaymentServiceClient(conn),
	}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCClient) CreatePayment(ctx context.Context, input service.CreatePaymentInput) (service.PaymentDetails, error) {
	resp, err := c.client.CreatePayment(grpcmeta.ContextWithRequestID(ctx), &pb.CreatePaymentRequest{
		BookingId: input.BookingID,
		UserId:    input.UserID,
		Amount:    input.Amount,
		Currency:  input.Currency,
	})
	if err != nil {
		return service.PaymentDetails{}, err
	}
	return paymentFromProto(resp.GetPayment())
}

func (c *GRPCClient) ProcessPayment(ctx context.Context, input service.ProcessPaymentInput) (service.ProcessPaymentResult, error) {
	resp, err := c.client.ProcessPayment(grpcmeta.ContextWithRequestID(ctx), &pb.ProcessPaymentRequest{
		PaymentId:       input.PaymentID,
		PaymentMethodId: input.PaymentMethodID,
	})
	if err != nil {
		return service.ProcessPaymentResult{}, err
	}
	return service.ProcessPaymentResult{
		PaymentID:     resp.GetPaymentId(),
		TransactionID: resp.GetTransactionId(),
		Status:        service.PaymentStatus(strings.ToUpper(resp.GetStatus())),
	}, nil
}

func (c *GRPCClient) GetPayment(ctx context.Context, paymentID string) (service.PaymentDetails, error) {
	resp, err := c.client.GetPayment(grpcmeta.ContextWithRequestID(ctx), &pb.GetPaymentRequest{PaymentId: paymentID})
	if err != nil {
		return service.PaymentDetails{}, err
	}
	return paymentFromProto(resp.GetPayment())
}

func (c *GRPCClient) GetPaymentByBooking(ctx context.Context, bookingID string) (service.PaymentDetails, error) {
	resp, err := c.client.GetPaymentByBooking(grpcmeta.ContextWithRequestID(ctx), &pb.GetPaymentByBookingRequest{BookingId: bookingID})
	if err != nil {
		return service.PaymentDetails{}, err
	}
	return paymentFromProto(resp.GetPayment())
}

func (c *GRPCClient) RefundPayment(ctx context.Context, input service.RefundPaymentInput) (service.RefundDetails, error) {
	resp, err := c.client.RefundPayment(grpcmeta.ContextWithRequestID(ctx), &pb.RefundPaymentRequest{
		PaymentId: input.PaymentID,
		Amount:    input.Amount,
		Reason:    input.Reason,
	})
	if err != nil {
		return service.RefundDetails{}, err
	}
	refund := resp.GetRefund()
	if refund == nil {
		return service.RefundDetails{}, errors.New("payment-service respondio sin reembolso")
	}
	return service.RefundDetails{
		RefundID:  refund.GetRefundId(),
		PaymentID: refund.GetPaymentId(),
		Amount:    refund.GetAmount(),
		Reason:    refund.GetReason(),
		Status:    service.PaymentStatus(strings.ToUpper(refund.GetStatus())),
	}, nil
}

func paymentFromProto(payment *pb.Payment) (service.PaymentDetails, error) {
	if payment == nil {
		return service.PaymentDetails{}, errors.New("payment-service respondio sin pago")
	}
	return service.PaymentDetails{
		PaymentID: payment.GetPaymentId(),
		BookingID: payment.GetBookingId(),
		UserID:    payment.GetUserId(),
		Amount:    payment.GetAmount(),
		Currency:  payment.GetCurrency(),
		Status:    service.PaymentStatus(strings.ToUpper(payment.GetStatus())),
	}, nil
}

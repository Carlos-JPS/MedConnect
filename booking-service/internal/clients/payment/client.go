package payment

import (
	"context"
	"errors"
	"strings"

	"github.com/MedConnect/booking-service/internal/clients/payment/pb"
	"github.com/MedConnect/booking-service/internal/service"
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

func (c *GRPCClient) GetPaymentStatus(ctx context.Context, paymentID string) (service.PaymentStatus, error) {
	resp, err := c.client.GetPayment(ctx, &pb.GetPaymentRequest{PaymentId: paymentID})
	if err != nil {
		return service.PaymentStatusUnspecified, err
	}
	if resp.GetPayment() == nil {
		return service.PaymentStatusUnspecified, errors.New("payment-service respondio sin pago")
	}

	return service.PaymentStatus(strings.ToUpper(resp.GetPayment().GetStatus())), nil
}

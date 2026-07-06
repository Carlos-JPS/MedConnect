package payment

import (
	"context"
	"errors"

	grpcmeta "github.com/MedConnect/api-gateway/internal/grpc/metadata"
	pb "github.com/sllanoscaro/payment-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.PaymentServiceClient
}

func NewClient(target string) (*Client, error) {
	if target == "" {
		return nil, errors.New("PAYMENT_SERVICE_TARGET no esta configurado")
	}

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewPaymentServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	return c.client.CreatePayment(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	return c.client.ProcessPayment(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) GetPayment(ctx context.Context, req *pb.GetPaymentRequest) (*pb.GetPaymentResponse, error) {
	return c.client.GetPayment(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) GetPaymentsByUser(ctx context.Context, req *pb.GetPaymentsByUserRequest) (*pb.GetPaymentsByUserResponse, error) {
	return c.client.GetPaymentsByUser(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) GetPaymentByBooking(ctx context.Context, req *pb.GetPaymentByBookingRequest) (*pb.GetPaymentByBookingResponse, error) {
	return c.client.GetPaymentByBooking(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) RefundPayment(ctx context.Context, req *pb.RefundPaymentRequest) (*pb.RefundPaymentResponse, error) {
	return c.client.RefundPayment(grpcmeta.ContextWithRequestID(ctx), req)
}

package booking

import (
	"context"
	"errors"

	grpcmeta "github.com/MedConnect/api-gateway/internal/grpc/metadata"
	pb "github.com/MedConnect/booking-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.BookingServiceClient
}

func NewClient(target string) (*Client, error) {
	if target == "" {
		return nil, errors.New("BOOKING_SERVICE_TARGET no esta configurado")
	}

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewBookingServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) CreateBooking(ctx context.Context, req *pb.CreateBookingRequest) (*pb.CreateBookingResponse, error) {
	return c.client.CreateBooking(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) CancelBooking(ctx context.Context, req *pb.CancelBookingRequest) (*pb.CancelBookingResponse, error) {
	return c.client.CancelBooking(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) GetBooking(ctx context.Context, req *pb.GetBookingRequest) (*pb.GetBookingResponse, error) {
	return c.client.GetBooking(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) ListBookingsByPatient(ctx context.Context, req *pb.ListBookingsByPatientRequest) (*pb.ListBookingsByPatientResponse, error) {
	return c.client.ListBookingsByPatient(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) ConfirmBooking(ctx context.Context, req *pb.ConfirmBookingRequest) (*pb.ConfirmBookingResponse, error) {
	return c.client.ConfirmBooking(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) StartBookingSaga(ctx context.Context, req *pb.StartBookingSagaRequest) (*pb.StartBookingSagaResponse, error) {
	return c.client.StartBookingSaga(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) GetBookingSaga(ctx context.Context, req *pb.GetBookingSagaRequest) (*pb.GetBookingSagaResponse, error) {
	return c.client.GetBookingSaga(grpcmeta.ContextWithRequestID(ctx), req)
}

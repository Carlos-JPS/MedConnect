package availability

import (
	"context"
	"errors"

	pb "github.com/Carlos-JPS/medconnect/availability-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.AvailabilityServiceClient
}

func NewClient(target string) (*Client, error) {
	if target == "" {
		return nil, errors.New("AVAILABILITY_SERVICE_TARGET no esta configurado")
	}

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewAvailabilityServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetAvailableSlots(ctx context.Context, req *pb.GetAvailableSlotsRequest) (*pb.GetAvailableSlotsResponse, error) {
	return c.client.GetAvailableSlots(ctx, req)
}

func (c *Client) GetDoctorAgenda(ctx context.Context, req *pb.GetDoctorAgendaRequest) (*pb.GetDoctorAgendaResponse, error) {
	return c.client.GetDoctorAgenda(ctx, req)
}

func (c *Client) HoldSlot(ctx context.Context, req *pb.HoldSlotRequest) (*pb.HoldSlotResponse, error) {
	return c.client.HoldSlot(ctx, req)
}

func (c *Client) ConfirmSlotBooking(ctx context.Context, req *pb.ConfirmSlotBookingRequest) (*pb.ConfirmSlotBookingResponse, error) {
	return c.client.ConfirmSlotBooking(ctx, req)
}

func (c *Client) ReleaseHeldSlot(ctx context.Context, req *pb.ReleaseHeldSlotRequest) (*pb.ReleaseHeldSlotResponse, error) {
	return c.client.ReleaseHeldSlot(ctx, req)
}

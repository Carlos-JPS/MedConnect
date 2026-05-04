package availability

import (
	"context"
	"errors"

	"github.com/MedConnect/booking-service/internal/clients/availability/pb"
	"github.com/MedConnect/booking-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.AvailabilityServiceClient
}

func NewGRPCClient(address string) (*GRPCClient, error) {
	if address == "" {
		return nil, errors.New("direccion de availability-service no configurada")
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		conn:   conn,
		client: pb.NewAvailabilityServiceClient(conn),
	}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCClient) HoldSlot(ctx context.Context, input service.HoldSlotInput) error {
	_, err := c.client.HoldSlot(ctx, &pb.HoldSlotRequest{
		SlotId:    input.SlotID,
		BookingId: input.BookingID,
		HeldUntil: timestamppb.New(input.HeldUntil),
	})
	return err
}

func (c *GRPCClient) ReleaseHeldSlot(ctx context.Context, input service.ReleaseHeldSlotInput) error {
	_, err := c.client.ReleaseHeldSlot(ctx, &pb.ReleaseHeldSlotRequest{
		SlotId:    input.SlotID,
		BookingId: input.BookingID,
	})
	return err
}

func (c *GRPCClient) ConfirmSlotBooking(ctx context.Context, input service.ConfirmSlotBookingInput) error {
	_, err := c.client.ConfirmSlotBooking(ctx, &pb.ConfirmSlotBookingRequest{
		SlotId:    input.SlotID,
		BookingId: input.BookingID,
	})
	return err
}

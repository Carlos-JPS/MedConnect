package auth

import (
	"context"
	"errors"

	grpcmeta "github.com/MedConnect/api-gateway/internal/grpc/metadata"
	pb "github.com/MedConnect/auth-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.AuthServiceClient
}

func NewClient(target string) (*Client, error) {
	if target == "" {
		return nil, errors.New("AUTH_SERVICE_TARGET no esta configurado")
	}

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewAuthServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	return c.client.RegisterUser(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return c.client.Login(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	return c.client.ValidateToken(grpcmeta.ContextWithRequestID(ctx), req)
}

func (c *Client) GetUserById(ctx context.Context, req *pb.GetUserByIdRequest) (*pb.GetUserByIdResponse, error) {
	return c.client.GetUserById(grpcmeta.ContextWithRequestID(ctx), req)
}

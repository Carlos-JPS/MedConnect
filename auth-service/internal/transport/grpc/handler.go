package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/MedConnect/auth-service/internal/service"
	"github.com/MedConnect/auth-service/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	if req.Email == "" || req.Password == "" || req.FullName == "" || req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "missing required fields")
	}

	user, err := h.svc.RegisterUser(ctx, req.Email, req.Password, req.FullName, req.Role)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}
		return nil, status.Errorf(codes.Internal, "internal error: %v", err)
	}

	return &pb.RegisterUserResponse{
		UserId:    user.ID.String(),
		Role:      string(user.Role),
		IsActive:  user.IsActive,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "missing required fields")
	}

	token, user, err := h.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Errorf(codes.Internal, "internal error: %v", err)
	}

	return &pb.LoginResponse{
		AccessToken: token,
		UserId:      user.ID.String(),
		Role:        string(user.Role),
		ExpiresAt:   timestamppb.New(time.Now().UTC().Add(24 * time.Hour)),
	}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "missing token")
	}

	user, err := h.svc.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		if errors.Is(err, service.ErrTokenExpired) {
			return nil, status.Error(codes.Unauthenticated, "token expired")
		}
		if errors.Is(err, service.ErrTokenInvalid) {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		return nil, status.Errorf(codes.Internal, "internal error: %v", err)
	}

	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: user.ID.String(),
		Role:   string(user.Role),
	}, nil
}

func (h *AuthHandler) GetUserById(ctx context.Context, req *pb.GetUserByIdRequest) (*pb.GetUserByIdResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing user id")
	}

	userId, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	user, err := h.svc.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "internal error: %v", err)
	}

	return &pb.GetUserByIdResponse{
		UserId:    user.ID.String(),
		FullName:  user.FullName,
		Email:     user.Email,
		Role:      string(user.Role),
		IsActive:  user.IsActive,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}, nil
}

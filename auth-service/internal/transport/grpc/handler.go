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

// AuthHandler implementa la interfaz gRPC generada para el servicio de autenticación
type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	svc service.AuthService
}

// NewAuthHandler crea una nueva instancia del handler gRPC
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// RegisterUser maneja la petición gRPC para registrar un nuevo usuario
func (h *AuthHandler) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	// Validación básica de campos obligatorios
	if req.Email == "" || req.Password == "" || req.FullName == "" || req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "faltan campos obligatorios")
	}

	user, err := h.svc.RegisterUser(ctx, req.Email, req.Password, req.FullName, req.Role)
	if err != nil {
		// Mapeo de errores de negocio a códigos de estado gRPC
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "el correo electrónico ya está registrado")
		}
		return nil, status.Errorf(codes.Internal, "error interno: %v", err)
	}

	// Conversión de la entidad de dominio a la respuesta protobuf
	return &pb.RegisterUserResponse{
		UserId:    user.ID.String(),
		Role:      string(user.Role),
		IsActive:  user.IsActive,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}, nil
}

// Login maneja la petición gRPC para autenticar a un usuario
func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "faltan credenciales")
	}

	token, user, err := h.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "credenciales inválidas")
		}
		return nil, status.Errorf(codes.Internal, "error interno: %v", err)
	}

	return &pb.LoginResponse{
		AccessToken: token,
		UserId:      user.ID.String(),
		Role:        string(user.Role),
		ExpiresAt:   timestamppb.New(time.Now().UTC().Add(24 * time.Hour)),
	}, nil
}

// ValidateToken maneja la petición gRPC para verificar la validez de un token
func (h *AuthHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "falta el token de acceso")
	}

	user, err := h.svc.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		if errors.Is(err, service.ErrTokenExpired) {
			return nil, status.Error(codes.Unauthenticated, "el token ha expirado")
		}
		if errors.Is(err, service.ErrTokenInvalid) {
			return nil, status.Error(codes.Unauthenticated, "token inválido")
		}
		return nil, status.Errorf(codes.Internal, "error interno: %v", err)
	}

	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: user.ID.String(),
		Role:   string(user.Role),
	}, nil
}

// GetUserById maneja la petición gRPC para obtener detalles de un usuario por su ID
func (h *AuthHandler) GetUserById(ctx context.Context, req *pb.GetUserByIdRequest) (*pb.GetUserByIdResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "falta el ID de usuario")
	}

	userId, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "formato de ID de usuario inválido")
	}

	user, err := h.svc.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "usuario no encontrado")
		}
		return nil, status.Errorf(codes.Internal, "error interno: %v", err)
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

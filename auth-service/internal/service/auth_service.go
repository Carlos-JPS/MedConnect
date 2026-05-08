package service

import (
	"context"
	"errors"
	"time"

	"github.com/MedConnect/auth-service/internal/repository/postgres"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenInvalid       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrUserNotFound       = postgres.ErrUserNotFound
	ErrEmailAlreadyExists = postgres.ErrEmailAlreadyExists
)

// UserRepository defines the interface that the auth service requires from the database layer.
type UserRepository interface {
	CreateUser(ctx context.Context, user *postgres.User) error
	GetUserByEmail(ctx context.Context, email string) (*postgres.User, error)
	GetUserById(ctx context.Context, id uuid.UUID) (*postgres.User, error)
}

// AuthService defines the business logic interface
type AuthService interface {
	RegisterUser(ctx context.Context, email, password, fullName, role string) (*postgres.User, error)
	Login(ctx context.Context, email, password string) (string, *postgres.User, error)
	ValidateToken(ctx context.Context, tokenStr string) (*postgres.User, error)
	GetUserById(ctx context.Context, id uuid.UUID) (*postgres.User, error)
}

type authService struct {
	repo      UserRepository
	jwtSecret string
}

func NewAuthService(repo UserRepository, jwtSecret string) AuthService {
	return &authService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func (s *authService) RegisterUser(ctx context.Context, email, password, fullName, role string) (*postgres.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userRole := postgres.UserRole(role)

	user := &postgres.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		FullName:     fullName,
		Role:         userRole,
		CreatedAt:    time.Now().UTC(),
		IsActive:     true,
	}

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, *postgres.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, postgres.ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"role":    string(user.Role),
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", nil, err
	}

	return tokenString, user, nil
}

func (s *authService) ValidateToken(ctx context.Context, tokenStr string) (*postgres.User, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return nil, ErrTokenExpired
			}
		}

		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			return nil, ErrTokenInvalid
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, ErrTokenInvalid
		}

		user, err := s.repo.GetUserById(ctx, userID)
		if err != nil {
			return nil, err
		}

		return user, nil
	}

	return nil, ErrTokenInvalid
}

func (s *authService) GetUserById(ctx context.Context, id uuid.UUID) (*postgres.User, error) {
	return s.repo.GetUserById(ctx, id)
}

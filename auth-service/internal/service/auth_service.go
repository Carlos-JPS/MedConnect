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

// Errores de lógica de negocio
var (
	ErrInvalidCredentials = errors.New("credenciales inválidas")
	ErrTokenInvalid       = errors.New("token inválido")
	ErrTokenExpired       = errors.New("token expirado")
	ErrUserNotFound       = postgres.ErrUserNotFound
	ErrEmailAlreadyExists = postgres.ErrEmailAlreadyExists
)

// UserRepository define la interfaz que el servicio requiere de la capa de persistencia
type UserRepository interface {
	CreateUser(ctx context.Context, user *postgres.User) error
	GetUserByEmail(ctx context.Context, email string) (*postgres.User, error)
	GetUserById(ctx context.Context, id uuid.UUID) (*postgres.User, error)
}

// AuthService define la interfaz con los casos de uso de autenticación
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

// NewAuthService crea una nueva instancia del servicio de autenticación
func NewAuthService(repo UserRepository, jwtSecret string) AuthService {
	return &authService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

// RegisterUser maneja el registro de nuevos usuarios aplicando hashing a la contraseña
func (s *authService) RegisterUser(ctx context.Context, email, password, fullName, role string) (*postgres.User, error) {
	// Generar hash seguro de la contraseña usando bcrypt
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

	// Guardar en la base de datos a través del repositorio
	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Login verifica credenciales y genera un token JWT si son correctas
func (s *authService) Login(ctx context.Context, email, password string) (string, *postgres.User, error) {
	// Buscar usuario por email
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, postgres.ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	// Comparar la contraseña ingresada con el hash guardado
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	// Generar token JWT con claims (usuario, rol y expiración)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"role":    string(user.Role),
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	// Firmar el token con el secreto del servidor
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", nil, err
	}

	return tokenString, user, nil
}

// ValidateToken comprueba la validez y expiración de un token JWT
func (s *authService) ValidateToken(ctx context.Context, tokenStr string) (*postgres.User, error) {
	// Parsear y validar firma del token
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
		// Verificar expiración manualmente por seguridad extra
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

		// Recuperar datos actualizados del usuario desde la BD
		user, err := s.repo.GetUserById(ctx, userID)
		if err != nil {
			return nil, err
		}

		return user, nil
	}

	return nil, ErrTokenInvalid
}

// GetUserById obtiene un usuario por su ID (UUID)
func (s *authService) GetUserById(ctx context.Context, id uuid.UUID) (*postgres.User, error) {
	return s.repo.GetUserById(ctx, id)
}

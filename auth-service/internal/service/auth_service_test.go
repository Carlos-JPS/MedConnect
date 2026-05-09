package service

import (
	"context"
	"testing"

	"github.com/MedConnect/auth-service/internal/repository/postgres"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository is a simple in-memory mock for the UserRepository interface
type mockUserRepository struct {
	usersByEmail map[string]*postgres.User
	usersByID    map[uuid.UUID]*postgres.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		usersByEmail: make(map[string]*postgres.User),
		usersByID:    make(map[uuid.UUID]*postgres.User),
	}
}

func (m *mockUserRepository) CreateUser(ctx context.Context, user *postgres.User) error {
	if _, exists := m.usersByEmail[user.Email]; exists {
		return postgres.ErrEmailAlreadyExists
	}
	m.usersByEmail[user.Email] = user
	m.usersByID[user.ID] = user
	return nil
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (*postgres.User, error) {
	user, exists := m.usersByEmail[email]
	if !exists {
		return nil, postgres.ErrUserNotFound
	}
	return user, nil
}

func (m *mockUserRepository) GetUserById(ctx context.Context, id uuid.UUID) (*postgres.User, error) {
	user, exists := m.usersByID[id]
	if !exists {
		return nil, postgres.ErrUserNotFound
	}
	return user, nil
}

func TestAuthService_RegisterAndLogin(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewAuthService(repo, "test-secret-key")
	ctx := context.Background()

	email := "test@test.com"
	password := "securepassword123"
	fullName := "John Doe"
	role := "PATIENT"

	// 1. Test RegisterUser
	user, err := svc.RegisterUser(ctx, email, password, fullName, role)
	if err != nil {
		t.Fatalf("expected no error on register, got %v", err)
	}
	if user.Email != email {
		t.Errorf("expected email %s, got %s", email, user.Email)
	}

	// Verify that password is hashed correctly
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		t.Errorf("expected hashed password to match plain password, got error: %v", err)
	}

	// 2. Test Duplicate Register
	_, err = svc.RegisterUser(ctx, email, password, fullName, role)
	if err != postgres.ErrEmailAlreadyExists {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}

	// 3. Test Login Success
	token, loginUser, err := svc.Login(ctx, email, password)
	if err != nil {
		t.Fatalf("expected no error on login, got %v", err)
	}
	if token == "" {
		t.Errorf("expected a valid JWT token, got empty string")
	}
	if loginUser.ID != user.ID {
		t.Errorf("expected logged in user ID to match registered user ID")
	}

	// 4. Test Login with Wrong Password
	_, _, err = svc.Login(ctx, email, "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	// 5. Test Validate Token
	validatedUser, err := svc.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("expected no error on token validation, got %v", err)
	}
	if validatedUser.ID != user.ID {
		t.Errorf("expected validated user ID to match original user ID")
	}
}

func TestAuthService_ValidateInvalidToken(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewAuthService(repo, "test-secret-key")
	ctx := context.Background()

	_, err := svc.ValidateToken(ctx, "invalid-token-string")
	if err != ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

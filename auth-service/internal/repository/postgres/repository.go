package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Errores comunes de repositorio
var (
	ErrUserNotFound       = errors.New("usuario no encontrado")
	ErrEmailAlreadyExists = errors.New("el correo electrónico ya existe")
)

// Repository maneja las operaciones de persistencia en PostgreSQL para usuarios
type Repository struct {
	db *sql.DB
}

// NewRepository crea una nueva instancia del repositorio
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser inserta un nuevo usuario en la base de datos
func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, password_hash, full_name, role, created_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	// Timeout de 5 segundos para la operación
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Role,
		user.CreatedAt,
		user.IsActive,
	)

	if err != nil {
		// Verificar si el error es por violación de unicidad (email duplicado)
		if isUniqueViolation(err) {
			return ErrEmailAlreadyExists
		}
		return err
	}

	return nil
}

// GetUserByEmail busca un usuario por su dirección de correo electrónico
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name, role, created_at, is_active
		FROM users
		WHERE email = $1
	`
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, query, email)

	var u User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Role,
		&u.CreatedAt,
		&u.IsActive,
	)

	if err != nil {
		// Manejar el caso donde no se encuentra ninguna fila
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

// GetUserById busca un usuario por su identificador único (UUID)
func (r *Repository) GetUserById(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name, role, created_at, is_active
		FROM users
		WHERE id = $1
	`
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, query, id)

	var u User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Role,
		&u.CreatedAt,
		&u.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

// isUniqueViolation verifica si un error de Postgres es una violación de unicidad (código 23505)
func isUniqueViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	if ok && pqErr.Code == "23505" {
		return true
	}
	return false
}

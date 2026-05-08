package postgres

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdmin   UserRole = "ADMIN"
	RoleDoctor  UserRole = "DOCTOR"
	RolePatient UserRole = "PATIENT"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FullName     string
	Role         UserRole
	CreatedAt    time.Time
	IsActive     bool
}

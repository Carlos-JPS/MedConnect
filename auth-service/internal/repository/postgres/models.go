package postgres

import (
	"time"

	"github.com/google/uuid"
)

// UserRole define los roles posibles en el sistema
type UserRole string

const (
	RoleAdmin   UserRole = "ADMIN"   // Administrador del sistema
	RoleDoctor  UserRole = "DOCTOR"  // Personal médico
	RolePatient UserRole = "PATIENT" // Paciente
)

// User representa la estructura de un usuario en la base de datos
type User struct {
	ID           uuid.UUID // Identificador único (UUID)
	Email        string    // Correo electrónico (único)
	PasswordHash string    // Hash de la contraseña (nunca texto plano)
	FullName     string    // Nombre completo del usuario
	Role         UserRole  // Rol asignado
	CreatedAt    time.Time // Fecha de creación
	IsActive     bool      // Estado de la cuenta
}

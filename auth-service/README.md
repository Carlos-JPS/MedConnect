# Auth Service

Servicio de autenticación y gestión de usuarios para MedConnect. Implementado en Go con gRPC.

## Responsabilidad

Este servicio se encarga de:

- **Registrar usuarios** nuevos (pacientes, doctores, administradores).
- **Autenticar usuarios** mediante email y contraseña, devolviendo un JWT.
- **Validar tokens** JWT para que otros servicios verifiquen la identidad del usuario.
- **Consultar usuarios** por ID para obtener datos de perfil.

## Contrato gRPC (Protobuf)

El archivo [`pb/auth.proto`](pb/auth.proto) define 4 métodos RPC:

| Método | Recibe | Devuelve |
|---|---|---|
| `RegisterUser` | email, password, full_name, role | user_id, role, is_active, created_at |
| `Login` | email, password | access_token, user_id, role, expires_at |
| `ValidateToken` | access_token | valid, user_id, role, expires_at |
| `GetUserById` | user_id | user_id, full_name, email, role, is_active, created_at |

> **Nota de seguridad:** Ninguna respuesta expone `password_hash`. El campo `password` solo viaja en los requests de registro y login.

## Modelo de Datos

La tabla `users` se crea con la migración en [`migrations/000001_create_users_table.up.sql`](migrations/000001_create_users_table.up.sql):

```sql
CREATE TYPE user_role AS ENUM ('ADMIN', 'DOCTOR', 'PATIENT');

CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    role user_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);
```

## Estructura del Proyecto

```
auth-service/
├── cmd/auth-service/          # Punto de entrada (main.go) — pendiente
│   └── main.go
├── internal/
│   ├── config/                # Carga de variables de entorno — pendiente
│   ├── service/               # Lógica de negocio (bcrypt, JWT) ✅
│   │   └── auth_service.go    # Implementación de hashing, verificación de clave y emisión de JWT
│   ├── repository/postgres/   # Queries SQL contra PostgreSQL ✅
│   │   ├── models.go          # Struct User y constantes de roles
│   │   └── repository.go     # CreateUser, GetUserByEmail, GetUserById
│   └── transport/grpc/        # Servidor gRPC (handler) — pendiente
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   └── 000001_create_users_table.down.sql
├── pb/
│   ├── auth.proto             # Contrato Protobuf
│   ├── auth.pb.go             # Código generado (mensajes)
│   └── auth_grpc.pb.go        # Código generado (stubs gRPC)
├── go.mod
└── README.md
```

## Variables de Entorno

| Variable | Descripción | Ejemplo |
|---|---|---|
| `AUTH_SERVICE_HOST` | Host de escucha gRPC | `0.0.0.0` |
| `AUTH_SERVICE_PORT` | Puerto gRPC | `50051` |
| `AUTH_DB_DSN` | DSN de conexión a PostgreSQL | `postgres://auth:auth_password@auth_db:5432/auth_db?sslmode=disable` |
| `JWT_SECRET` | Clave secreta para firmar tokens JWT | `mi-clave-secreta-segura` |

## Capa Repository

La capa de persistencia (`internal/repository/postgres/`) implementa el acceso directo a PostgreSQL:

- **`models.go`**: Define el struct `User` (con campos `ID`, `Email`, `PasswordHash`, `FullName`, `Role`, `CreatedAt`, `IsActive`) y las constantes de rol (`ADMIN`, `DOCTOR`, `PATIENT`).
- **`repository.go`**: Implementa 3 métodos:
  - `CreateUser(ctx, user)` — Inserta un usuario; detecta emails duplicados devolviendo `ErrEmailAlreadyExists`.
  - `GetUserByEmail(ctx, email)` — Busca por email; devuelve `ErrUserNotFound` si no existe.
  - `GetUserById(ctx, id)` — Busca por UUID; devuelve `ErrUserNotFound` si no existe.

> **Resiliencia:** Cada query usa `context.WithTimeout` de 5 segundos para evitar bloqueos con la base de datos. Los errores de constraint (`UNIQUE` en email) se traducen a errores de dominio legibles.

## Capa Service

La capa de negocio (`internal/service/`) implementa la lógica fundamental y desacopla la persistencia usando la interfaz `UserRepository`:

- **`auth_service.go`**: Expone la interfaz `AuthService` e implementa:
  - `RegisterUser` — Hashea la contraseña usando `bcrypt` (DefaultCost) e invoca la creación del usuario en el repositorio.
  - `Login` — Recupera el hash por email, lo compara con `bcrypt.CompareHashAndPassword` y, si es correcto, emite un token JWT firmado con `HS256` y válido por 24 horas.
  - `ValidateToken` — Verifica la firma y expiración del JWT (`jwt-go v5`), parsea los claims y valida que el usuario siga existiendo.
  - `GetUserById` — Capa passthrough para recuperar el perfil del usuario validado.

## Estado de Implementación

- [x] Contrato Protobuf (`auth.proto`) y código generado.
- [x] Migración SQL para la tabla `users`.
- [x] Capa repository (persistencia PostgreSQL).
- [x] Capa service (lógica de negocio, bcrypt, JWT).
- [ ] Capa transport (servidor gRPC).
- [ ] Punto de entrada (`cmd/auth-service/main.go`).
- [ ] Dockerfile.
- [ ] Integración con Docker Compose y API Gateway.

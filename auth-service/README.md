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
│   ├── service/               # Lógica de negocio (bcrypt, JWT) — pendiente
│   ├── repository/postgres/   # Queries SQL contra PostgreSQL — pendiente
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

## Estado de Implementación

- [x] Contrato Protobuf (`auth.proto`) y código generado.
- [x] Migración SQL para la tabla `users`.
- [ ] Capa repository (persistencia PostgreSQL).
- [ ] Capa service (lógica de negocio, bcrypt, JWT).
- [ ] Capa transport (servidor gRPC).
- [ ] Punto de entrada (`cmd/auth-service/main.go`).
- [ ] Dockerfile.
- [ ] Integración con Docker Compose y API Gateway.

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
├── cmd/auth-service/          # Punto de entrada (main.go) ✅
│   └── main.go                # Inyección de dependencias y servidor gRPC
├── internal/
│   ├── service/               # Lógica de negocio (bcrypt, JWT) ✅
│   │   └── auth_service.go    # Hashing, verificación de clave y emisión de JWT
│   ├── repository/postgres/   # Queries SQL contra PostgreSQL ✅
│   │   ├── models.go          # Struct User y constantes de roles
│   │   └── repository.go      # CreateUser, GetUserByEmail, GetUserById
│   └── transport/grpc/        # Servidor gRPC (handler) ✅
│       └── handler.go         # Mapeo de pb a service y códigos de error gRPC
├── migrations/                # Scripts SQL de base de datos
├── pb/                        # Contratos Protobuf y código generado
├── go.mod                     # Dependencias de Go
└── README.md                  # Documentación del servicio
```

## Variables de Entorno

| Variable | Descripción | Ejemplo |
|---|---|---|
| `AUTH_SERVICE_HOST` | Host de escucha del servicio gRPC | `0.0.0.0` |
| `AUTH_SERVICE_PORT` | Puerto del servicio gRPC | `50051` |
| `AUTH_DB_NAME` | Nombre de la base de datos | `auth_db` |
| `AUTH_DB_USER` | Usuario de PostgreSQL | `auth` |
| `AUTH_DB_PASSWORD` | Contraseña de PostgreSQL | `auth_password` |
| `AUTH_DB_DSN` | DSN completo de conexión (usado por la app) | `postgres://auth:auth_password@auth_db:5432/auth_db?sslmode=disable` |
| `JWT_SECRET` | Clave secreta para firmar y validar tokens JWT | `medconnect-jwt-secret-change-me` |

## Integración con API Gateway

El servicio está integrado en el **API Gateway** unificado de MedConnect. Las peticiones REST externas se traducen automáticamente a llamadas gRPC hacia este servicio.

### Endpoints REST (vía Gateway)

- **Registro**: `POST http://localhost:8080/auth/register`
- **Login**: `POST http://localhost:8080/auth/login`

## Pruebas

### Pruebas Unitarias
Ejecuta la lógica de negocio y validación de tokens en memoria:
```bash
go test -v ./internal/service/...
```

### Pruebas Funcionales (Insomnia)
Usa el archivo `Insomnia_2024-XX-XX.yaml` en la raíz del repositorio para probar el flujo completo:
1. Asegúrate de que el sistema esté arriba: `docker compose up --build -d`
2. Usa la carpeta **Authentication** en Insomnia.
3. El login devolverá un `access_token` que podrás usar para otros servicios protegidos.

## Estado de Implementación

- [x] Contrato Protobuf (`auth.proto`) y código generado.
- [x] Migración SQL para la tabla `users`.
- [x] Capa repository (persistencia PostgreSQL).
- [x] Capa service (lógica de negocio, bcrypt, JWT).
- [x] Capa transport (servidor gRPC).
- [x] Punto de entrada (`cmd/auth-service/main.go`).
- [x] Dockerfile y despliegue en contenedores.
- [x] Integración con API Gateway y rutas REST.
- [x] Comentarios en español en todo el código fuente.

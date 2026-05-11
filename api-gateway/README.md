# API Gateway - MedConnect

El **API Gateway** es el punto de entrada unificado para el ecosistema de MedConnect. Actúa como un **BFF (Backend For Frontends)** que expone una interfaz RESTful/HTTP al mundo exterior y coordina la comunicación interna con múltiples microservicios mediante gRPC.

## Arquitectura Técnica

- **Interface:** HTTP/1.1 REST
- **Internal Communication:** gRPC (Protocol Buffers)
- **Framework:** Go `net/http` estándar (sin frameworks externos para máxima ligereza)
- **Patrón:** Proxy Reverso + Transformación de Datos + Agregación de Identidad.

## Características Principales

### 1. Seguridad e Identidad
- **Middleware de Autenticación:** Valida tokens JWT delegando la verificación al `auth-service`.
- **Inyección de Identidad:** Extrae el `user_id` del token y lo inyecta en el contexto interno, evitando que el cliente tenga que enviar IDs manuales en el JSON.
- **Protección IDOR:** Previene que un usuario manipule recursos de terceros al forzar la identidad verificada por el Gateway.

### 2. Gestión de Errores
- **Mapeo gRPC a HTTP:** Traduce códigos de estado gRPC (como `NOT_FOUND` o `UNAVAILABLE`) a sus equivalentes semánticos en HTTP (`404`, `503`, etc.).

## Referencia de Endpoints

| Módulo | Método | Ruta | Protegido | Descripción |
| :--- | :--- | :--- | :---: | :--- |
| **Auth** | POST | `/auth/register` | No | Registro de nuevos usuarios |
| **Auth** | POST | `/auth/login` | No | Login y obtención de JWT |
| **Availability** | GET | `/availability/slots` | No | Consulta de slots disponibles |
| **Availability** | POST | `/availability/hold` | Sí | Bloqueo temporal de un slot |
| **Bookings** | POST | `/bookings` | **Sí** | Creación de reserva (ID de paciente automático) |
| **Bookings** | GET | `/bookings` | **Sí** | Listar reservas del usuario logeado |
| **Payments** | POST | `/payments` | **Sí** | Iniciar proceso de pago |

## Configuración (Variables de Entorno)

El servicio se configura mediante las siguientes variables (con valores por defecto funcionales):

| Variable | Valor por Defecto | Descripción |
| :--- | :--- | :--- |
| `API_GATEWAY_HOST` | `0.0.0.0` | Host de escucha |
| `API_GATEWAY_PORT` | `8080` | Puerto de escucha |
| `AUTH_SERVICE_TARGET` | `auth-service:50051` | Dirección del microservicio de Auth |
| `BOOKING_SERVICE_TARGET` | `booking-service:50051` | Dirección del microservicio de Reservas |
| `PAYMENT_SERVICE_TARGET` | `payment-service:50051` | Dirección del microservicio de Pagos |
| `API_GATEWAY_REQUEST_TIMEOUT` | `5s` | Timeout global para peticiones |

## Stack Tecnológico
- **Lenguaje:** Go 1.22+
- **Comunicación:** gRPC / Protocol Buffers
- **Contenerización:** Docker / Docker Compose

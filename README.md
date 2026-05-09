# MedConnect

Sistema de gestion clinica hospitalaria basado en microservicios. Para esta entrega, el flujo implementado y defendible del repositorio se concentra en reservas medicas y pagos usando:

`frontend -> api-gateway HTTP -> booking-service/availability-service/payment-service gRPC -> PostgreSQL`

## Arquitectura Actual

- **frontend**: cliente web React para demostrar el flujo de reservas desde navegador.
- **api-gateway**: unica entrada HTTP externa. Expone endpoints REST de autenticación, reservas y pagos.
- **auth-service**: servicio gRPC de autenticación y usuarios. Maneja registro, login y validación de tokens JWT.
- **booking-service**: servicio gRPC de reservas. Persiste citas y eventos en PostgreSQL.
- **availability-service**: servicio gRPC de disponibilidad. Persiste agendas y slots médicos en PostgreSQL.
- **payment-service**: servicio gRPC de pagos. Persiste pagos, transacciones y reembolsos en PostgreSQL.
- **auth_db**: base PostgreSQL de usuarios.
- **booking_db**: base PostgreSQL de reservas.
- **availability-db**: base PostgreSQL de disponibilidad.
- **payments-db**: base PostgreSQL de pagos.

`booking-service` integra `availability-service` por gRPC para bloquear, confirmar y liberar slots. La demo depende de que `availability-db` tenga cargados los slots definidos en `availability-service/db/002_seed_demo_slots.sql`.
La confirmación de una reserva depende además de un pago real creado para el mismo `booking_id` y procesado en `payment-service`.

## Puertos

- Frontend: `http://localhost:5173`
- API Gateway: `http://localhost:8080`
- `auth-service`: gRPC interno `50051`
- `booking-service`: gRPC interno `50051`
- `availability-service`: gRPC interno `50051`
- `payment-service`: gRPC interno `50051`

Solo el frontend y el API Gateway se exponen al host. Los servicios internos se comunican por la red Docker `medconnect_internal`.

## Requisitos

- Docker
- Docker Compose V2

No es necesario tener Go o Node instalados localmente para levantar la demo con Docker.

## Configuracion

Usa `.env.example` como referencia:

```bash
cp .env.example .env
```

Variables principales:

```bash
BOOKING_DB_DSN=postgres://booking:booking_password@booking_db:5432/booking_db?sslmode=disable
AVAILABILITY_SERVICE_TARGET=availability-service:50051
AVAILABILITY_SERVICE_PORT=50051
PAYMENT_SERVICE_TARGET=payment-service:50051
BOOKING_SERVICE_TARGET=booking-service:50051
API_GATEWAY_PORT=8080
FRONTEND_PORT=5173
VITE_API_BASE_URL=/api
```

El frontend usa `/api` para llamar al gateway por el mismo origen del navegador. En Docker, Nginx reenvia `/api/*` hacia `api-gateway:8080`; en desarrollo local, Vite hace el mismo proxy hacia `http://localhost:8080`.

## Levantar el Sistema

```bash
docker compose up --build -d
```

Ver contenedores:

```bash
docker compose ps
```

Ver logs:

```bash
docker compose logs -f api-gateway booking-service availability-service payment-service
```

Detener:

```bash
docker compose down
```

Detener y borrar volumenes de datos:

```bash
docker compose down -v
```

### Seed de Availability

En una base nueva, Docker ejecuta automaticamente `availability-service/db/init.sql` y `availability-service/db/002_seed_demo_slots.sql`. Si `availability-db` ya existia antes de agregar el seed, aplicalo manualmente:

```bash
docker exec -i availability-db psql -U postgres -d availability_db < availability-service/db/002_seed_demo_slots.sql
```

Para comprobar los slots demo:

```bash
docker exec availability-db psql -U postgres -d availability_db \
  -c "SELECT id, status, start_time FROM availability_slots ORDER BY start_time;"
```

### Migraciones de Booking en Bases Existentes

En una base nueva, Docker ejecuta automaticamente las migraciones montadas en `booking-service/migrations`. Si `booking_db` ya existia antes de agregar la migracion que permite reutilizar slots cancelados, aplicala manualmente:

```bash
docker exec -i medconnect-booking_db-1 psql -U booking -d booking_db < booking-service/migrations/000002_allow_rebooking_cancelled_slots.up.sql
```

Para comprobar el indice activo por slot:

```bash
docker exec medconnect-booking_db-1 psql -U booking -d booking_db \
  -c "SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'appointments';"
```

## Endpoints de Autenticación

### Registro de Usuario

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "paciente@test.com",
    "password": "password123",
    "full_name": "Juan Perez",
    "role": "PATIENT"
  }'
```

### Inicio de Sesión

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "paciente@test.com",
    "password": "password123"
  }'
```

La respuesta incluye un `access_token` (JWT) válido por 24 horas.

## Endpoints de Booking

Todas las rutas entran por `api-gateway`.

### Crear Reserva

```bash
curl -X POST http://localhost:8080/bookings \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "46bd4a6f-6a4d-4e81-ae7c-c9d7ac05b235",
    "doctor_id": "7e0d2ab1-164e-4a28-8b95-f24293dd0e91",
    "slot_id": "0f5c2b6a-1a87-4b7e-ae2c-37ef2f9f1c21",
    "notes": "Control creado desde demo"
  }'
```

Respuesta esperada si `availability-service` esta disponible y el slot esta en estado `available`:

```json
{
  "booking_id": "uuid",
  "status": "PENDING_PAYMENT",
  "reserved_until": "2026-05-07T12:00:00Z"
}
```

Si el slot no existe o ya fue usado, el gateway devuelve un error controlado proveniente de `availability-service` indicando que el slot no esta en estado `available`.

### Listar Reservas de Paciente

```bash
curl "http://localhost:8080/bookings?patient_id=46bd4a6f-6a4d-4e81-ae7c-c9d7ac05b235"
```

Filtro por estado:

```bash
curl "http://localhost:8080/bookings?patient_id=46bd4a6f-6a4d-4e81-ae7c-c9d7ac05b235&status=PENDING_PAYMENT"
```

### Obtener Detalle de Reserva

```bash
curl http://localhost:8080/bookings/{booking_id}
```

La respuesta incluye la reserva y sus eventos persistidos.

### Confirmar Reserva

Primero debe existir un pago aprobado o completado en `payment-service` para el mismo `booking_id` de la reserva. Un pago de otra reserva o un pago sin `booking_id` asociado es rechazado.

```bash
curl -X POST http://localhost:8080/bookings/{booking_id}/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": "{payment_id}"
  }'
```

`booking-service` valida por gRPC que el pago exista, tenga estado `APPROVED` o `COMPLETED`, y pertenezca a la misma reserva antes de solicitar la confirmacion del slot a `availability-service`.

### Cancelar Reserva

En esta entrega, la cancelacion soportada aplica a reservas `PENDING_PAYMENT`. Las reservas `CONFIRMED` quedan fuera del flujo porque requeririan una anulacion de slot confirmado y posible reembolso.

```bash
curl -X PATCH http://localhost:8080/bookings/{booking_id}/cancel \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Paciente solicita reagendar"
  }'
```

`booking-service` libera el slot retenido por gRPC contra `availability-service` antes de persistir la cancelacion.

## Endpoints de Payment

### Crear Pago

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{
    "booking_id": "{booking_id}",
    "user_id": "46bd4a6f-6a4d-4e81-ae7c-c9d7ac05b235",
    "amount": 15000,
    "currency": "CLP"
  }'
```

### Procesar Pago

```bash
curl -X POST http://localhost:8080/payments/{payment_id}/process \
  -H "Content-Type: application/json" \
  -d '{
    "payment_method_id": "method-demo"
  }'
```

### Consultar Pago

```bash
curl http://localhost:8080/payments/{payment_id}
```

## Demo Recomendada

1. Levantar el sistema con `docker compose up --build -d`.
2. Abrir `http://localhost:5173`.
3. Seleccionar un slot demo en el frontend.
4. Crear una reserva desde el frontend o Insomnia.
5. Listar reservas del paciente.
6. Consultar detalle y eventos de la reserva.
7. Crear un pago desde Insomnia usando el `booking_id` devuelto al crear la reserva.
8. Procesar ese pago y copiar el `payment_id` retornado.
9. Confirmar la reserva con ese `payment_id`.
10. Para probar cancelacion, crea otra reserva y cancelala mientras sigue `PENDING_PAYMENT`.

Si repites la demo con el mismo slot, `availability-service` puede rechazar la reserva porque el slot ya quedo `held` o `booked`. Usa otro slot demo, cancela la reserva para liberar el slot o reinicia los volumenes si necesitas volver al estado inicial.

## Pruebas con Insomnia

El archivo `Insomnia_2026-05-08.yaml` incluye carpetas para:

- `Bookings`
- `Availability`
- `Payments`
- `Authentication`

Importa el archivo en Insomnia y usa el ambiente base incluido. Las variables iniciales son `base_url = http://localhost:8080`, `patient_id`, `doctor_id`, `slot_id`, `booking_id`, `payment_id` y `payment_method_id`. Despues de crear una reserva o pago, copia los IDs retornados a `booking_id` y `payment_id`.

## Verificacion Tecnica

Comandos utiles:

```bash
docker compose config
docker compose build api-gateway booking-service availability-service payment-service frontend
docker run --rm -v "$PWD":/workspace -w /workspace/booking-service golang:1.26-alpine go test ./...
docker run --rm -v "$PWD":/workspace -w /workspace/api-gateway golang:1.26-alpine go test ./...
docker run --rm -v "$PWD":/workspace -w /workspace/availability-service golang:1.26-alpine go test ./...
docker run --rm -v "$PWD/frontend":/app -w /app node:22-alpine sh -c "npm install && npm test -- --run"
```

## Estructura Relevante

```text
MedConnect/
├── api-gateway/                    # Gateway HTTP unificado
├── auth-service/                   # Servicio gRPC de autenticación
├── availability-service/           # Servicio gRPC de disponibilidad
├── booking-service/                # Servicio gRPC de reservas
├── frontend/                       # Cliente web React
├── payment-service/                # Servicio gRPC de pagos
├── docker-compose.yml
├── .env.example
└── Insomnia_2026-05-08.yaml
```

## Estado de Entrega

Implementado:

- Contrato gRPC de reservas.
- Persistencia de reservas y eventos.
- API Gateway para reservas y pagos.
- Frontend minimo para flujo paciente.
- Docker Compose con bases de autenticacion, disponibilidad, reservas y pagos.
- Cliente de disponibilidad de `booking-service` para `HoldSlot`, `ReleaseHeldSlot` y `ConfirmSlotBooking`.
- Cliente de pagos de `booking-service` alineado con el contrato real de `payment-service`, incluyendo validacion de pertenencia del pago a la reserva.

Limitaciones actuales:

- No hay orquestacion transaccional distribuida entre reservas, disponibilidad y pagos; cada servicio persiste en su propia base y los errores se manejan con estados y respuestas controladas.
- Las migraciones se ejecutan por inicializacion de contenedores PostgreSQL. En volumenes existentes hay que aplicar manualmente las nuevas migraciones indicadas en este README.

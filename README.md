# MedConnect

Sistema de gestion clinica hospitalaria basado en microservicios. Para esta entrega, el flujo implementado y defendible del repositorio se concentra en reservas medicas y pagos usando:

`frontend -> api-gateway HTTP -> booking-service/payment-service gRPC -> PostgreSQL`

## Arquitectura Actual

- **frontend**: cliente web React para demostrar el flujo de reservas desde navegador.
- **api-gateway**: unica entrada HTTP externa. Expone endpoints REST de reservas y pagos.
- **booking-service**: servicio gRPC de reservas. Persiste citas y eventos en PostgreSQL.
- **payment-service**: servicio gRPC de pagos. Persiste pagos, transacciones y reembolsos en PostgreSQL.
- **booking_db**: base PostgreSQL de reservas.
- **payments-db**: base PostgreSQL de pagos.

Servicios como `availability-service` todavia no estan completos en este repositorio. `booking-service` ya tiene cliente gRPC y manejo de errores para esa dependencia, pero los casos `CreateBooking`, `CancelBooking` y `ConfirmBooking` requieren que availability implemente `HoldSlot`, `ReleaseHeldSlot` y `ConfirmSlotBooking` para una demo end-to-end completa.

## Puertos

- Frontend: `http://localhost:5173`
- API Gateway: `http://localhost:8080`
- `booking-service`: gRPC interno `50051`
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
PAYMENT_SERVICE_TARGET=payment-service:50051
BOOKING_SERVICE_TARGET=booking-service:50051
API_GATEWAY_PORT=8080
FRONTEND_PORT=5173
VITE_API_BASE_URL=http://localhost:8080
```

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
docker compose logs -f api-gateway booking-service payment-service
```

Detener:

```bash
docker compose down
```

Detener y borrar volumenes de datos:

```bash
docker compose down -v
```

## Endpoints de Booking

Todas las rutas entran por `api-gateway`.

### Crear Reserva

```bash
curl -X POST http://localhost:8080/bookings \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-demo",
    "doctor_id": "doctor-demo",
    "slot_id": "slot-cardio-0900",
    "notes": "Control creado desde demo"
  }'
```

Respuesta esperada si `availability-service` esta disponible:

```json
{
  "booking_id": "uuid",
  "status": "PENDING_PAYMENT",
  "reserved_until": "2026-05-07T12:00:00Z"
}
```

Si `availability-service` no esta implementado o no esta levantado, el gateway devuelve un error controlado proveniente de `booking-service`.

### Listar Reservas de Paciente

```bash
curl "http://localhost:8080/bookings?patient_id=patient-demo"
```

Filtro por estado:

```bash
curl "http://localhost:8080/bookings?patient_id=patient-demo&status=PENDING_PAYMENT"
```

### Obtener Detalle de Reserva

```bash
curl http://localhost:8080/bookings/{booking_id}
```

La respuesta incluye la reserva y sus eventos persistidos.

### Confirmar Reserva

Primero debe existir un pago aprobado o completado en `payment-service`.

```bash
curl -X POST http://localhost:8080/bookings/{booking_id}/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": "{payment_id}"
  }'
```

`booking-service` valida el pago por gRPC contra `payment-service` y luego solicita confirmar el slot a `availability-service`.

### Cancelar Reserva

```bash
curl -X PATCH http://localhost:8080/bookings/{booking_id}/cancel \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Paciente solicita reagendar"
  }'
```

`booking-service` libera el slot por gRPC contra `availability-service` antes de persistir la cancelacion.

## Endpoints de Payment

### Crear Pago

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{
    "booking_id": "{booking_id}",
    "user_id": "patient-demo",
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
7. Crear y procesar un pago desde Insomnia.
8. Confirmar la reserva con el `payment_id`.
9. Cancelar una reserva y verificar el cambio de estado.

Mientras `availability-service` no exista, los pasos que reservan, confirman o liberan slots sirven para demostrar resiliencia y traduccion de errores, pero no para completar el flujo end-to-end exitoso.

## Pruebas con Insomnia

El archivo `Insomnia_2026-05-07.yaml` incluye carpetas para:

- `Bookings`
- `Payments`

Importa el archivo en Insomnia y usa el ambiente base con `base_url = http://localhost:8080`.

## Verificacion Tecnica

Comandos utiles:

```bash
docker compose config
docker compose build api-gateway booking-service payment-service frontend
docker run --rm -v "$PWD":/workspace -w /workspace/booking-service golang:1.26-alpine go test ./...
docker run --rm -v "$PWD":/workspace -w /workspace/api-gateway golang:1.26-alpine go test ./...
```

## Estructura Relevante

```text
MedConnect/
├── api-gateway/                    # Gateway HTTP unificado
├── booking-service/                # Servicio gRPC de reservas
├── frontend/                       # Cliente web React
├── payment-service/                # Servicio gRPC de pagos
├── docker-compose.yml
├── .env.example
└── Insomnia_2026-05-07.yaml
```

## Estado de Entrega

Implementado:

- Contrato gRPC de reservas.
- Persistencia de reservas y eventos.
- API Gateway para reservas y pagos.
- Frontend minimo para flujo paciente.
- Docker Compose con bases de reservas y pagos.
- Cliente de pagos de `booking-service` alineado con el contrato real de `payment-service`.

Pendiente por dependencia externa:

- `availability-service` real para completar `HoldSlot`, `ReleaseHeldSlot` y `ConfirmSlotBooking`.

# Implementación Kafka: notificaciones asíncronas con outbox transaccional

## 1. Responsabilidad implementada

La implementación agrega comunicación asíncrona basada en Kafka para publicar eventos de reservas desde `booking-service` y consumirlos en un nuevo `notification-service`.

El objetivo no es reemplazar el flujo síncrono existente de reservas, pagos y disponibilidad. La responsabilidad de Kafka queda acotada a un caso de uso claro: generar notificaciones simuladas cuando una reserva cambia de estado.

## 2. Problema técnico resuelto

Antes de esta implementación, `booking-service` solo registraba eventos en su base de datos local mediante `appointment_events`. Esos eventos servían como auditoría interna, pero no podían ser consumidos por otro microservicio sin acoplarlo directamente a la base de datos de reservas o sin agregar llamadas síncronas adicionales.

La solución implementada usa el patrón outbox transaccional:

1. `booking-service` guarda la reserva.
2. En la misma transacción guarda el evento de auditoría en `appointment_events`.
3. En la misma transacción guarda el mensaje pendiente en `outbox_events`.
4. Un dispatcher interno publica los mensajes pendientes en Kafka.
5. `notification-service` consume los eventos, genera una notificación simulada y la persiste idempotentemente.

Esto evita el problema clásico de publicar directo a Kafka después de guardar en la base de datos: si la aplicación cae entre el commit de PostgreSQL y la publicación del mensaje, el evento se perdería.

## 3. Arquitectura

```text
API Gateway ──gRPC──► Booking Service ──transacción──► booking_db
                                      │                 ├─ appointments
                                      │                 ├─ appointment_events
                                      │                 └─ outbox_events
                                      │
                                      ▼
                              Outbox Dispatcher
                                      │
                                      ▼
                         medconnect.booking.events.v1
                                      │
                                      ▼
                            Notification Service
                                      │
                                      ▼
                              notification_db
```

## 4. Componentes modificados o agregados

### `docker-compose.yml`

Se agregó:

- `kafka` con imagen `apache/kafka:4.2.1` en modo KRaft.
- `kafka-init` para crear tópicos automáticamente.
- `notification-service`.
- `notification_db`.
- Variables de entorno para configurar broker, tópicos, dispatcher y consumer group.

### `booking-service`

Se agregó:

- Migración `000003_create_outbox_events`.
- Tabla `outbox_events`.
- Inserción de eventos outbox dentro de las transacciones existentes de creación, confirmación y cancelación de reservas.
- Dispatcher interno con polling, publicación Kafka y backoff exponencial.
- Productor Kafka basado en `franz-go`.

### `notification-service`

Nuevo microservicio Go que:

- Consume `medconnect.booking.events.v1`.
- Valida contrato del evento.
- Persiste notificaciones simuladas en PostgreSQL.
- Usa `event_id` único para idempotencia.
- Publica en `medconnect.booking.events.dlq.v1` cuando el mensaje es inválido o se agotan los reintentos.
- Confirma offset manualmente solo después de persistir o enviar a DLQ.

## 5. Tópicos Kafka

| Tópico | Particiones | Retención | Uso |
|---|---:|---:|---|
| `medconnect.booking.events.v1` | 3 | 7 días | Eventos válidos de reservas. |
| `medconnect.booking.events.dlq.v1` | 1 | 14 días | Mensajes inválidos o no procesables. |

La clave del mensaje es `booking_id`. Esto mantiene el orden relativo de eventos de una misma reserva dentro de una partición.

## 6. Contrato de evento

Ejemplo de evento publicado:

```json
{
  "event_id": "11111111-1111-1111-1111-111111111111",
  "event_type": "booking.created",
  "schema_version": 1,
  "occurred_at": "2026-05-03T23:00:00Z",
  "aggregate_id": "22222222-2222-2222-2222-222222222222",
  "payload": {
    "booking_id": "22222222-2222-2222-2222-222222222222",
    "patient_id": "33333333-3333-3333-3333-333333333333",
    "doctor_id": "44444444-4444-4444-4444-444444444444",
    "slot_id": "55555555-5555-5555-5555-555555555555",
    "status": "PENDING_PAYMENT",
    "reserved_until": "2026-05-03T23:15:00Z",
    "event_payload": {
      "status": "PENDING_PAYMENT"
    }
  }
}
```

Eventos soportados:

- `booking.created`
- `booking.confirmed`
- `booking.cancelled`
- `booking.expired`

## 7. Justificación técnica

### Outbox transaccional

Se eligió outbox transaccional porque la base de datos de reservas y Kafka no comparten una transacción distribuida. El outbox permite que la intención de publicar quede persistida junto con el cambio de estado de la reserva.

Si Kafka está caído, la reserva no falla por esa razón. El evento queda en `outbox_events` con `published_at = NULL` y el dispatcher lo reintenta.

### Entrega at-least-once

La implementación busca entrega al menos una vez. Esto significa que un evento puede publicarse más de una vez en escenarios como:

- El dispatcher publica en Kafka.
- El proceso cae antes de marcar `published_at`.
- Al reiniciar, vuelve a publicar el mismo evento.

El consumidor maneja ese caso con idempotencia mediante `UNIQUE(event_id)` en `notification_db.notifications`.

### Commit manual de offsets

`notification-service` tiene auto-commit deshabilitado. El offset se confirma solo después de:

- guardar correctamente la notificación, o
- publicar el mensaje en DLQ.

Esto evita marcar como procesado un evento que todavía no tuvo efecto durable.

### DLQ

La DLQ evita que un mensaje inválido bloquee indefinidamente el consumo de eventos posteriores. El mensaje original se publica junto con headers que indican:

- `dlq_reason`
- `source_topic`

## 8. Comportamiento ante fallas

| Escenario | Comportamiento |
|---|---|
| Kafka caído al crear reserva | La reserva se persiste; el evento queda pendiente en `outbox_events`. |
| Kafka vuelve a estar disponible | El dispatcher publica los eventos pendientes. |
| `notification-service` caído | Kafka retiene los eventos y el consumer group retoma al reiniciar. |
| Mensaje duplicado | `notification_db.notifications.event_id` evita duplicar la notificación. |
| Mensaje inválido | Se envía a DLQ y se confirma el offset. |
| PostgreSQL de notificaciones falla | El processor reintenta; si agota reintentos, envía a DLQ. |

## 9. Archivos principales

| Archivo | Rol |
|---|---|
| `booking-service/migrations/000003_create_outbox_events.up.sql` | Crea tabla outbox. |
| `booking-service/internal/repository/postgres/outbox.go` | Construye eventos y administra outbox. |
| `booking-service/internal/outbox/dispatcher.go` | Publica eventos pendientes con reintentos. |
| `booking-service/internal/messaging/kafka/publisher.go` | Productor Kafka. |
| `notification-service/internal/consumer/kafka.go` | Consumidor Kafka, commit manual y DLQ. |
| `notification-service/internal/processor/processor.go` | Validación, notificación simulada, reintentos. |
| `notification-service/internal/repository/postgres/repository.go` | Persistencia idempotente. |
| `notification-service/db/init.sql` | Esquema de notificaciones. |

## 10. Cómo ejecutar

Desde `MedConnect/`:

```bash
docker compose up --build
```

Para revisar eventos pendientes en el outbox:

```bash
docker compose exec booking_db psql -U booking -d booking_db -c "SELECT id, event_type, attempts, published_at, last_error FROM outbox_events ORDER BY created_at DESC;"
```

Para revisar notificaciones generadas:

```bash
docker compose exec notification_db psql -U notification -d notification_db -c "SELECT event_id, booking_id, event_type, recipient_id, message, created_at FROM notifications ORDER BY created_at DESC;"
```

Para revisar tópicos:

```bash
docker compose exec kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --list
```

## 11. Verificación ejecutada

Pruebas unitarias:

```bash
cd booking-service
go test ./...
```

```bash
cd notification-service
go test ./...
```

Validación de Compose:

```bash
docker compose config
```

La construcción Docker de `booking-service` y `notification-service` depende de que Docker Desktop esté activo localmente.

## 12. Limitaciones asumidas

- Kafka está configurado con un solo broker local y replication factor 1.
- No se implementa correo, SMS ni push real; la notificación es simulada y persistida.
- No se agrega Schema Registry; el contrato se versiona con `schema_version`.
- No se agregan métricas ni trazabilidad distribuida.
- No se implementa TLS/SASL para Kafka porque el entorno es local de curso.
- No se modifica el frontend ni se agrega endpoint público para notificaciones.

## 13. Relación con la rúbrica

| Criterio | Evidencia implementada |
|---|---|
| Comunicación asíncrona | Kafka con tópico de eventos de reservas y consumidor independiente. |
| Desacoplamiento | Booking no llama directamente a notification-service. |
| Tolerancia a fallos | Outbox, reintentos, DLQ e idempotencia. |
| Escalabilidad | Tópico principal con 3 particiones y consumer group. |
| Documentación técnica | Este documento describe arquitectura, contrato, decisiones y pruebas. |
| Defensa individual | La implementación tiene decisiones técnicas defendibles: outbox, at-least-once, commit manual e idempotencia. |

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
- Dispatcher interno con polling, claim de filas mediante `FOR UPDATE SKIP LOCKED`, publicación Kafka y backoff exponencial.
- Estado final `FAILED` para eventos outbox que superan `OUTBOX_MAX_ATTEMPTS`.
- Productor Kafka basado en `franz-go`.
- Métricas Prometheus de eventos pendientes, edad del evento pendiente más antiguo, publicaciones exitosas, fallos y fallos finales.

### `notification-service`

Nuevo microservicio Go que:

- Consume `medconnect.booking.events.v1`.
- Valida contrato del evento.
- Persiste notificaciones simuladas en PostgreSQL.
- Usa `event_id` único para idempotencia.
- Publica en `medconnect.booking.events.dlq.v1` cuando el mensaje es inválido o se agotan los reintentos.
- Confirma offset manualmente solo después de persistir o enviar a DLQ.
- Expone métricas Prometheus de eventos insertados, duplicados y enviados a DLQ.

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

Nota sobre `booking.expired`: el contrato y el consumidor lo soportan para mantener compatibilidad con una futura expiración automática de reservas. En el flujo actual demostrado por la API se generan `booking.created`, `booking.confirmed` y `booking.cancelled`; `booking.expired` no se genera automáticamente todavía.

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

### Alternativas descartadas

| Alternativa | Por qué se descartó |
|---|---|
| Llamada gRPC directa desde `booking-service` a `notification-service` | Habría acoplado el flujo de reserva a la disponibilidad de notificaciones. Si notification-service cae, la reserva podría degradarse por una responsabilidad secundaria. |
| Publicar directo a Kafka después del commit de PostgreSQL | Puede perder eventos si la aplicación cae entre guardar la reserva y publicar el mensaje. El outbox transaccional evita esa ventana de inconsistencia. |
| Auto-commit de offsets en el consumidor | Podía confirmar un mensaje antes de persistir la notificación. Se eligió commit manual para confirmar solo después de un efecto durable o DLQ. |
| Exactly-once con transacciones Kafka | Aumentaba complejidad para un caso de notificaciones simuladas. Se eligió at-least-once más idempotencia por `event_id`, suficiente para evitar notificaciones duplicadas. |

## 8. Comportamiento ante fallas

| Escenario | Comportamiento |
|---|---|
| Kafka caído al crear reserva | La reserva se persiste; el evento queda pendiente en `outbox_events`. |
| Kafka vuelve a estar disponible | El dispatcher publica los eventos pendientes. |
| Múltiples dispatchers de booking activos | Cada dispatcher reclama filas con `FOR UPDATE SKIP LOCKED` y `locked_until`, evitando que dos instancias procesen el mismo evento al mismo tiempo. |
| Publicación falla repetidamente | Se incrementa `attempts`; al superar `OUTBOX_MAX_ATTEMPTS`, el evento queda en estado `FAILED` con `last_error`. |
| `notification-service` caído | Kafka retiene los eventos y el consumer group retoma al reiniciar. |
| Mensaje duplicado | `notification_db.notifications.event_id` evita duplicar la notificación. |
| Mensaje inválido | Se envía a DLQ y se confirma el offset. |
| PostgreSQL de notificaciones falla | El processor reintenta; si agota reintentos, envía a DLQ. |

## 9. Métricas agregadas

`booking-service` expone en `/metrics`:

- `medconnect_booking_outbox_published_total`
- `medconnect_booking_outbox_publish_failed_total`
- `medconnect_booking_outbox_final_failed_total`
- `medconnect_booking_outbox_pending`
- `medconnect_booking_outbox_oldest_pending_age_seconds`
- `medconnect_booking_outbox_final_failed`

`notification-service` expone en `/metrics`:

- `medconnect_notification_events_processed_total{result="inserted|duplicate|dlq|error"}`
- `medconnect_notification_dlq_messages_total`

Prometheus scrapea `notification-service:9090` además de los servicios existentes.

## 10. Archivos principales

| Archivo | Rol |
|---|---|
| `booking-service/migrations/000003_create_outbox_events.up.sql` | Crea tabla outbox. |
| `booking-service/internal/repository/postgres/outbox.go` | Construye eventos y administra outbox. |
| `booking-service/internal/outbox/dispatcher.go` | Publica eventos pendientes con reintentos. |
| `booking-service/internal/outbox/metrics.go` | Métricas Prometheus del outbox. |
| `booking-service/internal/messaging/kafka/publisher.go` | Productor Kafka. |
| `booking-service/internal/messaging/kafka/publisher_integration_test.go` | Smoke test opcional contra Kafka real. |
| `notification-service/internal/consumer/kafka.go` | Consumidor Kafka, commit manual y DLQ. |
| `notification-service/internal/consumer/kafka_integration_test.go` | Smoke test opcional de publicación a DLQ contra Kafka real. |
| `notification-service/internal/processor/processor.go` | Validación, notificación simulada, reintentos. |
| `notification-service/internal/repository/postgres/repository.go` | Persistencia idempotente. |
| `notification-service/internal/observability/metrics.go` | Métricas Prometheus de procesamiento y DLQ. |
| `notification-service/db/init.sql` | Esquema de notificaciones. |

## 11. Cómo ejecutar

Desde `MedConnect/`:

```bash
docker compose up --build
```

Si ya existía un volumen de `booking_db` antes de agregar `status`, `locked_until` y `failed_at` al outbox, recrear los volúmenes para que Docker ejecute nuevamente las migraciones de inicialización:

```bash
docker compose down -v
docker compose up --build
```

Para revisar eventos pendientes en el outbox:

```bash
docker compose exec booking_db psql -U booking -d booking_db -c "SELECT id, event_type, status, attempts, published_at, failed_at, last_error FROM outbox_events ORDER BY created_at DESC;"
```

Para revisar notificaciones generadas:

```bash
docker compose exec notification_db psql -U notification -d notification_db -c "SELECT event_id, booking_id, event_type, recipient_id, message, created_at FROM notifications ORDER BY created_at DESC;"
```

Para revisar tópicos:

```bash
docker compose exec kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --list
```

## 12. Verificación ejecutada

Evidencia local tomada el 2026-07-06.

Pruebas unitarias:

```bash
cd booking-service
go test ./...
```

Salida resumida:

```text
ok github.com/MedConnect/booking-service/internal/config
ok github.com/MedConnect/booking-service/internal/messaging/kafka
ok github.com/MedConnect/booking-service/internal/outbox
ok github.com/MedConnect/booking-service/internal/repository/postgres
ok github.com/MedConnect/booking-service/internal/service
ok github.com/MedConnect/booking-service/internal/transport/grpc
ok github.com/MedConnect/booking-service/pb
```

```bash
cd notification-service
go test ./...
```

Salida resumida:

```text
ok github.com/MedConnect/notification-service/internal/config
ok github.com/MedConnect/notification-service/internal/consumer
ok github.com/MedConnect/notification-service/internal/processor
ok github.com/MedConnect/notification-service/internal/repository/postgres
```

Validación de Compose:

```bash
docker compose config
```

Resultado: configuración válida; Compose resolvió servicios, redes, volúmenes y variables sin errores.

Listado de tópicos observado con el stack corriendo:

```text
__consumer_offsets
medconnect.booking.events.dlq.v1
medconnect.booking.events.v1
```

Consulta de outbox observada antes de generar una reserva de demo:

```text
 id | event_type | attempts | published_at | last_error
----+------------+----------+--------------+------------
(0 rows)
```

Consulta de notificaciones observada antes de generar una reserva de demo:

```text
 event_id | booking_id | event_type | recipient_id | message | created_at
----------+------------+------------+--------------+---------+------------
(0 rows)
```

Para una demo completa, crear/cancelar/confirmar una reserva desde el API Gateway y repetir las dos consultas anteriores. En ese caso, `outbox_events` debe mostrar el evento publicado con `status = 'PUBLISHED'` y `notifications` debe mostrar la notificación persistida.

Pruebas de integración opcionales contra Kafka real:

```bash
cd booking-service
KAFKA_INTEGRATION=1 KAFKA_INTEGRATION_BROKERS=localhost:9092 go test ./internal/messaging/kafka -run TestPublisherPublishesToRealKafka
```

```bash
cd notification-service
KAFKA_INTEGRATION=1 KAFKA_INTEGRATION_BROKERS=localhost:9092 go test ./internal/consumer -run TestDLQPublisherPublishesToRealKafka
```

La construcción Docker de `booking-service` y `notification-service` depende de que Docker Desktop esté activo localmente.

## 13. Limitaciones asumidas y trade-offs aceptados

- Kafka está configurado con un solo broker local y replication factor 1. Se aceptó porque se busca demostrar integración y tolerancia a fallos de aplicación, no alta disponibilidad real de Kafka.
- No se implementa correo, SMS ni push real; la notificación es simulada y persistida. Se aceptó porque el objetivo es el flujo asíncrono y la idempotencia, no la integración con proveedores externos.
- No se agrega Schema Registry; el contrato se versiona con `schema_version`. Se aceptó para evitar infraestructura adicional en una demo local, manteniendo una base para evolucionar contratos.
- No se implementa TLS/SASL para Kafka porque el entorno es de demo local. En producción se requeriría autenticación, autorización y cifrado.
- No se modifica el frontend ni se agrega endpoint público para notificaciones. Se aceptó para mantener el bloque acotado a backend y demostrar el resultado con consultas SQL.
- `booking.expired` está soportado por contrato, pero no se genera automáticamente en el flujo actual. Se dejó preparado para una futura tarea programada de expiración de reservas.
- El outbox usa `locked_until` como lease. Si una instancia cae después de publicar y antes de marcar `PUBLISHED`, el evento puede republicarse al expirar el lease; esto es consistente con at-least-once y se controla con idempotencia en el consumidor.


# Documento tecnico vivo - SAGA en MedConnect Backend

**Bloque individual:** SAGA  
**Servicios involucrados:** `api-gateway`, `booking-service`, `availability-service`, `payment-service`  
**Servicio orquestador:** `booking-service`  
**Modelo elegido:** SAGA orquestada  
**Estado:** orquestador de dominio, contrato gRPC y endpoints REST SAGA implementados; repositorio PostgreSQL pendiente  
**Base:** `rubrica_entrega2.md`, `README.md`, `docs/sharding_technical_doc.md`, `docs/observabilidad_technical_doc.md`  
**Ultima actualizacion:** 2026-07-06

---

## 1. Descripcion del bloque SAGA

El bloque SAGA coordina una reserva medica completa cuando la operacion cruza mas de una base de datos y mas de un microservicio. En MedConnect no existe una transaccion ACID unica entre `booking_db`, los shards de `availability-service` y `payments_db`, por lo que la consistencia se logra con pasos locales y compensaciones.

El flujo que se busca orquestar es:

```text
Cliente autenticado
  -> API Gateway REST
  -> booking-service (orquestador SAGA)
  -> availability-service: HoldSlot
  -> booking_db: crear reserva PENDING_PAYMENT
  -> payment-service: CreatePayment
  -> payment-service: ProcessPayment
  -> availability-service: ConfirmSlotBooking
  -> booking_db: confirmar reserva
```

Si un paso falla despues de haber aplicado cambios previos, el orquestador ejecuta acciones compensatorias para dejar el sistema en un estado conocido. Por ejemplo, si el pago falla despues de retener un slot, la SAGA debe liberar ese slot y cancelar la reserva.

Servicios participantes:

- `api-gateway`: expone endpoints REST y propaga identidad/request ID.
- `booking-service`: orquesta la SAGA, guarda estado durable y decide compensaciones.
- `availability-service`: retiene, confirma o libera slots. El sharding queda encapsulado dentro de este servicio.
- `payment-service`: crea, procesa y reembolsa pagos.

`auth-service` participa solo antes de la SAGA para validar identidad en el gateway. No se considera participante transaccional porque registrar o validar usuarios no forma parte de la consistencia de una reserva.

---

## 2. Decision tecnica principal

La decision definida para este bloque es usar una SAGA orquestada, con `booking-service` como orquestador.

Razones especificas del dominio:

- `booking-service` ya es el servicio que conoce el ciclo de vida de una reserva.
- El flujo de reserva ya llama a `availability-service` para slots y a `payment-service` para pagos.
- La reserva es la entidad que conecta `patient_id`, `doctor_id`, `slot_id` y `payment_id`.
- Permite guardar el estado de la SAGA cerca de `appointments` y `appointment_events`.
- Evita que `api-gateway` tenga reglas de negocio o compensaciones.
- Respeta el sharding existente: `booking-service` no necesita conocer shards, solo usa los RPC actuales de disponibilidad.

Alternativas descartadas:

- SAGA coreografiada con eventos: se descarta para este bloque porque requiere infraestructura/eventos adicionales y se cruza con el bloque Kafka del grupo.
- Orquestador en `api-gateway`: se descarta porque el gateway debe transformar HTTP/gRPC y autenticar, no decidir transiciones de negocio ni compensaciones.
- Nuevo microservicio `saga-service`: se descarta por complejidad operacional innecesaria para el alcance del proyecto; agregaria otro despliegue y otra base sin aportar claridad al flujo actual.
- Transaccion distribuida tipo 2PC: se descarta porque PostgreSQL separados y gRPC entre servicios no estan preparados para coordinar commits distribuidos; ademas, el objetivo de la rubrica es demostrar SAGA y compensaciones.

---

## 3. Alcance funcional propuesto

La SAGA cubrira el caso de uso de reserva pagada de punta a punta:

```text
crear reserva + bloquear slot + crear pago + procesar pago + confirmar slot + confirmar reserva
```

Endpoints REST propuestos:

```text
POST /booking-sagas
GET  /booking-sagas/{saga_id}
```

Payload propuesto para iniciar la SAGA:

```json
{
  "doctor_id": "7e0d2ab1-164e-4a28-8b95-f24293dd0e91",
  "slot_id": "0f5c2b6a-1a87-4b7e-ae2c-37ef2f9f1c21",
  "notes": "Control de rutina",
  "amount": 15000,
  "currency": "CLP",
  "payment_method_id": "method-demo"
}
```

Respuesta esperada al completar correctamente:

```json
{
  "saga_id": "...",
  "status": "COMPLETED",
  "booking_id": "...",
  "payment_id": "...",
  "confirmation_code": "MED-..."
}
```

El flujo REST actual del README puede mantenerse para compatibilidad y pruebas manuales existentes. La SAGA se demostrara con endpoints nuevos para no romper el happy path ya usado por otros bloques.

---

## 4. Estados de la SAGA

Estados durables propuestos:

| Estado | Significado |
|---|---|
| `STARTED` | La SAGA fue creada y aun no ejecuta cambios externos. |
| `SLOT_HELD` | `availability-service` retuvo el slot. |
| `BOOKING_CREATED` | La reserva existe en `booking_db` como `PENDING_PAYMENT`. |
| `PAYMENT_CREATED` | `payment-service` creo el pago asociado. |
| `PAYMENT_COMPLETED` | El pago fue procesado exitosamente. |
| `SLOT_CONFIRMED` | El slot fue confirmado como `booked`. |
| `COMPLETED` | La reserva quedo `CONFIRMED` y la SAGA termino. |
| `COMPENSATING` | El orquestador esta ejecutando compensaciones. |
| `COMPENSATED` | Los cambios previos fueron compensados y el sistema quedo controlado. |
| `FAILED` | La SAGA fallo antes de requerir compensacion o tras compensarse. |
| `COMPENSATION_FAILED` | Al menos una compensacion fallo y requiere revision manual. |

Los eventos de SAGA deben registrar cada paso para que el estado sea auditable en la demo y en logs.

---

## 5. Pasos y compensaciones

| Paso | Accion | Servicio | Compensacion si falla un paso posterior |
|---|---|---|---|
| 1 | Crear registro SAGA `STARTED` | `booking-service` | No aplica. |
| 2 | `HoldSlot(slot_id, booking_id)` | `availability-service` | `ReleaseHeldSlot(slot_id, booking_id)`. |
| 3 | Crear appointment `PENDING_PAYMENT` | `booking-service` | Cancelar appointment si fue creado; liberar slot. |
| 4 | `CreatePayment(booking_id, user_id, amount, currency)` | `payment-service` | Cancelar appointment; liberar slot. |
| 5 | `ProcessPayment(payment_id, payment_method_id)` | `payment-service` | Si queda rechazado/fallido: cancelar appointment y liberar slot. |
| 6 | `ConfirmSlotBooking(slot_id, booking_id)` | `availability-service` | `RefundPayment(payment_id)`, `ReleaseHeldSlot`, cancelar appointment. |
| 7 | Confirmar appointment `CONFIRMED` | `booking-service` | `RefundPayment(payment_id)`, `ReleaseHeldSlot`, cancelar appointment si es posible. |

La compensacion principal del dominio es liberar el slot para no bloquear disponibilidad medica si la reserva pagada no termina correctamente. Si el pago ya fue capturado, se agrega reembolso como compensacion monetaria.

---

## 6. Comportamiento ante fallos

| Escenario | Comportamiento esperado |
|---|---|
| `availability-service` no responde en `HoldSlot` | La SAGA queda `FAILED`; no hay cambios previos que compensar. |
| Slot no disponible | La SAGA queda `FAILED`; el error se reporta como estado invalido/precondicion fallida. |
| Falla DB al crear appointment tras `HoldSlot` | Se ejecuta `ReleaseHeldSlot`; la SAGA queda `COMPENSATED` o `COMPENSATION_FAILED`. |
| `payment-service` no crea pago | Se cancela la reserva si existe y se libera el slot. |
| Procesador de pago falla o rechaza | Se registra fallo, se cancela la reserva y se libera el slot. |
| Pago aprobado pero falla `ConfirmSlotBooking` | Se intenta `RefundPayment`, luego liberar slot y cancelar reserva. |
| Pago aprobado y slot confirmado, pero falla confirmar reserva | Se intenta reembolso y liberacion del slot; si algo falla queda `COMPENSATION_FAILED`. |
| Compensacion falla por dependencia caida | Se reintenta acotadamente y se deja estado durable para revision. |

---

## 7. Reintentos e idempotencia

Configuracion cargada por `booking-service` para que el orquestador SAGA la use en los siguientes pasos de implementacion:

```text
BOOKING_SAGA_MAX_RETRIES=3
BOOKING_SAGA_RETRY_DELAY=200ms
```

Criterios:

- Reintentar llamadas gRPC externas y compensaciones con limite fijo.
- No reintentar escrituras locales dentro de una transaccion SQL ya fallida sin volver a evaluar estado.
- Consultar `GetPaymentByBooking` antes de crear otro pago para una SAGA que ya avanzo.
- Tratar pago `COMPLETED` como exito si el reintento encuentra que el pago ya fue procesado.
- Tratar pago `REFUNDED` como compensacion ya aplicada.
- Mantener `booking_id` estable dentro de la SAGA para que `HoldSlot`, `ConfirmSlotBooking` y `ReleaseHeldSlot` operen sobre la misma reserva.

---

## 8. Relacion con Sharding y Observabilidad

### Sharding

La SAGA no debe conocer shards ni abrir conexiones directas a las bases de disponibilidad. Todas las operaciones de slots se hacen por los RPC existentes de `availability-service`:

```text
HoldSlot
ConfirmSlotBooking
ReleaseHeldSlot
```

Esto mantiene la decision del bloque Sharding: `availability-service` resuelve internamente `slot_id -> shard` mediante su directorio construido al iniciar.

### Observabilidad

La SAGA debe aprovechar el `X-Request-ID` existente propagado desde `api-gateway`. Los logs nuevos deberian incluir:

```text
saga_id
booking_id
payment_id
step
status
```

Las metricas propuestas para una etapa posterior son contadores por resultado de SAGA y compensaciones. Deben agregarse en `booking-service` sin cambiar los nombres de metricas ya implementados por Observabilidad.

---

## 9. Persistencia propuesta

Las tablas nuevas viviran en `booking_db`, porque la SAGA es parte del ciclo de vida de reservas y necesita referenciar `appointments`.

Tablas:

- `booking_sagas`: estado actual de cada SAGA.
- `booking_saga_events`: historial ordenado de pasos, fallos y compensaciones.

Campos clave de `booking_sagas`:

- `id`: identificador de SAGA.
- `booking_id`: reserva asociada cuando exista.
- `payment_id`: pago asociado cuando exista.
- `patient_id`, `doctor_id`, `slot_id`: datos de routing y auditoria.
- `status`: estado actual de la SAGA.
- `current_step`: ultimo paso alcanzado.
- `compensation_status`: estado de compensacion si aplica.
- `retry_count`: reintentos realizados para el paso actual.
- `last_error`: ultimo error observable.
- `created_at`, `updated_at`, `completed_at`.

---

## 10. Trade-offs y limitaciones

- Se acepta que `booking-service` concentre la orquestacion porque ya coordina reservas, pagos y disponibilidad. El trade-off es que el servicio crece en responsabilidad, pero evita un microservicio adicional.
- La SAGA inicial sera sin worker asincrono. El request puede esperar a que termine el flujo. Esto simplifica la demo y mantiene consistencia visible, pero operaciones largas podrian exceder timeouts si un participante demora demasiado.
- La recuperacion de `COMPENSATION_FAILED` quedara inicialmente manual/observable. Esto es aceptable para una primera entrega porque deja el caso explicitamente registrado y no oculta inconsistencias.
- Las compensaciones no son transacciones inversas perfectas: liberar un slot o reembolsar un pago depende de que los servicios participantes esten disponibles.
- No se modifica el sharding de disponibilidad; si el directorio `slot_id -> shard` de `availability-service` no conoce un slot, la SAGA falla aunque el orquestador este correcto.

---

## 11. Plan de implementacion

1. Crear este documento tecnico vivo.
2. Agregar migracion SQL para `booking_sagas` y `booking_saga_events`.
3. Montar la migracion nueva en `docker-compose.yml` para inicializacion local de `booking_db`.
4. Extender configuracion de `booking-service` con reintentos SAGA.
5. Extender cliente interno de `payment-service` usado por `booking-service`.
6. Implementar orquestador SAGA en la capa `internal/service` de `booking-service`.
7. Agregar repositorio SAGA en `booking-service/internal/repository/postgres`.
8. Agregar metodos gRPC y REST para iniciar y consultar SAGA.
9. Agregar tests de camino feliz, fallos y compensaciones.
10. Validar con tests por modulo y flujo real Docker.

---

## 12. Evidencia esperada

Comandos de verificacion por modulo afectado:

```bash
cd booking-service
go test ./...

cd ../api-gateway
go test ./...

cd ../payment-service
go test ./...
```

Validacion backend real:

```bash
docker compose down -v
docker compose up -d --build api-gateway
docker compose ps
docker compose logs -f booking-service
```

Casos a demostrar:

- SAGA completa termina `COMPLETED`.
- Pago rechazado/fallido termina compensado y slot vuelve a `available`.
- Fallo despues de `HoldSlot` ejecuta `ReleaseHeldSlot`.
- Estado de SAGA se puede consultar por `GET /booking-sagas/{saga_id}`.

---

## 13. Registro de avance

| Fecha | Cambio | Estado |
|---|---|---|
| 2026-07-06 | Se define SAGA orquestada con `booking-service` como orquestador. | Completado |
| 2026-07-06 | Se crea documento tecnico vivo inicial. | Completado |
| 2026-07-06 | Se planifica persistencia de `booking_sagas` y `booking_saga_events`. | Completado |
| 2026-07-06 | Se agregan `BOOKING_SAGA_MAX_RETRIES` y `BOOKING_SAGA_RETRY_DELAY` a la configuracion de `booking-service`. | Completado |
| 2026-07-06 | Se extiende el cliente interno de `payment-service` en `booking-service` con `CreatePayment`, `ProcessPayment`, `RefundPayment` y `GetPaymentByBooking`. | Completado |
| 2026-07-06 | Se implementa el orquestador SAGA en `booking-service/internal/service` con interfaz `SagaRepository`, flujo feliz, compensaciones y tests unitarios. | Completado |
| 2026-07-06 | Se agregan RPC `StartBookingSaga` y `GetBookingSaga` al contrato `booking.proto` y se regeneran `booking.pb.go` / `booking_grpc.pb.go` con `protoc`. | Completado |
| 2026-07-06 | Se agregan endpoints REST autenticados `POST /booking-sagas` y `GET /booking-sagas/{saga_id}` en `api-gateway`. | Completado |

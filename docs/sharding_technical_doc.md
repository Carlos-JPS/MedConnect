# Documento técnico vivo · Sharding en MedConnect Backend

**Bloque individual:** Sharding  
**Servicio:** `availability-service`  
**Estado:** sharding implementado y validado en backend local  
**Base:** `docs/sharding_plan.md`, `rubrica_entrega2.md`, `README.md` y `Sharding y Consistent Hashing.pdf`  
**Última actualización:** 2026-06-29

---

## 1. Descripción del bloque Sharding

El bloque Sharding busca dividir los datos de disponibilidad médica en múltiples bases PostgreSQL. En vez de guardar todos los calendarios y slots en una sola base de `availability-service`, el diseño propone repartirlos entre shards.

El flujo general del sistema se mantiene:

```text
Cliente
  -> API Gateway REST
  -> booking-service / availability-service vía gRPC
  -> availability-service
  -> shard PostgreSQL correspondiente
```

El API Gateway y `booking-service` no conocen los detalles del sharding. El routing es responsabilidad interna de `availability-service`.

---

## 2. Alcance

Se shardea únicamente el dominio de disponibilidad médica:

- `doctor_calendars`.
- `availability_slots`.

No se shardean:

- `auth-service`.
- `booking-service`.
- `payment-service`.
- `api-gateway`.
- `frontend/`.

Esto es intencional: la rúbrica exige que el bloque funcione dentro del sistema real, no que todo MedConnect esté shardeado.

---

## 3. Servicio involucrado

El servicio involucrado directamente es:

```text
availability-service
```

Servicios relacionados durante el flujo completo:

- `api-gateway`: expone endpoints REST.
- `booking-service`: solicita bloqueo, confirmación o liberación de slots.
- `availability-service`: resuelve el shard correcto y opera sobre PostgreSQL.

---

## 4. Shard key elegida

La shard key propuesta es:

```text
doctor_id
```

Razones:

1. La disponibilidad médica pertenece naturalmente a un médico.
2. `GetDoctorAgenda` ya trabaja con `doctor_id`.
3. Permite mantener calendario y slots del médico en el mismo shard.
4. Evita repartir una misma agenda entre varias bases.
5. Es más defendible que usar fecha, especialidad o `slot_id`.

---

## 5. Decisión técnica principal

La decisión principal es usar:

```text
hash por doctor_id + particiones lógicas fijas + partitionMap
```

Flujo propuesto:

```text
doctor_id
  -> hash(doctor_id)
  -> partition_id = hash % NUM_LOGICAL_PARTITIONS
  -> shard_id = partitionMap[partition_id]
  -> conexión PostgreSQL del shard
```

Ejemplo inicial:

```text
NUM_LOGICAL_PARTITIONS = 16

0-7   -> shard0
8-15  -> shard1
```

Esta decisión permite explicar una ruta clara de datos sin usar directamente `hash(key) % número_de_shards_físicos`.

### Estado de implementación del router

Ya se implementó la primera pieza del diseño en:

```text
availability-service/modules/sharding
```

El router usa `crc32.ChecksumIEEE` de la librería estándar de Go para calcular una ruta estable por `doctor_id`:

```text
partition = crc32(doctor_id) % partitionCount
shard = partitionMap[partition]
```

Esta pieza ya está conectada a la configuración, al repositorio shardeado y al arranque de `availability-service`.

Validación ejecutada:

```bash
cd availability-service
go test ./modules/sharding/...
go test ./...
```

Resultado:

```text
PASS
```

### Estado de implementación de la configuración

También se extendió `availability-service/modules/config/config.go` para cargar configuración opcional de sharding sin romper el modo actual de DB única.

Variables incorporadas:

```env
AVAILABILITY_SHARDING_ENABLED
AVAILABILITY_PARTITION_COUNT
AVAILABILITY_SHARDS
AVAILABILITY_PARTITION_MAP
AVAILABILITY_SHARD0_DSN
AVAILABILITY_SHARD1_DSN
```

El comportamiento por defecto sigue siendo compatible:

```text
si AVAILABILITY_SHARDING_ENABLED no está presente -> usar DB única con DSN()
```

Cuando sharding está habilitado, `ValidateShardingConfig()` valida:

- cantidad de particiones;
- shards declarados;
- DSN por shard;
- mapa completo de particiones;
- referencias a shards existentes.

Validación ejecutada:

```bash
cd availability-service
go test ./modules/config/...
go test ./...
```

Resultado:

```text
PASS
```

Esta configuración ya es usada por `main.go` para seleccionar modo single DB o modo sharded. Docker Compose define dos PostgreSQL reales para availability cuando sharding está habilitado.

### Estado de implementación del repositorio shardeado

Se implementó una versión aislada de `ShardedRepository` en:

```text
availability-service/modules/repository/sharded.go
```

El repositorio aplica las reglas centrales del diseño:

```text
GetDoctorAgenda(doctor_id) -> un solo shard
GetAvailableSlots(specialty) -> scatter/gather
Hold/Confirm/Release(slot_id) -> directorio slot_id -> shard
```

También se agregó `ErrSlotShardNotFound` para distinguir el caso en que un `slot_id` no tiene entrada en el directorio.

Validación ejecutada:

```bash
cd availability-service
go test ./modules/repository/...
go test ./...
```

Resultado:

```text
PASS
```

Esta etapa ya está conectada desde `main.go` y fue validada con PostgreSQL shardeados reales en Docker Compose.

### Estado de integración en el arranque

Se modificó `availability-service/main.go` para construir el repositorio adecuado según configuración:

```text
AVAILABILITY_SHARDING_ENABLED=false -> PostgresRepository único
AVAILABILITY_SHARDING_ENABLED=true  -> ShardedRepository
```

En modo sharded, el arranque realiza:

1. Validación de configuración.
2. Creación del router de particiones.
3. Apertura de una conexión PostgreSQL por shard.
4. Construcción del directorio `slot_id -> shard` leyendo slots reales desde cada shard.
5. Creación de `ShardedRepository`.

Para construir el directorio se agregó un método concreto:

```go
PostgresRepository.ListSlotIDs(ctx)
```

Esto mantiene el contrato principal `AvailabilityRepository` centrado en operaciones de negocio y deja la lectura del directorio como detalle de inicialización.

Validación ejecutada:

```bash
cd availability-service
go test ./...
```

Resultado:

```text
PASS
```

Limitación: el directorio se construye al iniciar. Si en el futuro se crean slots en caliente, será necesario refrescarlo o persistir el directorio como metadata actualizable.

---

## 6. Alternativas descartadas

### `hash(doctor_id) % N`

Se descartó usar directamente `hash(doctor_id) % N`, donde `N` es la cantidad de shards físicos.

Motivo: si cambia `N`, muchos médicos cambiarían de shard y habría que mover gran parte de los datos. Con particiones lógicas fijas, en el futuro se podrían mover particiones completas entre shards sin cambiar la función de hash principal.

### Consistent hashing completo con virtual nodes

Se descartó para esta entrega porque aumenta la complejidad y no es necesario para demostrar el bloque según la rúbrica.

### Otros servicios

- `auth-service`: complica la unicidad global de email y login.
- `booking-service`: coordina reservas, disponibilidad y pagos.
- `payment-service`: es menos visible para demostrar distribución y routing.

---

## 7. Flujo de lectura y escritura

### 7.1 Consulta de agenda médica

```text
GetDoctorAgenda(doctor_id)
  -> calcular partición
  -> resolver shard
  -> consultar solo ese shard
```

Este es el caso ideal para la shard key elegida.

### 7.2 Consulta de slots por especialidad

`specialty` no es la shard key. Por eso, la consulta requiere scatter/gather:

```text
GetAvailableSlots(specialty, rango_fechas)
  -> consultar todos los shards
  -> combinar resultados
  -> ordenar por fecha
  -> retornar respuesta única
```

### 7.3 Bloqueo, confirmación y liberación de slots

Las operaciones actuales reciben `slot_id`:

- `HoldSlot`.
- `ConfirmSlotBooking`.
- `ReleaseHeldSlot`.

Como `slot_id` no permite calcular directamente el shard por `doctor_id`, se usará un directorio:

```text
slot_id -> shard
```

Flujo:

```text
HoldSlot(slot_id)
  -> buscar slot_id en directorio
  -> obtener shard dueño
  -> ejecutar UPDATE condicional en ese shard
```

La actualización debe evitar doble reserva:

```sql
WHERE id = $slot_id
  AND status = 'available'
```

---

## 8. Comportamiento ante fallos

### Shard caído en operación directa

Si un médico pertenece a un shard caído:

- la consulta de su agenda falla;
- no se consulta otro shard;
- médicos ubicados en shards sanos pueden seguir funcionando.

Esto permite degradación parcial.

### Shard caído en scatter/gather

Si `GetAvailableSlots` necesita consultar todos los shards y uno falla:

- la consulta completa debe fallar;
- no se devuelven resultados parciales como si fueran completos.

Motivo: el contrato actual no distingue respuestas parciales.

### Directorio `slot_id -> shard` inconsistente

Si el directorio apunta a un shard, pero el slot no existe allí:

- se retorna error;
- se registra el problema;
- no se busca silenciosamente en otros shards.

Esto evita escribir en el shard incorrecto.

### Partition map inválido

Si falta una partición o un shard configurado no existe, `availability-service` debería fallar al iniciar. Es preferible fallar temprano antes que enrutar datos médicos incorrectamente.

### Falla posterior a `HoldSlot` en `booking-service`

El flujo real de reserva primero bloquea el slot en `availability-service` y luego persiste la reserva en `booking_db`. Si el insert de la reserva falla después de un `HoldSlot` exitoso, `booking-service` ejecuta una compensación local con `ReleaseHeldSlot(slot_id, booking_id)` para no dejar el slot retenido sin reserva persistida.

Si esa compensación también falla, se conserva el error original de creación de reserva y el slot podría quedar retenido hasta intervención o expiración futura. Esta limitación se acepta porque evita ocultar la causa principal del fallo y mantiene el cambio acotado al flujo actual.

---

## 9. Casos de borde

| Caso | Comportamiento esperado |
|---|---|
| `doctor_id` vacío | Error de validación |
| `slot_id` vacío | Error de validación |
| `slot_id` no existe en directorio | `NotFound` |
| `slot_id` apunta a shard incorrecto | Error de inconsistencia |
| Dos reservas intentan el mismo slot | Solo una cambia `available -> held` |
| Consulta por especialidad | Scatter/gather |
| Shard caído en consulta directa | Error controlado |
| Shard caído en scatter/gather | Falla completa |
| Médico muy demandado | Posible hot spot documentado |
| Insert de booking falla después de `HoldSlot` | Compensación con `ReleaseHeldSlot` |

---

## 10. Trade-offs y limitaciones

### Hot spot por médico popular

Aunque el hash distribuye médicos, un médico muy demandado seguirá concentrando carga en un solo shard.

Trade-off aceptado: mantener toda la agenda de un médico junta simplifica reservas y evita coordinar múltiples shards.

### Scatter/gather por especialidad

Las consultas por especialidad son más costosas porque deben revisar todos los shards. Además, dependen del shard más lento.

Trade-off aceptado: se prioriza una shard key natural para operaciones críticas por médico.

### Directorio crítico

El directorio `slot_id -> shard` agrega metadata crítica. Si está desactualizado, las operaciones por slot pueden fallar.

Trade-off aceptado: evita cambiar contratos gRPC actuales.

### Sin rebalanceo automático

La entrega no implementará rebalanceo automático. El diseño con particiones lógicas deja un camino futuro para mover particiones entre shards.

### Directorio construido al iniciar

El directorio `slot_id -> shard` se construye al iniciar leyendo los slots existentes de cada shard. Si en el futuro se crean slots en caliente, se necesitará refrescar ese directorio o persistir la metadata de routing.

---

## 11. Evidencia de funcionamiento

### Evidencia baseline previa a sharding

El backend fue levantado con:

```bash
docker compose up -d --build api-gateway
docker compose ps
```

Estado observado:

- API Gateway activo en `localhost:8080`.
- Servicios backend activos: `auth-service`, `availability-service`, `booking-service`, `payment-service`.
- Bases PostgreSQL actuales en estado healthy.
- `availability-service` aún usa una sola base: `availability-db`.

Pruebas REST realizadas:

| Flujo | Resultado |
|---|---|
| Registro de usuario | `201 Created` |
| Login | `200 OK` con JWT |
| Consulta de slots por especialidad | `200 OK` usando `from_date`/`to_date` |
| Consulta de agenda por médico | `200 OK` |
| Crear reserva | `502`, falla en `HoldSlot` con `DeadlineExceeded` |

Nota relevante: el README fue actualizado para usar `from_date`/`to_date`, que son los parámetros aceptados por el handler actual.

La falla inicial de `POST /bookings` quedó registrada como problema baseline previo a sharding. El diagnóstico mostró que el slot fijo del README estaba `held` por datos persistidos antiguos y que `availability-service` no persistía `booking_id`/`held_until` al hacer `HoldSlot`.

### Corrección baseline aplicada antes de sharding

Antes de implementar sharding se corrigió `HoldSlot` en `availability-service` para dejar una base más consistente:

- `context.Context` se propaga desde el handler gRPC hasta PostgreSQL.
- Las queries usan `QueryContext` / `QueryRowContext`.
- `HoldSlot` persiste `booking_id` y `held_until`.
- `ConfirmSlotBooking` y `ReleaseHeldSlot` verifican `booking_id`.
- Slot no disponible se mapea a `FailedPrecondition`, que el API Gateway traduce a HTTP `409 Conflict`.

Verificación posterior:

| Flujo | Resultado |
|---|---|
| `/availability/hold` con slot disponible y UUID válido | `200 OK` |
| `/availability/release` del mismo slot | `200 OK` |
| `/availability/hold` sobre slot ya `held` | `409 Conflict` |
| `POST /bookings` con slot disponible | `201 Created` |
| Cancelación de la reserva de prueba | `200 OK` |

Esta corrección es relevante para sharding porque el directorio `slot_id -> shard` y las operaciones mutantes dependen de que `booking_id` y `held_until` sean metadata confiable.

### Evidencia esperada después de implementar sharding

Evidencia ejecutada:

```bash
docker compose up -d --build booking-service api-gateway
docker compose ps
cd availability-service && go test ./...
cd ../booking-service && go test ./...
docker compose config
```

El stack muestra `availability-service` y dos shards de availability:

```text
availability-db-shard-0
availability-db-shard-1
availability-service
api-gateway
booking-service
auth-service
payment-service
```

Logs observados:

```text
availability-service iniciando en modo sharded: particiones=16 shards_físicos=2 names=[shard0 shard1] shards_enrutados=[shard0 shard1]
shard "shard1" aportó 1 slots al directorio
shard "shard0" aportó 2 slots al directorio
directorio de slots sharded construido con 3 entradas
ShardedRepository listo: shards=2 entradas_directorio=3
```

Pruebas finales:

| Flujo | Resultado |
|---|---|
| `GET /availability/slots` por Traumatología | `200 OK` |
| `POST /bookings` con slot de Traumatología | `201 Created`, `PENDING_PAYMENT` |
| `PATCH /bookings/{id}/cancel` | `200 OK`, `CANCELLED` |
| Consulta directa en `availability-db-shard-0` | slot vuelve a `available` |
| Caso borde: insert duplicado en `booking_db` después de `HoldSlot` | error controlado y slot queda `available` por compensación |

Distribución demo validada:

```text
doctor_id 7e0d2ab1... Cardiología      -> partición 2  -> shard0
doctor_id d2f50707... Traumatología    -> partición 7  -> shard0
doctor_id 4f1cb247... Medicina interna -> partición 12 -> shard1
```

---

## 12. Guion breve para video de 3 minutos

> Mi bloque es Sharding y lo apliqué en `availability-service`, porque la disponibilidad médica se agrupa naturalmente por médico. No se shardea todo MedConnect porque la rúbrica pide que el bloque funcione dentro del sistema real, no que todos los servicios estén particionados.
>
> La shard key elegida es `doctor_id`. Esto permite que la agenda completa de un médico viva en un solo shard. Para evitar el problema de `hash(key) % número_de_shards`, propuse usar particiones lógicas fijas y un `partitionMap`.
>
> Las consultas por agenda médica van directo a un shard. Las consultas por especialidad usan scatter/gather, porque la especialidad no es la shard key. Para operaciones que solo reciben `slot_id`, como bloquear o confirmar un slot, el diseño usa un directorio `slot_id -> shard`.
>
> En la demo se levantan dos PostgreSQL de availability. `availability-service` construye al iniciar un directorio de slots, consulta un shard específico para agenda por médico y usa scatter/gather para búsquedas por especialidad. También se validó que una reserva puede bloquear y liberar un slot shardeado desde el flujo real del API Gateway.
>
> Las principales limitaciones son los hot spots por médicos muy demandados, el costo de scatter/gather, que el directorio se construye al iniciar y que no hay rebalanceo automático. Aun así, la implementación demuestra distribución de datos, routing y manejo de fallos coherente con el backend actual.

---

## 13. Relación con la rúbrica

| Criterio | Cómo se cubre |
|---|---|
| Descripción del bloque | Se explica qué se shardea, por qué y cómo encaja en MedConnect |
| Decisiones técnicas | Se documenta shard key, estrategia de particiones y alternativas descartadas |
| Comportamiento ante fallos | Se describen shard caído, scatter/gather fallido, directorio inconsistente y partition map inválido |
| Trade-offs y limitaciones | Se reconocen hot spots, scatter/gather, directorio crítico y falta de rebalanceo automático |
| Coherencia implementación-documento | Shard key, routing, directorio y scatter/gather coinciden con el código validado |

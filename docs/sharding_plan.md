# Plan de Implementación de Sharding en MedConnect Backend

**Bloque individual:** Sharding  
**Servicio propuesto:** `availability-service`  
**Estado:** plan aceptado de implementación. Este documento describe lo que se implementará y cómo se validará; no afirma que el bloque ya esté implementado.  
**Fuentes:** `rubrica_entrega2.md`, `README.md`, `AGENTS.md` y `Sharding y Consistent Hashing.pdf`.

---

## 1. Objetivo del bloque

El objetivo del bloque Sharding es dividir los datos de disponibilidad médica en múltiples bases de datos PostgreSQL, llamadas **shards**, para que los calendarios y slots no vivan todos en una única base.

El foco será `availability-service`, porque la disponibilidad se agrupa naturalmente por médico. Cada médico tiene una agenda propia, y las operaciones principales del servicio trabajan sobre `doctor_id`, `slot_id`, especialidad, rango de fechas y estado del slot (`available`, `held`, `booked`).

La rúbrica no exige shardear todo MedConnect. Exige que el bloque asignado funcione dentro del sistema real. Por eso, el alcance será un sharding real en `availability-service`, manteniendo el flujo backend existente a través del API Gateway, `booking-service` y gRPC.

---

## 2. Alcance

### Se shardea

- `availability-service`.
- Tablas de disponibilidad:
  - `doctor_calendars`.
  - `availability_slots`.
- Cada shard tendrá el mismo esquema lógico, pero solo una parte de los médicos y slots.

### No se shardea

- `auth-service`.
- `booking-service`.
- `payment-service`.
- `api-gateway`.
- `frontend/`.

Esto es intencional: el proyecto ya separa bases por servicio, pero sharding significa dividir internamente un dominio de datos. El dominio elegido es disponibilidad médica.

---

## 3. Shard key elegida

La shard key será:

```text
doctor_id
```

Razones:

1. La agenda médica pertenece naturalmente a un médico.
2. `GetDoctorAgenda` ya recibe `doctor_id`, por lo que puede resolverse en un solo shard.
3. Mantiene juntos el calendario y los slots de un mismo médico.
4. Reduce la necesidad de operaciones distribuidas para acciones críticas de disponibilidad.
5. Es más estable y defendible que usar fecha, especialidad o `slot_id`.

Flujo conceptual:

```text
doctor_id
  -> hash(doctor_id)
  -> partición lógica
  -> mapa de particiones
  -> shard físico
```

---

## 4. Técnica de particionamiento

Se usará:

```text
particionamiento por hash de clave + particiones lógicas fijas + partition map
```

En vez de usar directamente:

```text
hash(doctor_id) % cantidad_de_shards_físicos
```

se usará:

```text
partition_id = hash(doctor_id) % NUM_LOGICAL_PARTITIONS
shard_id = partitionMap[partition_id]
```

Ejemplo inicial para la entrega:

```text
NUM_LOGICAL_PARTITIONS = 16

partitionMap:
  0  -> shard0
  1  -> shard0
  2  -> shard0
  3  -> shard0
  4  -> shard0
  5  -> shard0
  6  -> shard0
  7  -> shard0
  8  -> shard1
  9  -> shard1
  10 -> shard1
  11 -> shard1
  12 -> shard1
  13 -> shard1
  14 -> shard1
  15 -> shard1
```

Esta técnica se basa en el material del curso: el PDF explica que `hash(key) % N` remapea muchos datos cuando cambia `N`. Con particiones lógicas fijas, el hash sigue apuntando a la misma partición y lo que puede cambiar en el futuro es qué shard físico posee cada partición.

No se implementará rebalanceo automático completo en esta entrega. Si se agregara un nuevo shard, habría que mover particiones completas y actualizar el `partitionMap`.

---

## 5. Routing de requests

El routing vivirá dentro de `availability-service`.

```text
api-gateway / booking-service
          -> availability-service
          -> router interno
          -> shard PostgreSQL correcto
```

Los consumidores externos seguirán usando:

```env
AVAILABILITY_SERVICE_TARGET=availability-service:50051
```

Esto mantiene estable el API Gateway y evita que `booking-service` conozca detalles del sharding.

En términos del PDF, `availability-service` actúa como una capa consciente del particionamiento: recibe la request, conoce el mapa de shards y enruta internamente al shard correcto.

---

## 6. Directorio `slot_id -> shard`

La decisión aceptada para esta entrega es usar un directorio interno:

```text
slot_id -> shard
```

Motivo: las operaciones actuales reciben solo `slot_id`:

- `HoldSlot`.
- `ConfirmSlotBooking`.
- `ReleaseHeldSlot`.

Como la shard key es `doctor_id`, estas operaciones no pueden calcular directamente el shard dueño solo con el proto actual.

El directorio permitirá resolver:

```text
slot_id -> doctor_id -> partition_id -> shard_id
```

Uso conceptual:

```text
HoldSlot(slot_id)
  -> buscar slot_id en directorio
  -> obtener shard dueño
  -> ejecutar UPDATE en ese shard
```

Ventajas para la entrega:

1. Evita cambiar contratos gRPC.
2. Mantiene `doctor_id` como shard key principal.
3. Permite enrutar operaciones críticas sin hacer scatter en cada mutación.
4. Es fácil de defender como metadata de routing.

Limitación: el directorio se vuelve metadata crítica. Si está desactualizado o caído, las operaciones por `slot_id` no pueden rutearse correctamente.

---

## 7. Flujos esperados

### 7.1 `GetDoctorAgenda`

Recibe `doctor_id`, por lo tanto va a un solo shard.

```text
GetDoctorAgenda(doctor_id, from_date, to_date)
  -> hash(doctor_id)
  -> partition_id
  -> shard_id
  -> consultar solo el shard dueño
```

Este es el caso ideal para sharding por clave.

### 7.2 `GetAvailableSlots` por especialidad

La especialidad no es la shard key. Médicos de la misma especialidad pueden estar distribuidos en varios shards.

Por eso se usará:

```text
scatter/gather
```

Flujo:

```text
GetAvailableSlots(specialty, from_date, to_date)
  -> consultar todos los shards
  -> filtrar localmente por specialty y rango de fechas
  -> combinar resultados
  -> ordenar por start_time
  -> retornar lista final
```

Este trade-off está alineado con el PDF: las búsquedas por índices secundarios no mapean naturalmente a una única partición.

### 7.3 `HoldSlot`

```text
HoldSlot(slot_id, booking_id)
  -> buscar slot_id en directorio
  -> obtener shard dueño
  -> UPDATE available -> held en ese shard
```

La actualización debe ser condicional para evitar doble reserva:

```sql
WHERE id = $slot_id
  AND status = 'available'
```

### 7.4 `ConfirmSlotBooking`

```text
ConfirmSlotBooking(slot_id, booking_id)
  -> buscar slot_id en directorio
  -> obtener shard dueño
  -> UPDATE held -> booked en ese shard
```

### 7.5 `ReleaseHeldSlot`

```text
ReleaseHeldSlot(slot_id, booking_id)
  -> buscar slot_id en directorio
  -> obtener shard dueño
  -> UPDATE held/booked -> available en ese shard
```

---

## 8. Comportamiento ante fallos

### Shard caído en operación directa

Si `GetDoctorAgenda` apunta al shard de un médico y ese shard está caído:

- la operación falla de forma explícita;
- no se consulta otro shard;
- otros médicos ubicados en shards sanos pueden seguir funcionando.

Esto demuestra degradación parcial.

### Shard caído en operación por `slot_id`

Si el directorio resuelve `slot_id -> shard1` y `shard1` está caído:

- `HoldSlot`, `ConfirmSlotBooking` o `ReleaseHeldSlot` fallan;
- no se escribe en otro shard;
- se evita inconsistencia.

### Scatter/gather con un shard caído

Para `GetAvailableSlots` por especialidad, si un shard falla durante scatter/gather, la consulta completa debe fallar.

Razón: el proto actual no tiene forma de indicar resultados parciales. En disponibilidad médica es preferible fallar explícitamente antes que mostrar una lista incompleta como si fuera completa.

### Partition map inválido

Si faltan particiones, un shard configurado no existe o el mapa es inconsistente, `availability-service` no debería iniciar. Es mejor fallar al arranque que enrutar datos médicos al shard incorrecto.

### Directorio inconsistente

Si el directorio apunta a un shard pero el slot no existe allí, se debe retornar error y registrar el problema. No se debe escribir en otro shard como fallback silencioso.

---

## 9. Hot spots y limitaciones

### Médico muy popular

El hash por `doctor_id` distribuye médicos entre shards, pero no elimina completamente los hot spots. Si un médico concentra muchas consultas o reservas, todas sus operaciones caen en el mismo shard.

Ejemplo:

```text
doctor_id muy demandado
  -> partition 6
  -> shard0
  -> shard0 recibe más carga
```

Esto coincide con el material del curso: el hash reduce skew promedio, pero no evita que una clave individual muy caliente genere un hot spot.

Trade-off aceptado: mantener toda la agenda de un médico en un solo shard simplifica operaciones críticas y evita que cada reserva deba coordinar múltiples bases.

### Scatter/gather

Las búsquedas por especialidad son más costosas que las búsquedas por doctor porque consultan todos los shards. Además, dependen del shard más lento.

### Directorio crítico

El directorio `slot_id -> shard` agrega metadata crítica que debe mantenerse consistente.

### Sin rebalanceo automático

La estrategia de particiones lógicas fijas permite explicar cómo se moverían particiones completas en el futuro, pero esta entrega no implementa migración ni rebalanceo automático.

---

## 10. Alternativas descartadas

### Shardear todo MedConnect

Descartado porque la rúbrica evalúa el bloque individual funcionando en el sistema real, no que todos los servicios estén shardeados. Shardear todo aumentaría mucho el riesgo y la complejidad.

### `auth-service`

Descartado porque el login depende de email y la unicidad global de email se complica en un sistema shardeado.

### `booking-service`

Descartado porque coordina reservas, disponibilidad y pagos. Shardearlo mezclaría sharding con problemas de consistencia distribuida más complejos.

### `payment-service`

Descartado porque es menos visible para demostrar distribución de datos y routing. Además, el dominio de pagos requiere especial cuidado con idempotencia y trazabilidad.

### Shard key `slot_id`

Descartada porque facilitaría mutaciones por slot, pero empeoraría `GetDoctorAgenda`: la agenda de un médico quedaría repartida y requeriría scatter/gather.

### Shard key `specialty`

Descartada porque especialidades populares podrían concentrar carga y generar skew.

### Shard por fecha

Descartado porque las fechas cercanas concentrarían writes, similar al problema de timestamp como clave mostrado en el PDF.

### `hash(doctor_id) % número_de_shards`

Descartado como estrategia principal porque al cambiar el número de shards físicos se remapearían muchos médicos. Se prefiere hash hacia particiones lógicas fijas.

### Consistent hashing completo con virtual nodes

Descartado para esta entrega por complejidad. La solución con particiones lógicas fijas cubre el objetivo académico y es más simple de implementar y defender.

---

## 11. Plan paso a paso de implementación

### Paso 0: Verificar baseline

```bash
cp .env.example .env
docker compose down -v
docker compose up -d --build api-gateway
docker compose ps
```

Validar el flujo del README antes de cambiar sharding:

1. Registrar usuario.
2. Login.
3. Consultar disponibilidad.
4. Crear reserva.
5. Crear pago.
6. Procesar pago.
7. Confirmar reserva.
8. Consultar reserva.

### Paso 1: Crear router de sharding

Nuevo paquete sugerido:

```text
availability-service/modules/sharding/
```

Responsabilidades:

- calcular `hash(doctor_id) % NUM_LOGICAL_PARTITIONS`;
- resolver `partition_id -> shard_id`;
- validar mapa de particiones;
- exponer todos los shards para scatter/gather.

Pruebas:

```bash
cd availability-service
go test ./modules/sharding/...
```

### Paso 2: Extender configuración

Modificar:

```text
availability-service/modules/config/config.go
.env.example
```

Variables sugeridas:

```env
AVAILABILITY_SHARDING_ENABLED=true
AVAILABILITY_PARTITION_COUNT=16
AVAILABILITY_SHARDS=shard0,shard1
AVAILABILITY_SHARD0_DSN=host=availability-db-shard-0 port=5432 user=postgres password=postgres dbname=availability_db sslmode=disable
AVAILABILITY_SHARD1_DSN=host=availability-db-shard-1 port=5432 user=postgres password=postgres dbname=availability_db sslmode=disable
AVAILABILITY_PARTITION_MAP=0:shard0,1:shard0,2:shard0,3:shard0,4:shard0,5:shard0,6:shard0,7:shard0,8:shard1,9:shard1,10:shard1,11:shard1,12:shard1,13:shard1,14:shard1,15:shard1
```

Mantener fallback a la DB única actual para no romper desarrollo local.

### Paso 3: Crear múltiples DBs de availability

Modificar `docker-compose.yml` para reemplazar o complementar `availability-db` con:

```text
availability-db-shard-0
availability-db-shard-1
```

Cada shard debe tener:

- volumen propio;
- mismo `availability-service/db/init.sql`;
- seed propio;
- healthcheck con `pg_isready`.

### Paso 4: Separar seeds

Crear seeds por shard:

```text
availability-service/db/seed_shard0.sql
availability-service/db/seed_shard1.sql
```

Criterio:

```text
un doctor nunca debe aparecer repartido entre dos shards
```

### Paso 5: Implementar directorio `slot_id -> shard`

Puede implementarse como tabla de metadata o estructura cargada al iniciar, según el menor cambio seguro para la entrega.

Debe permitir:

```text
slot_id -> shard_id
```

y registrar logs cuando se use:

```text
slot_directory slot_id=... shard=...
```

### Paso 6: Crear repositorio shardeado

Nuevo archivo sugerido:

```text
availability-service/modules/repository/sharded.go
```

Responsabilidades:

- mantener una conexión/pool por shard;
- enrutar por `doctor_id`;
- enrutar por `slot_id` usando el directorio;
- ejecutar scatter/gather para búsquedas por especialidad;
- devolver errores claros ante shard caído.

### Paso 7: Ajustar queries PostgreSQL

Modificar `availability-service/modules/repository/postgres.go` para:

- revisar `rows.Err()`;
- usar contexto si se incorpora `QueryContext`/`QueryRowContext`;
- soportar lectura de datos necesarios para el directorio;
- mantener actualizaciones condicionales por estado.

### Paso 8: Conectar modo sharded en `main.go`

Modificar `availability-service/main.go`:

```text
si AVAILABILITY_SHARDING_ENABLED=true:
    crear router
    abrir conexión a cada shard
    crear ShardedRepository
si no:
    usar PostgresRepository actual
```

### Paso 9: Agregar logs demostrables

Logs sugeridos:

```text
sharding: doctor_id=... partition=... shard=...
sharding: scatter/gather specialty=... shards=2
sharding: slot_id=... routed_by_directory shard=...
```

Estos logs servirán como evidencia para el video.

### Paso 10: Probar módulos Go

Ejecutar tests por módulo, no desde la raíz:

```bash
cd availability-service && go test ./...
cd ../booking-service && go test ./...
cd ../api-gateway && go test ./...
```

Opcional:

```bash
cd ../auth-service && go test ./...
cd ../payment-service && go test ./...
```

### Paso 11: Probar end-to-end con Docker

```bash
docker compose down -v
docker compose up -d --build api-gateway
docker compose ps
```

Luego ejecutar los cURL del README para validar el flujo completo.

### Paso 12: Probar fallo de shard

```bash
docker compose stop availability-db-shard-1
```

Validar:

- doctor en shard sano responde;
- doctor en shard caído falla claro;
- scatter/gather falla completo;
- no se devuelven resultados parciales silenciosos.

Recuperar:

```bash
docker compose start availability-db-shard-1
```

---

## 12. Casos de borde a cubrir

| Caso | Resultado esperado |
|---|---|
| `doctor_id` vacío | Error de validación |
| `slot_id` vacío | Error de validación |
| `from_date > to_date` | Error de validación |
| `slot_id` no existe en directorio | `NotFound` |
| `slot_id` existe en directorio pero no en shard | Error de inconsistencia |
| Dos reservas bloquean el mismo slot | Solo una cambia `available -> held` |
| Consulta por especialidad | Scatter/gather |
| Shard caído en consulta directa | Error controlado |
| Shard caído en scatter/gather | Falla consulta completa |
| Médico con demasiada demanda | Posible hot spot documentado |

---

## 13. Evidencia para validación y video

### Mostrar shards levantados

```bash
docker compose ps
```

Debe verse:

```text
availability-service
availability-db-shard-0
availability-db-shard-1
```

### Mostrar datos distribuidos

```bash
docker compose exec -T availability-db-shard-0 psql -U postgres -d availability_db \
  -c "SELECT c.doctor_id, c.specialty, count(s.id) AS slots FROM doctor_calendars c LEFT JOIN availability_slots s ON s.calendar_id = c.id GROUP BY c.doctor_id, c.specialty;"

docker compose exec -T availability-db-shard-1 psql -U postgres -d availability_db \
  -c "SELECT c.doctor_id, c.specialty, count(s.id) AS slots FROM doctor_calendars c LEFT JOIN availability_slots s ON s.calendar_id = c.id GROUP BY c.doctor_id, c.specialty;"
```

### Mostrar routing por doctor

Ejecutar consulta de agenda médica y mostrar logs:

```text
sharding: doctor_id=... partition=... shard=...
```

### Mostrar scatter/gather

Ejecutar consulta por especialidad y mostrar logs:

```text
sharding: scatter/gather specialty=Cardiología shards=2
```

### Mostrar reserva usando directorio

Crear una reserva desde API Gateway y mostrar logs:

```text
sharding: slot_id=... routed_by_directory shard=...
```

### Mostrar fallo controlado

Detener un shard y explicar:

- las operaciones del shard caído fallan;
- las de shards sanos pueden seguir funcionando;
- scatter/gather falla completo para evitar datos parciales.

---

## 14. Guion breve para video individual

> Mi bloque es Sharding y lo apliqué en `availability-service`, no en todo MedConnect. Esto es intencional porque la rúbrica pide que el bloque funcione dentro del sistema real, y disponibilidad es el dominio más natural para particionar: los calendarios y slots pertenecen a médicos.
>
> La shard key elegida es `doctor_id`. Uso particionamiento por hash de clave, pero no hago `hash(doctor_id) % número de shards`, porque eso remapea muchos datos si cambia la cantidad de shards. En su lugar uso `hash(doctor_id) % 16` para obtener una partición lógica, y luego un mapa asigna esa partición a un shard físico.
>
> El routing queda dentro de `availability-service`. El API Gateway y `booking-service` no saben que hay shards. `GetDoctorAgenda` va directo al shard del médico. En cambio, `GetAvailableSlots` por especialidad usa scatter/gather porque `specialty` no es la shard key.
>
> Para `HoldSlot`, `ConfirmSlotBooking` y `ReleaseHeldSlot` hay un problema: solo reciben `slot_id`. Para esta entrega uso un directorio `slot_id -> shard`, que permite encontrar el shard dueño sin cambiar el contrato gRPC.
>
> La principal limitación es que un médico muy demandado puede generar un hot spot, y las consultas por especialidad dependen del shard más lento. Aun así, el diseño demuestra sharding real, routing, múltiples shards, manejo de fallos y coherencia con el flujo backend de MedConnect.

---

## 15. Relación directa con la rúbrica

### Documento del bloque

Este plan cubre:

- qué hace Sharding en MedConnect;
- cómo encaja en el sistema real;
- qué servicios involucra;
- por qué se shardea `availability-service` y no todo el sistema.

### Decisiones técnicas

Decisión principal:

```text
doctor_id + hash key + particiones lógicas fijas + partition map
```

Alternativas descartadas documentadas:

- `auth-service`;
- `booking-service`;
- `payment-service`;
- `slot_id` como shard key;
- `specialty` como shard key;
- particionamiento por fecha;
- `hash(key) % N` directo;
- consistent hashing completo con virtual nodes.

### Comportamiento ante fallos

Se documentan fallos de:

- shard caído;
- scatter/gather parcial;
- directorio inconsistente;
- partition map inválido.

### Trade-offs y limitaciones

Se reconocen limitaciones reales:

- hot spot por médico popular;
- scatter/gather para especialidad;
- directorio como metadata crítica;
- sin rebalanceo automático completo.

### Coherencia con implementación

La implementación posterior debe coincidir con este documento en:

- servicio shardeado: `availability-service`;
- shard key: `doctor_id`;
- técnica: hash + particiones lógicas fijas + partition map;
- routing interno;
- directorio `slot_id -> shard`;
- scatter/gather para especialidad;
- manejo explícito de errores.

# Reporte vivo de implementación · Sharding MedConnect Backend

**Bloque individual:** Sharding  
**Servicio objetivo:** `availability-service`  
**Estado actual:** baseline inicial revisado; sharding aún no implementado  
**Base:** `docs/sharding_plan.md`, `rubrica_entrega2.md`, `README.md`, `AGENTS.md`  
**Última actualización:** 2026-06-29

---

## 1. Propósito del reporte

Este documento registra el avance real de la implementación de Sharding en MedConnect backend. Debe reflejar solo lo efectivamente realizado: pasos ejecutados, comandos usados, resultados obtenidos, decisiones tomadas, problemas encontrados y pendientes.

> Importante: por ahora Sharding no está implementado. Este reporte parte como bitácora inicial y se irá actualizando en cada paso.

---

## 2. Resumen del diseño aprobado

- Se shardea solo `availability-service`.
- No se shardean `auth-service`, `booking-service`, `payment-service` ni `api-gateway`.
- Shard key: `doctor_id`.
- Técnica: hash por clave + particiones lógicas fijas + `partitionMap`.
- Operaciones por `slot_id` usarán un directorio interno `slot_id -> shard`.
- `GetDoctorAgenda` irá a un solo shard.
- `GetAvailableSlots` por especialidad usará `scatter/gather`.
- El routing vivirá dentro de `availability-service`.

---

## 3. Responsabilidades delegadas

| Responsabilidad | Subagente / rol | Estado |
|---|---|---|
| Verificar baseline backend antes de tocar código | `sharding-verifier` | Completado parcialmente |
| Diseñar estructura de reporte y documento técnico | `sharding-documenter` | Completado |
| Implementar cambios backend controlados | `sharding-implementer` | Pendiente |
| Validar build, tests, Docker y flujo backend | `sharding-verifier` | Pendiente por cada paso |
| Mantener coherencia con rúbrica y video | `sharding-documenter` | En curso |

---

## 4. Bitácora de avance

| Fecha | Paso | Comando / cambio | Resultado | Estado |
|---|---|---|---|---|
| 2026-06-29 | Diseño aceptado | `docs/sharding_plan.md` | Plan de sharding aprobado por el usuario | Completado |
| 2026-06-29 | Decisión de routing por slot | Conversación de diseño | Se acepta directorio `slot_id -> shard` para esta entrega | Completado |
| 2026-06-29 | Verificación baseline | `docker compose ps` | No hay contenedores corriendo | Completado |
| 2026-06-29 | Verificación baseline | `docker compose ps -a` | Existen contenedores antiguos detenidos | Completado |
| 2026-06-29 | Tests Go | `cd availability-service && go test ./...` | PASS; no hay tests reales en el módulo | Completado |
| 2026-06-29 | Tests Go | `cd booking-service && go test ./...` | PASS | Completado |
| 2026-06-29 | Tests Go | `cd auth-service && go test ./...` | PASS | Completado |
| 2026-06-29 | Tests Go | `cd payment-service && go test ./...` | PASS; sin tests reales en el módulo | Completado |
| 2026-06-29 | Tests Go | `cd api-gateway && go test ./...` | FAIL preexistente en `internal/http` por autenticación requerida | Bloqueante parcial |
| 2026-06-29 | Documentación viva | Crear `docs/sharding_implementation_report.md` | Reporte inicial creado | Completado |
| 2026-06-29 | Documento técnico vivo | Crear `docs/sharding_technical_doc.md` | Borrador técnico inicial creado | Completado |
| 2026-06-29 | Baseline Docker | `docker compose up -d --build api-gateway` | Imágenes backend construidas y servicios iniciados | Completado |
| 2026-06-29 | Baseline Docker | `docker compose ps` | `api-gateway`, servicios backend y DBs aparecen `Up`; DBs healthy | Completado |
| 2026-06-29 | Baseline REST | `POST /auth/register` y `POST /auth/login` | Registro y login funcionan tras estabilizar servicios | Completado |
| 2026-06-29 | Baseline REST | `GET /availability/slots?specialty=Traumatología&from_date=...&to_date=...` | Retorna slot disponible | Completado |
| 2026-06-29 | Baseline REST | `GET /availability/doctors/{doctor_id}?from_date=...&to_date=...` | Retorna agenda del médico | Completado |
| 2026-06-29 | Baseline REST | `POST /bookings` | Falla con `502`; dependencia externa `HoldSlot` termina en `DeadlineExceeded` | Bloqueante parcial |
| 2026-06-29 | Diagnóstico HoldSlot | Revisión delegada + pruebas directas `/availability/hold` | El slot fijo del README estaba `held`; slots disponibles funcionan. Se detectó que availability no persistía `booking_id`/`held_until` ni propagaba `context.Context` a SQL | Completado |
| 2026-06-29 | Corrección HoldSlot | Cambios en `availability-service` handler/service/repository | Se propaga contexto, se persisten `booking_id` y `held_until`, y slot no disponible retorna `FailedPrecondition` | Completado |
| 2026-06-29 | Tests post-corrección | `cd availability-service && go test ./...` | PASS | Completado |
| 2026-06-29 | Tests post-corrección | `cd booking-service && go test ./...` | PASS | Completado |
| 2026-06-29 | Tests post-corrección | `cd api-gateway && go test ./...` | FAIL preexistente por tests sin auth | Bloqueante parcial |
| 2026-06-29 | Docker post-corrección | `docker compose up -d --build availability-service` | Imagen reconstruida y contenedor reiniciado | Completado |
| 2026-06-29 | Verificación post-corrección | `/availability/hold` con UUID válido + `/availability/release` | Hold persiste metadata y release restaura slot | Completado |
| 2026-06-29 | Verificación post-corrección | `/availability/hold` sobre slot ya `held` | Retorna HTTP `409` en vez de error interno | Completado |
| 2026-06-29 | Verificación post-corrección | `POST /bookings` con slot disponible y posterior cancelación | Reserva retorna `201` y cancelación retorna `200` | Completado |
| 2026-06-29 | Router de sharding | Crear `availability-service/modules/sharding` | Router por `doctor_id -> CRC32 -> partición lógica -> shard` implementado | Completado |
| 2026-06-29 | Tests router | `cd availability-service && go test ./modules/sharding/... && go test ./...` | PASS | Completado |
| 2026-06-29 | Configuración sharding | Extender `availability-service/modules/config/config.go` | Carga `AVAILABILITY_SHARDING_*` con fallback a DB única | Completado |
| 2026-06-29 | Tests configuración | `cd availability-service && go test ./modules/config/... && go test ./...` | PASS | Completado |

---

## 5. Estado actual observado

- Al inicio de la verificación no existía `.env` en la raíz; sí existía `.env.example`.
- Docker está disponible.
- El stack backend fue levantado posteriormente y quedó activo.
- Antes de levantar el stack había contenedores antiguos detenidos del proyecto.
- `docker-compose.yml` todavía define una sola base de disponibilidad: `availability-db`.
- `availability-service` todavía usa una sola conexión PostgreSQL.
- No existen variables `AVAILABILITY_SHARDING_*` en el código actual.
- El stack backend fue levantado con `docker compose up -d --build api-gateway`.
- `docker compose ps` mostró servicios backend activos y DBs healthy.
- `availability-service/modules/config/config.go` solo carga:
  - `DB_HOST`
  - `DB_PORT`
  - `DB_USER`
  - `DB_PASSWORD`
  - `DB_NAME`
  - `GRPC_PORT`

---

## 6. Resultado de tests iniciales

| Módulo | Comando | Resultado | Observación |
|---|---|---|---|
| `availability-service` | `go test ./...` | PASS | No hay archivos de test reales |
| `booking-service` | `go test ./...` | PASS | Baseline unitario correcto |
| `auth-service` | `go test ./...` | PASS | Baseline unitario correcto |
| `payment-service` | `go test ./...` | PASS | Sin tests reales |
| `api-gateway` | `go test ./...` | FAIL | Tests esperan respuestas sin auth, pero gateway exige token |

### Falla preexistente en `api-gateway`

Errores observados:

```text
expected status 201, got 401: {"error":"token de acceso requerido"}
expected status 200, got 401: {"error":"token de acceso requerido"}
expected status 409, got 401: {"error":"token de acceso requerido"}
```

Interpretación: la falla parece estar relacionada con pruebas desactualizadas frente al middleware de autenticación actual. Debe registrarse para no atribuirla erróneamente al sharding.

---

## 7. Resultado baseline REST inicial

### Servicios levantados

El usuario levantó el backend con:

```bash
docker compose up -d --build api-gateway
docker compose ps
```

Evidencia observada:

- `medconnect-api-gateway-1`: `Up`, puerto `8080` publicado.
- `availability-service`: `Up`.
- `medconnect-booking-service-1`: `Up`.
- `medconnect-payment-service-1`: `Up`.
- `medconnect-auth-service-1`: `Up`.
- `availability-db`, `booking_db`, `payments-db`, `auth_db`: `healthy`.

### Registro y login

Resultado posterior a la estabilización de servicios:

```text
POST /auth/register -> 201
POST /auth/login    -> 200
```

### Disponibilidad

El README usa `start_date`/`end_date`, pero el handler actual de `api-gateway` exige `from_date`/`to_date`.

Consulta válida:

```bash
GET /availability/slots?specialty=Traumatología&from_date=2026-01-01T00:00:00Z&to_date=2027-12-31T00:00:00Z
```

Resultado:

```text
200 OK, retorna slot disponible de Traumatología
```

Consulta de agenda:

```bash
GET /availability/doctors/7e0d2ab1-164e-4a28-8b95-f24293dd0e91?from_date=2026-01-01T00:00:00Z&to_date=2027-12-31T00:00:00Z
```

Resultado:

```text
200 OK, retorna agenda del médico
```

### Flujo de reserva

Al intentar crear una reserva con un slot disponible:

```text
POST /bookings -> 502
```

Error observado:

```text
rpc error: code = Unavailable desc = fallo en dependencia externa: hold slot: rpc error: code = DeadlineExceeded desc = context deadline exceeded
```

Interpretación inicial: el baseline E2E todavía no está completamente sano. La consulta de disponibilidad funciona, pero el flujo de reserva falla cuando `booking-service` intenta ejecutar `HoldSlot` en `availability-service`.

Este problema debe registrarse como preexistente al sharding y revisarse antes de la validación final de implementación.

---

## 8. Diagnóstico y corrección de `HoldSlot`

### Diagnóstico

Se delegó la revisión a `sharding-implementer` y `sharding-verifier`.

Hallazgos principales:

1. El slot fijo del README (`0f5c2b6a-1a87-4b7e-ae2c-37ef2f9f1c21`) ya estaba en estado `held` por una reserva antigua `PENDING_PAYMENT`.
2. `/availability/hold` directo falla correctamente si el slot está `held`.
3. `/bookings` sí funciona cuando se usa un slot realmente `available`.
4. `availability-service` recibía `context.Context` en el handler gRPC, pero no lo propagaba hacia service/repository/SQL.
5. `HoldSlot` ignoraba `booking_id` y `held_until`, aunque el proto ya los trae.
6. Los errores de slot no disponible se devolvían como `Internal`, lo que terminaba en HTTP 500/502 en vez de conflicto de estado.

### Corrección aplicada

Archivos modificados en `availability-service`:

- `modules/handler/grpc_handler.go`
- `modules/service/service.go`
- `modules/repository/repository.go`
- `modules/repository/postgres.go`
- `modules/repository/models.go`

Cambios:

- Se propagó `context.Context` desde gRPC hasta PostgreSQL.
- Se reemplazaron consultas por variantes con contexto (`QueryContext`, `QueryRowContext`).
- `HoldSlot` ahora persiste `booking_id` y `held_until` en `availability_slots`.
- `ConfirmSlotBooking` y `ReleaseHeldSlot` usan `booking_id` para evitar confirmar/liberar slots de otra reserva.
- Se agregó error de dominio `ErrSlotNotAvailable`.
- Slot no disponible ahora se mapea a gRPC `FailedPrecondition`, que el API Gateway traduce a HTTP `409 Conflict`.
- Se revisa `rows.Err()` al iterar resultados.

### Verificación post-corrección

Comandos ejecutados:

```bash
cd availability-service && go test ./...
cd ../booking-service && go test ./...
cd ../api-gateway && go test ./...
docker compose up -d --build availability-service
```

Resultados:

| Verificación | Resultado |
|---|---|
| Tests `availability-service` | PASS |
| Tests `booking-service` | PASS |
| Tests `api-gateway` | FAIL preexistente por tests sin auth |
| Rebuild `availability-service` | OK |
| `/availability/hold` con UUID válido | `200 OK` |
| `/availability/release` con mismo UUID | `200 OK` |
| `/availability/hold` sobre slot ya `held` | `409 Conflict` |
| `POST /bookings` con slot disponible | `201 Created` |
| Cancelación de reserva de prueba | `200 OK`, slot restaurado |

Estado final de slots tras la prueba:

```text
Cardiología      -> held      (estado preexistente del slot fijo del README)
Medicina interna -> available
Traumatología    -> available
```

---

## 9. Comandos pendientes para baseline E2E

Estos comandos aún no se ejecutaron en esta etapa para evitar modificar estado o borrar volúmenes sin necesidad.

### Preparar entorno local

```bash
cp .env.example .env
```

### Levantar solo backend

```bash
docker compose up -d --build api-gateway
docker compose ps
```

### Ver logs backend

```bash
docker compose logs api-gateway auth-service availability-service booking-service payment-service
```

### Reset limpio si se cambian SQL o seeds

```bash
docker compose down -v
```

Usar este comando solo cuando sea necesario, porque borra volúmenes.

---

## 10. Riesgos antes de implementar router

1. Baseline end-to-end aún no está comprobado porque los servicios están detenidos.
2. No existe `.env` raíz, aunque Docker Compose tiene defaults funcionales.
3. Tests de `api-gateway` fallan por autenticación requerida.
4. `availability-service` no tiene tests reales; habrá que agregar pruebas para sharding.
5. Hay contenedores antiguos detenidos con exit code distinto de cero en algunos servicios.
6. Debe evitarse tocar `frontend/`.
7. El slot fijo del README está `held` por datos persistidos antiguos; para pruebas repetibles conviene usar slots disponibles o resetear volúmenes.
8. README y handler difieren en nombres de query params para disponibilidad: README usa `start_date`/`end_date`; el código exige `from_date`/`to_date`.

---

## 11. Próximos pasos

- [x] Ejecutar baseline Docker con `docker compose up -d --build api-gateway`.
- [x] Registrar evidencia inicial del flujo actual antes de sharding.
- [x] Investigar y corregir comportamiento baseline de `HoldSlot`.
- [x] Implementar paquete `availability-service/modules/sharding`.
- [x] Agregar tests del router.
- [x] Integrar configuración `AVAILABILITY_SHARDING_*` con fallback a DB única.
- [ ] Actualizar este reporte con comandos, resultados y decisiones.
- [ ] Actualizar `docs/sharding_technical_doc.md` con evidencia relevante.

---

## 12. Implementación del router de sharding

Se implementó el paquete:

```text
availability-service/modules/sharding
```

Archivos creados:

```text
availability-service/modules/sharding/router.go
availability-service/modules/sharding/router_test.go
```

Responsabilidades implementadas:

- `NewRouter(partitionCount, partitionMap)` valida que todas las particiones lógicas tengan shard asignado.
- `ShardForDoctor(doctorID)` calcula una ruta estable usando `crc32.ChecksumIEEE`.
- `ShardForPartition(partition)` resuelve una partición lógica ya calculada.
- `AllShards()` retorna shards únicos, ordenados y deterministas.

Decisión técnica aplicada:

```text
partition = crc32(doctor_id) % partitionCount
shard = partitionMap[partition]
```

Esto implementa la primera pieza del diseño aprobado: particionamiento por hash de clave hacia particiones lógicas fijas.

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

Casos cubiertos por tests:

- determinismo para el mismo `doctor_id`;
- rechazo de `doctor_id` vacío;
- rechazo de `partitionCount <= 0`;
- rechazo de particiones faltantes;
- rechazo de shard vacío;
- rechazo de particiones fuera de rango;
- `AllShards()` ordenado y sin duplicados;
- copia defensiva del `partitionMap`.

---

## 13. Integración de configuración de sharding

Se extendió la configuración de `availability-service` para soportar modo sharded sin romper el modo actual de DB única.

Archivos modificados/creados:

```text
availability-service/modules/config/config.go
availability-service/modules/config/config_test.go
availability-service/.env.example
```

Variables soportadas:

```env
AVAILABILITY_SHARDING_ENABLED=true
AVAILABILITY_PARTITION_COUNT=16
AVAILABILITY_SHARDS=shard0,shard1
AVAILABILITY_PARTITION_MAP=0:shard0,1:shard1,2:shard0,3:shard1
AVAILABILITY_SHARD0_DSN=...
AVAILABILITY_SHARD1_DSN=...
```

Decisiones:

- Si `AVAILABILITY_SHARDING_ENABLED` está ausente, el servicio sigue en modo DB única.
- `DSN()` se mantiene compatible con las variables actuales `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`.
- Si sharding está habilitado y no se define `AVAILABILITY_PARTITION_COUNT`, se usa `16` como default.
- `ValidateShardingConfig()` valida el mapa usando el router de `modules/sharding` para no duplicar reglas.

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

Casos cubiertos por tests:

- modo single DB sin variables de sharding;
- modo sharded válido;
- default de `16` particiones cuando sharding está habilitado;
- partition map incompleto;
- DSN faltante para un shard declarado;
- booleano inválido;
- partition count inválido.

Pendiente: esta configuración todavía no se conecta al `main.go` ni al repositorio shardeado. Ese será un paso posterior.

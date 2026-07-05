# Notas para agentes en MedConnect

## Alcance y conducta
- Trata el trabajo actual como **solo backend**; ignora `frontend/` salvo que el usuario lo pida explícitamente. `docker-compose.yml` aún define un servicio frontend, así que prefiere comandos Compose dirigidos al backend.
- Es un proyecto de curso: preserva el aprendizaje. Explica cambios no triviales y mantén la implementación incremental en vez de entregar soluciones opacas completas.
- La configuración local de OpenCode vive en `.opencode/opencode.json`; el agente primario por defecto es `ejecutor-medconnect`, con subagentes de sharding en `.opencode/agent/`.
- Para commits, sigue Conventional Commits: encabezado breve con tipo/alcance e información general, y cuerpo con detalle útil pero no extenso. Siempre espera validación del mensaje antes de hacer commit o push.

## Arquitectura backend
- No hay `go.mod` ni `go.work` en la raíz; cada servicio backend es su propio módulo Go: `api-gateway`, `auth-service`, `availability-service`, `booking-service`, `payment-service`.
- El único punto público es el API Gateway REST en `localhost:8080`; las llamadas entre servicios son gRPC dentro de la red interna de Docker.
- Entrypoints reales: `api-gateway/cmd/api-gateway/main.go`, `auth-service/cmd/auth-service/main.go`, `booking-service/cmd/booking-service/main.go`, `availability-service/main.go`, `payment-service/main.go`.
- Los contratos Protobuf están en `pb/*.proto` dentro de cada servicio; no edites a mano los generados `*.pb.go` / `*_grpc.pb.go`.
- Los SQL de inicialización de Docker solo corren al crear el volumen por primera vez. Tras cambiar migraciones o `db/*.sql`, reinicia con `docker compose down -v` antes de reconstruir.

## Comandos fáciles de adivinar mal
- Entorno inicial: `cp .env.example .env`.
- Stack solo backend: `docker compose up -d --build api-gateway` levanta el gateway y sus dependencias backend sin iniciar `frontend`.
- Stack completo, solo si se pide explícitamente: `docker compose up -d --build`.
- Estado/logs/reset: `docker compose ps`, `docker compose logs -f`, `docker compose down -v`.
- Ejecuta tests Go dentro de cada módulo, no desde la raíz: `go test ./...` con workdir en `api-gateway/`, `auth-service/`, `availability-service/`, `booking-service/` o `payment-service/`.
- Ejemplos enfocados: `go test ./internal/service/...` en `auth-service/`; `go test ./internal/service/...` o `go test ./internal/repository/postgres/...` en `booking-service/`.
- La verificación manual de API usa `Insomnia_2026-05-08.yaml` o los cURL del README; el README actualmente menciona un nombre de Insomnia más antiguo.

## Guía para el módulo Sharding
- Usa `rubrica_entrega2.md` como fuente de evaluación: pide que el bloque asignado funcione en el sistema real, no que todos los servicios estén shardeados.
- Prefiere `availability-service` para sharding salvo que aparezca evidencia en contra: la disponibilidad se agrupa naturalmente por médico, y `doctor_id` ya existe en slots/calendarios y requests como `GetDoctorAgenda`.
- Desde `Sharding y Consistent Hashing.pdf`, alinea terminología y trade-offs con: particionamiento por hash/clave, skew/hot spots, scatter/gather, request routing, consistent hashing y virtual nodes al hablar de rebalanceo.
- Sé explícito: `hash(key) % N` es simple para una demo local fija, pero remapea muchos datos cuando cambia `N`; documenta esa limitación o usa particiones fijas/consistent hashing si implementas rebalanceo.
- Si `availability-service` se shardea por `doctor_id`, `GetDoctorAgenda` puede ir a un solo shard; `GetAvailableSlots` por `specialty` puede requerir scatter/gather salvo que exista índice secundario/directorio.
- `HoldSlot`, `ConfirmSlotBooking` y `ReleaseHeldSlot` reciben solo `slot_id` en el proto actual, así que un diseño shardeado necesita una forma verificada de enrutar `slot_id` al shard dueño: directorio, shard codificado, scatter fallback o cambio de proto/API.
- Documenta alternativas descartadas: `auth-service` complica unicidad global de email/login, `booking-service` coordina pagos y disponibilidad, y `payment-service` es menos visible para demostrar distribución de datos.

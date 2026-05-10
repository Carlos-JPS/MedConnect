# MedConnect

**MedConnect** es una plataforma distribuida de gestión de citas médicas diseñada para pacientes y clínicas. Permite a los usuarios buscar médicos, visualizar horarios disponibles, bloquear horas y procesar pagos. La arquitectura está construida sobre un ecosistema de **microservicios independientes**, lo que garantiza alta disponibilidad, tolerancia a fallos y escalabilidad horizontal, separando responsabilidades operativas como Autenticación, Reservas, Disponibilidad de agendas y Transacciones de pago.

## Arquitectura Actual

La arquitectura implementa el patrón **API Gateway (BFF - Backend For Frontend)**. Los clientes externos se comunican exclusivamente vía HTTP/REST con el API Gateway. Este actúa como traductor y enrutador, comunicándose con los microservicios internos utilizando **gRPC**. La red interna está aislada en Docker y cada microservicio posee su propia instancia de base de datos PostgreSQL independiente (Patrón Database-per-service).

- **api-gateway**: Única entrada HTTP externa. Expone endpoints REST de autenticación, disponibilidad, reservas y pagos, traduciéndolos a gRPC.
- **auth-service**: Servicio gRPC. Maneja registro, login y validación de tokens JWT.
- **availability-service**: Servicio gRPC. Maneja las agendas médicas y la disponibilidad de slots.
- **booking-service**: Servicio gRPC. Persiste intenciones de citas y orquesta con disponibilidad y pagos.
- **payment-service**: Servicio gRPC. Persiste pagos y transacciones.
- **Bases de datos**: `auth_db`, `availability-db`, `booking_db`, `payments_db` (Instancias PostgreSQL independientes).

*(Nota: El desarrollo de la interfaz de usuario / Frontend está marcado como To-Do y se conectará directamente a este Gateway en el futuro).*

## Casos de Uso y Flujos de Comunicación

1. **Búsqueda de disponibilidad (Lectura)**
   - **Usuario:** El paciente busca en la plataforma las horas disponibles de un médico específico.
   - **Flujo Técnico:** El cliente envía una petición REST (`GET /availability/doctors/.../agenda`) al API Gateway. El Gateway traduce la petición a gRPC hacia el `availability-service`. Este servicio consulta su base de datos aislada (`availability-db`) y retorna los slots al Gateway, que responde con JSON al cliente. Ningún otro servicio interviene.

2. **Bloqueo de Reserva (Escritura temporal)**
   - **Usuario:** El paciente selecciona un bloque de tiempo y avanza al pago. El sistema retiene la hora temporalmente.
   - **Flujo Técnico:** El API Gateway recibe un `POST /bookings` y llama vía gRPC a `booking-service`. Antes de crear la reserva, `booking-service` llama gRPC a `availability-service` (`HoldSlot`) para cambiar el estado a "held" y evitar colisiones. Luego, `booking-service` guarda la reserva en `booking_db` con estado "pending".

3. **Procesamiento de Pago**
   - **Usuario:** El paciente ingresa sus datos y completa el pago de la consulta médica.
   - **Flujo Técnico:** Petición REST al API Gateway que se rutea vía gRPC al `payment-service`. Este servicio valida la transacción, persiste en `payments_db` y retorna éxito o rechazo.

4. **Confirmación Definitiva (Sincronización)**
   - **Usuario:** Tras pagar, el paciente recibe la confirmación final de que su hora fue agendada exitosamente.
   - **Flujo Técnico:** Tras un pago exitoso, se llama a confirmar la reserva (`POST /bookings/.../confirm`). `booking-service` valida con `payment-service`, actualiza su estado a "confirmed" en `booking_db` y realiza una llamada gRPC a `availability-service` (`ConfirmSlotBooking`) mutando el slot de "held" a "booked".

## Decisiones Técnicas y Trade-offs

- **Patrón Database-per-service (Bases de datos separadas)**
  - *Justificación:* Garantiza bajo acoplamiento y aísla fallas. Si la BD de pagos cae, la búsqueda de horas médicas sigue funcionando perfectamente.
  - *Trade-off:* Complica las transacciones distribuidas. Obliga a manejar consistencia eventual y compensaciones (ej: liberar un slot si el pago falla o expira).

- **Comunicación interna mediante gRPC vs REST**
  - *Justificación:* gRPC con Protobuf ofrece serialización binaria ultrarrápida, contratos estrictos y menor latencia para tráfico de máquina a máquina.
  - *Trade-off:* Curva de aprendizaje más alta y pérdida de legibilidad humana directa de payloads en la red, requiriendo herramientas especiales para debug.

- **Uso de API Gateway único**
  - *Justificación:* Oculta la complejidad de los microservicios, unifica la exposición externa, y traduce protocolos automáticamente.
  - *Trade-off:* Es un Single Point of Failure (SPOF) y añade latencia marginal por el salto de red adicional.

## Guía Operacional: Levantar el Sistema

El proyecto está completamente Dockerizado. No es necesario instalar Go localmente.

### 1. Configuración Inicial
Copia el archivo de entorno base:
```bash
cp .env.example .env
```
*(Los valores por defecto de `.env.example` son completamente funcionales para el ambiente de desarrollo local).*

### 2. Iniciar Servicios
Levanta toda la infraestructura (Gateway, 4 Microservicios, 4 BDs):
```bash
docker compose up -d --build
```

- **Ver contenedores:** `docker compose ps`
- **Ver logs:** `docker compose logs -f`
- **Detener y borrar datos:** `docker compose down -v`

### 3. Puertos Expuestos al Host
- **API Gateway (REST):** `http://localhost:8080`
*(Los servicios gRPC y las BD operan de manera segura y privada solo en la red interna de Docker `medconnect_internal`).*

## Pruebas de Endpoints (API Gateway)

La forma recomendada de probar el sistema es utilizando la colección de Insomnia.

### Pruebas con Insomnia (Recomendado)
En la raíz del repositorio se encuentra el archivo `Insomnia_2026-05-07.yaml`.
1. Importa este archivo en tu cliente **Insomnia**.
2. Contiene carpetas ordenadas para **Auth**, **Availability**, **Bookings** y **Payments**.
3. Todas las peticiones apuntan a `http://localhost:8080` (API Gateway).

---

### Pruebas Manuales (cURL)

<details>
<summary>Ver comandos cURL para todos los endpoints</summary>

#### Autenticación

**Registro de Usuario**
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

**Inicio de Sesión**
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "paciente@test.com",
    "password": "password123"
  }'
```

#### Booking & Availability

**Crear Reserva (Requiere Availability Service)**
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

**Listar Reservas de Paciente**
```bash
curl "http://localhost:8080/bookings?patient_id=46bd4a6f-6a4d-4e81-ae7c-c9d7ac05b235"
```

**Confirmar Reserva**
```bash
curl -X POST http://localhost:8080/bookings/{booking_id}/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": "{payment_id}"
  }'
```

**Cancelar Reserva**
```bash
curl -X PATCH http://localhost:8080/bookings/{booking_id}/cancel \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Paciente solicita reagendar"
  }'
```

#### Payments

**Crear Pago**
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

**Procesar Pago**
```bash
curl -X POST http://localhost:8080/payments/{payment_id}/process \
  -H "Content-Type: application/json" \
  -d '{
    "payment_method_id": "method-demo"
  }'
```

</details>

## Estructura del Repositorio

```text
MedConnect/
├── api-gateway/                    # Gateway HTTP unificado (BFF)
├── auth-service/                   # Servicio gRPC de autenticación
├── availability-service/           # Servicio gRPC de disponibilidad
├── booking-service/                # Servicio gRPC de reservas
├── payment-service/                # Servicio gRPC de pagos
├── docker-compose.yml              # Orquestación de infraestructura
└── Insomnia_2026-05-07.yaml        # Colección de pruebas
```

## Próximos Pasos (To-Do)
- [ ] Construir e integrar la capa **Frontend** (React/Vite) para conectarse mediante HTTP REST al API Gateway.
- [ ] Implementar middleware JWT completo en API Gateway.

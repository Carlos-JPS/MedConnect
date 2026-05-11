# MedConnect

**MedConnect** es una plataforma distribuida de gestión de citas médicas diseñada para pacientes y clínicas. Permite a los usuarios buscar médicos, visualizar horarios disponibles, bloquear horas y procesar pagos. El sistema está construido sobre un ecosistema de microservicios independientes que se comunican internamente por gRPC y exponen una interfaz unificada al exterior a través de un API Gateway REST. Cada servicio posee su propia base de datos PostgreSQL aislada.

## Guía Operacional: Levantar el Sistema

El proyecto está completamente Dockerizado. No es necesario instalar Go localmente.

### Requisitos

- Docker
- Docker Compose V2

### 1. Variables de Entorno

Copia el archivo de entorno base. Los valores por defecto son funcionales para desarrollo local y no requieren modificación:

```bash
cp .env.example .env
```

Las variables principales que usa el sistema son:

```env
API_GATEWAY_HOST=0.0.0.0
API_GATEWAY_PORT=8080
API_GATEWAY_REQUEST_TIMEOUT=5s

BOOKING_SERVICE_TARGET=booking-service:50051
PAYMENT_SERVICE_TARGET=payment-service:50051
AVAILABILITY_SERVICE_TARGET=availability-service:50051
AUTH_SERVICE_TARGET=auth-service:50051

BOOKING_DB_DSN=postgres://booking:booking_password@booking_db:5432/booking_db?sslmode=disable
AVAILABILITY_DB_USER=postgres
AVAILABILITY_DB_PASSWORD=postgres
AVAILABILITY_DB_NAME=availability_db

PAYMENTS_DB_USER=postgres
PAYMENTS_DB_PASSWORD=postgres
PAYMENTS_DB_NAME=payments_db

AUTH_DB_USER=auth
AUTH_DB_PASSWORD=auth_password
AUTH_DB_NAME=auth_db
JWT_SECRET=medconnect-jwt-secret-change-me
```

### 2. Iniciar Servicios

```bash
docker compose up -d --build
```

Espera unos 15-20 segundos para que todas las bases de datos inicialicen antes de enviar peticiones.

```bash
# Verificar que todos los contenedores estén en ejecución
docker compose ps

# Ver logs en tiempo real
docker compose logs -f

# Detener y eliminar datos
docker compose down -v
```

### 3. Punto de Acceso

Una vez levantado, el API Gateway queda disponible en:

```
http://localhost:8080
```

Todos los servicios gRPC y bases de datos operan únicamente dentro de la red interna `medconnect_internal` y no son accesibles desde el host.

## Pruebas con Insomnia

El archivo `Insomnia_2026-05-07.yaml` en la raíz del repositorio contiene todas las peticiones listas para usar.

### Paso 1: Importar la colección

1. Abre Insomnia
2. Ve a **File > Import**
3. Selecciona el archivo `Insomnia_2026-05-07.yaml`
4. La colección se carga con carpetas para **Auth**, **Availability**, **Bookings** y **Payments**

### Paso 2: Configurar el entorno

En algunos casos, Insomnia no carga los valores de entorno automáticamente, entonces, debes crearlos manualmente (continúa con el paso siguiente si no es así):

1. En la colección importada, haz clic en el selector de entorno (arriba a la izquierda, junto al nombre de la colección)
2. Selecciona **Manage Environments**
3. Crea un nuevo entorno llamado `Local` y pega el siguiente JSON:

```json
{
  "base_url": "http://localhost:8080",
  "patient_id": "46bd4a6f-6a4d-4e81-ae7c-c9d7ac05b235",
  "doctor_id": "7e0d2ab1-164e-4a28-8b95-f24293dd0e91",
  "slot_id": "0f5c2b6a-1a87-4b7e-ae2c-37ef2f9f1c21",
  "booking_id": "",
  "payment_id": "",
  "payment_method_id": "method-demo"
}
```

4. Guarda el entorno y selecciónalo como activo
5. A medida que avances en el flujo, actualiza `booking_id` y `payment_id` con los valores que retorne cada petición

### Paso 3: Flujo de prueba completo (Happy Path)

Ejecuta las peticiones en este orden exacto en Insomnia. Cada paso depende del anterior.

---

**Paso 1: Registrar un usuario**

`Auth / Register User`

```json
{
  "email": "paciente@test.com",
  "password": "password123",
  "full_name": "Juan Perez",
  "role": "PATIENT"
}
```

---

**Paso 2: Iniciar sesión**

`Auth / Login`

```json
{
  "email": "paciente@test.com",
  "password": "password123"
}
```

---

**Paso 3: Consultar disponibilidad**

`Availability / Get Available Slots`

Los datos de prueba se insertan automáticamente al levantar el sistema. Deberías ver 3 slots con estado `available` para las especialidades Cardiología, Traumatología y Medicina interna.

---

**Paso 4: Crear una reserva**

`Bookings / Create Booking`

```json
{
  "doctor_id": "{{doctor_id}}",
  "slot_id": "{{slot_id}}",
  "notes": "Control de rutina"
}
```

> [!NOTE]
> El `patient_id` ya no es necesario en el JSON. El API Gateway lo obtiene automáticamente de tu token de autenticación.

Copia el `booking_id` de la respuesta y actualízalo en el entorno de Insomnia.

---

**Paso 5: Crear un pago**

`Payments / Create Payment`

```json
{
  "booking_id": "{{booking_id}}",
  "amount": 15000,
  "currency": "CLP"
}
```

> [!NOTE]
> El `user_id` se extrae automáticamente del token.

Copia el `payment_id` de la respuesta y actualízalo en el entorno de Insomnia.

---

**Paso 6: Procesar el pago**

`Payments / Process Payment`

```json
{
  "payment_method_id": "{{payment_method_id}}"
}
```

El pago debe quedar en estado `APPROVED` o `COMPLETED`.

---

**Paso 7: Confirmar la reserva**

`Bookings / Confirm Booking`

```json
{
  "payment_id": "{{payment_id}}"
}
```

La reserva pasa a estado `CONFIRMED` y el slot en `availability-db` queda en `booked`.

---

**Paso 8: Verificar estado final**

`Bookings / Get Booking`

Verifica que la reserva muestre estado `CONFIRMED` y sus eventos de auditoría.

---

### Flujo alternativo: Cancelar una reserva

Si en lugar de confirmar deseas probar la cancelación, después del Paso 5 ejecuta:

`Bookings / Cancel Booking`

```json
{
  "reason": "Paciente solicita reagendar"
}
```

El slot vuelve a estado `available` en `availability-db` y la reserva queda en `CANCELLED`.

## Pruebas con cURL (alternativa a Insomnia)

Todos los endpoints son HTTP/REST y se acceden exclusivamente a través del API Gateway en `http://localhost:8080`. Los servicios gRPC internos no están expuestos al host, por lo que no se requiere `grpcurl`.

Ejecuta los siguientes comandos en el mismo orden secuencial del Happy Path.

```bash
# Paso 1: Registrar usuario
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"paciente@test.com","password":"password123","full_name":"Juan Perez","role":"PATIENT"}'

# Paso 2: Iniciar sesión
curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"paciente@test.com","password":"password123"}'

# Paso 3: Consultar disponibilidad (los datos se insertan automáticamente al levantar el sistema)
curl -s "http://localhost:8080/availability/slots?specialty=Cardiología&start_date=2026-01-01T00:00:00Z&end_date=2027-12-31T00:00:00Z"

# Paso 4: Crear reserva (reemplaza {TOKEN} con el valor del Paso 2)
curl -s -X POST http://localhost:8080/bookings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {TOKEN}" \
  -d '{"doctor_id":"7e0d2ab1-164e-4a28-8b95-f24293dd0e91","slot_id":"0f5c2b6a-1a87-4b7e-ae2c-37ef2f9f1c21","notes":"Control de rutina"}'

# Paso 5: Crear pago
curl -s -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {TOKEN}" \
  -d '{"booking_id":"{BOOKING_ID}","amount":15000,"currency":"CLP"}'

# Paso 6: Procesar pago (reemplaza {PAYMENT_ID} con el valor del paso anterior)
curl -s -X POST http://localhost:8080/payments/{PAYMENT_ID}/process \
  -H "Content-Type: application/json" \
  -d '{"payment_method_id":"method-demo"}'

# Paso 7: Confirmar reserva
curl -s -X POST http://localhost:8080/bookings/{BOOKING_ID}/confirm \
  -H "Content-Type: application/json" \
  -d '{"payment_id":"{PAYMENT_ID}"}'

# Paso 8: Verificar estado final
curl -s http://localhost:8080/bookings/{BOOKING_ID}
```

**Flujo alternativo: Cancelar reserva** (en lugar del Paso 8)

```bash
curl -s -X PATCH http://localhost:8080/bookings/{BOOKING_ID}/cancel \
  -H "Content-Type: application/json" \
  -d '{"reason":"Paciente solicita reagendar"}'
```

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

- [ ] Construir e integrar la capa **Frontend** para conectarse mediante HTTP REST al API Gateway.

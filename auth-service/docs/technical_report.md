# Reporte Técnico: Auth Service - MedConnect

Este documento detalla la implementación, arquitectura y decisiones técnicas del **Auth Service**, cumpliendo con los criterios de evaluación de la Rúbrica Técnica.

## 1. Descripción del Servicio
El **Auth Service** es el microservicio encargado de la gestión de identidad y seguridad en el ecosistema MedConnect. Su propósito es proveer un mecanismo centralizado y seguro para el registro de usuarios (Pacientes, Doctores, Administradores), la autenticación de los mismos y la emisión/validación de credenciales digitales (Tokens JWT).

## 2. Casos de Uso y Flujos Técnicos

### Caso de Uso 1: Registro de nuevo Usuario
*   **Operación**: Un usuario envía sus datos (email, password, nombre, rol) para crear una cuenta.
*   **Resultado**: El sistema crea el usuario de forma persistente y devuelve sus datos básicos (excepto contraseña).
*   **Flujo Técnico**:
    1.  `API Gateway` recibe `POST /auth/register` (REST).
    2.  `API Gateway` llama a `RegisterUser` (gRPC) en `Auth Service`.
    3.  `Auth Service` genera un hash seguro de la contraseña usando **Bcrypt**.
    4.  `Auth Service` persiste el registro en la base de datos `auth_db` (PostgreSQL).
    5.  Retorno exitoso con el ID único (UUID) generado.

### Caso de Uso 2: Inicio de Sesión (Login)
*   **Operación**: El usuario ingresa su email y contraseña.
*   **Resultado**: El sistema valida las credenciales y entrega un **Access Token (JWT)** válido por 24 horas.
*   **Flujo Técnico**:
    1.  `API Gateway` recibe `POST /auth/login`.
    2.  `Auth Service` recupera el hash de la contraseña de `auth_db` por email.
    3.  Se verifica la contraseña usando Bcrypt.
    4.  Si es válida, `Auth Service` genera un JWT firmado con una clave secreta (HS256) que incluye el `user_id` y `role` en los claims.
    5.  Retorno del token al cliente.

### Caso de Uso 3: Validación de Identidad (Interno)
*   **Operación**: Un servicio (ej. Booking) recibe una petición y necesita verificar quién es el usuario.
*   **Resultado**: El token es validado y se retorna la identidad (ID y Rol) del usuario.
*   **Flujo Técnico**:
    1.  `Booking Service` recibe un token en el header.
    2.  Llama a `ValidateToken` (gRPC) en `Auth Service`.
    3.  `Auth Service` verifica la firma y la fecha de expiración del JWT.
    4.  `Auth Service` consulta `auth_db` para asegurar que el usuario sigue activo.
    5.  Retorno de validez y claims al servicio solicitante.

### Caso de Uso 4: Consulta de Perfil
*   **Operación**: El sistema requiere mostrar los datos básicos de un usuario por su ID.
*   **Resultado**: Datos de perfil (Nombre, Email, Rol).
*   **Flujo Técnico**:
    1.  Llamada gRPC `GetUserById`.
    2.  `Auth Service` realiza una consulta `SELECT` por UUID en `auth_db`.
    3.  Respuesta con los datos serializados en Protobuf.

## 3. Decisiones Técnicas y Trade-offs

| Tecnología | Decisión | Trade-off / Alternativa descartada |
| :--- | :--- | :--- |
| **Go (Golang)** | Lenguaje principal por su eficiencia en concurrencia y excelente soporte de gRPC. | Se descartó Node.js por el tipado fuerte y la facilidad de crear binarios estáticos pequeños para Docker. |
| **PostgreSQL** | Base de datos relacional independiente por servicio. | Se descartó compartir base de datos con `Booking` para evitar el acoplamiento (Coupling Negativo) y permitir escalado independiente. |
| **Bcrypt** | Algoritmo de hashing con "salt" automático para contraseñas. | Se descartó SHA-256 por ser vulnerable a ataques de fuerza bruta/rainbow tables al no tener costo de cómputo ajustable. |
| **JWT (JSON Web Token)** | Mecanismo de autenticación Stateless. | Se descartaron sesiones en servidor (Redis) para mantener el sistema escalable horizontalmente sin necesidad de compartir estado de sesión entre réplicas. |

## 4. Diseño de Protobuf
El contrato en `auth.proto` fue diseñado siguiendo el principio de **Responsabilidad Única**. Se utilizan tipos estándar de Google (`google.protobuf.Timestamp`) para las fechas, asegurando compatibilidad multiplataforma. Las contraseñas nunca son devueltas en los mensajes de respuesta para garantizar la seguridad.

## 5. Resiliencia y Configuración
*   **Manejo de Errores**: Todas las llamadas a la base de datos incluyen un `context.WithTimeout` de 5 segundos. Si la base de datos no responde, el servicio devuelve un código gRPC `Unavailable` en lugar de quedar bloqueado.
*   **Configuración**: No existen credenciales ni puertos hardcodeados. Todo se gestiona vía variables de entorno (`AUTH_DB_DSN`, `JWT_SECRET`, etc.) facilitando el despliegue en entornos de staging o producción.

# Rúbrica Técnica de Evaluación: Microservicios y Arquitectura Distribuida

**Puntaje Total:** 100 puntos  
**Distribución:** 60 pts Grupal | 40 pts Individual

---

## 1. Criterios Grupales (60 pts)
*Todos los integrantes reciben la misma nota en esta sección.*

### A. Arquitectura (30 pts)

| Criterio | Excelente (Máx) | Suficiente | Insuficiente |
| :--- | :---: | :---: | :---: |
| **Separación de responsabilidades** | **4 pts**: Al menos 3 servicios de negocio (+ API Gateway y Usuarios). Responsabilidad clara y sin lógica mezclada. | **2 pts**: Servicios separados pero con responsabilidades poco claras o superpuestas. | **0 pts**: Sin separación coherente. |
| **Implementación en Go** | **3 pts**: Al menos 2 servicios de negocio escritos en Go. | **2 pts**: Solo 1 servicio de negocio escrito en Go. | **0 pts**: Ningún servicio escrito en Go. |
| **Coupling negativo** | **4 pts**: Sin base de datos compartida, estructuras internas expuestas ni terceros innecesarios. | **2 pts**: Acoplamiento cuestionable que no compromete la independencia. | **0 pts**: Comparten BD o exponen estructuras internas. |
| **Justificación de límites** | **4 pts**: Límites basados en cohesión y responsabilidad deliberada, no en división de trabajo. | **2 pts**: Algunos servicios parecen separados por conveniencia o asignación. | **0 pts**: División arbitraria sin justificación. |
| **API Gateway** | **4 pts**: Punto de entrada único. Traduce HTTP/REST a gRPC. Servicios internos inaccesibles desde fuera de Docker. | **2 pts**: Existe Gateway pero hay servicios accesibles o traducción incompleta. | **0 pts**: No existe Gateway. Llamadas gRPC directas desde el cliente. |
| **gRPC y Protobuf** | **5 pts**: Protobuf bien definido, tipos correctos y comunicación funcional. | **3 pts**: Funciona pero con decisiones de diseño cuestionables. | **0 pts**: gRPC no funciona o Protobuf mal definido. |
| **Docker Compose y Red** | **3 pts**: Levanta con `docker compose up`. Comunicación por nombres de servicio, sin IPs hardcodeadas. Uso de `.env`. | **2 pts**: Levanta pero hay IPs hardcodeadas o faltan variables en `.env`. | **0 pts**: El sistema no levanta o no hay comunicación. |
| **Entidades de dominio** | **3 pts**: Al menos 2 entidades de dominio distintas con persistencia real en BD. | **2 pts**: Solo 1 entidad con persistencia real o están incompletas. | **0 pts**: Todo queda in-memory. |

### B. Documento Técnico (30 pts)

| Criterio | Excelente (Máx) | Suficiente | Insuficiente |
| :--- | :---: | :---: | :---: |
| **Descripción del sistema** | **2 pts**: Clara y precisa sobre qué hace el sistema y para quién. | **1 pts**: Vaga o incompleta. | **0 pts**: Sin descripción o no corresponde. |
| **Diagrama de arquitectura** | **2 pts**: Refleja el sistema real: servicios, BD, Gateway y flujos de comunicación. | **1 pts**: Presente pero incompleto o desactualizado. | **0 pts**: Ausente o incorrecto. |
| **Use cases y flujos** | **8 pts**: 4 casos con: (1) Operación de usuario/resultado y (2) Flujo técnico (quién llama a quién y persistencia). | **5 pts**: Descritos pero falta detalle en el orden técnico o incompletos. | **0 pts**: Solo nombres de servicios sin flujos. |
| **Decisiones y trade-offs** | **14 pts**: Justificación de cada tecnología y límite con trade-off explícito y alternativas descartadas. | **8 pts**: Justificadas pero sin mencionar trade-offs o alternativas. | **0 pts**: Sin argumentos técnicos. |
| **README operacional** | **4 pts**: Instrucciones para levantar, variables de ejemplo y cómo probar endpoints. | **2 pts**: Faltan variables o instrucciones de prueba incompletas. | **0 pts**: No permite levantar el sistema. |

---

## 2. Criterios Individuales: Implementación de tu Servicio (40 pts)
*Evaluación por cada servicio individual, no afecta al resto del grupo.*

| Criterio | Excelente (Máx) | Suficiente | Insuficiente |
| :--- | :---: | :---: | :---: |
| **Diseño del Protobuf** | **8 pts**: Tipos correctos, nombres de negocio claros, sin campos innecesarios. | **5 pts**: Nombres genéricos o tipos incorrectos. | **0 pts**: Mal diseñado o no representa los datos. |
| **Persistencia** | **8 pts**: Datos persisten correctamente en BD. | **5 pts**: Funciona pero con brechas menores. | **0 pts**: Datos in-memory o persistencia fallida. |
| **Métodos gRPC** | **8 pts**: Todos los métodos expuestos funcionan correctamente end-to-end. | **5 pts**: La mayoría funciona con fallas menores. | **2 pts**: No funciona o no responde. |
| **Resiliencia externa** | **8 pts**: Si un externo falla, el servicio no muere (manejo de error, retry o fallback). | **5 pts**: Sobrevive a la mayoría pero hay casos de pánico o cuelgue. | **0 pts**: El servicio muere si falla un externo. |
| **Configuración** | **2 pts**: Sin URLs/puertos hardcodeados. Todo vía variables de entorno. | **1 pts**: Mínimos valores hardcodeados presentes. | **0 pts**: Credenciales o URLs en el código. |
| **Estructura interna** | **6 pts**: Separación clara entre handler gRPC, lógica de negocio y acceso a datos. | **4 pts**: Separación parcial o intentos de organización. | **0 pts**: Todo en un solo archivo/función. |

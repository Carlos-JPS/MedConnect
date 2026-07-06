# Documento técnico vivo · Observabilidad en MedConnect Backend

**Bloque individual:** Observabilidad  
**Servicios involucrados:** `api-gateway`, `auth-service`, `booking-service`, `availability-service`, `payment-service`  
**Estado:** observabilidad implementada y validada en backend local  
**Última actualización:** 2026-07-06

---

## 1. Descripción del bloque Observabilidad

El bloque Observabilidad permite entender qué ocurre dentro del backend mientras el sistema está funcionando. En MedConnect se implementó para poder revisar métricas, logs y errores sin tener que entrar manualmente a cada servicio.

El sistema tiene un API Gateway REST como punto de entrada público y varios servicios internos que se comunican por gRPC. La observabilidad se agregó sobre ese flujo existente:

```text
Cliente
  -> API Gateway REST
  -> servicios backend por gRPC
  -> métricas expuestas en /metrics
  -> Prometheus recolecta métricas
  -> Alloy recolecta logs de Docker
  -> Loki guarda logs
  -> Grafana muestra métricas y logs
```

Los servicios involucrados son:

- `api-gateway`: recibe requests HTTP, enruta hacia los servicios internos y expone métricas HTTP.
- `auth-service`: maneja autenticación, login, registro y validación de tokens.
- `booking-service`: maneja reservas y coordina llamadas a disponibilidad y pagos.
- `availability-service`: maneja agendas y slots médicos.
- `payment-service`: maneja creación, consulta, procesamiento y reembolso de pagos.

El API Gateway expone métricas sobre las requests HTTP que recibe. Los servicios internos exponen métricas sobre las llamadas gRPC que atienden. Además, los logs de los contenedores se centralizan en Loki y se pueden revisar desde Grafana.

Para poder seguir una operación entre servicios se agregó el header:

```text
X-Request-ID
```

Si el cliente envía ese header, el API Gateway lo reutiliza. Si no lo envía, el gateway genera uno nuevo. Luego ese valor se propaga a los servicios gRPC mediante metadata. Así, una misma request puede buscarse en los logs aunque haya pasado por más de un servicio.

---

## 2. Implementación del bloque

### 2.1 Métricas HTTP en el API Gateway

El API Gateway expone métricas en:

```text
GET /metrics
```

Se agregó un middleware en:

```text
api-gateway/internal/http/metrics.go
```

Métricas principales:

| Métrica | Tipo | Labels | Uso |
|---|---|---|---|
| `medconnect_api_gateway_http_requests_total` | Counter | `method`, `route`, `status` | Contar requests HTTP |
| `medconnect_api_gateway_http_request_duration_seconds` | Histogram | `method`, `route`, `status` | Medir duración de requests HTTP |

Las rutas se normalizan antes de registrarse. Por ejemplo:

```text
/bookings/123              -> /bookings/{id}
/payments/456/process     -> /payments/{id}/process
/availability/doctors/789 -> /availability/doctors/{id}
```

Esto evita que Prometheus cree una serie distinta para cada identificador.

### 2.2 Métricas gRPC en servicios internos

Los servicios internos exponen métricas en un servidor HTTP separado, usando el puerto `9090`:

```text
GET /metrics
```

Servicios instrumentados:

- `auth-service`.
- `booking-service`.
- `availability-service`.
- `payment-service`.

Cada servicio usa un interceptor gRPC unary para registrar la llamada cuando termina.

Métricas principales:

| Métrica | Tipo | Labels | Uso |
|---|---|---|---|
| `medconnect_grpc_server_requests_total` | Counter | `service`, `method`, `code` | Contar llamadas gRPC |
| `medconnect_grpc_server_request_duration_seconds` | Histogram | `service`, `method`, `code` | Medir duración de llamadas gRPC |

El label `code` permite diferenciar respuestas exitosas y errores, por ejemplo `OK`, `NotFound`, `Unavailable` o `DeadlineExceeded`.

### 2.3 Logs centralizados

Los servicios escriben logs a stdout/stderr. Alloy lee esos logs desde Docker y los envía a Loki.

Configuración principal:

```text
observability/alloy/config.alloy
observability/loki/loki-config.yml
```

Alloy agrega labels útiles a los logs, como:

- `container`.
- `service_name`.
- `compose_project`.

Loki guarda los logs localmente con una retención de 7 días:

```text
retention_period: 168h
```

### 2.4 Dashboard de Grafana

Grafana se levanta con datasources y dashboard provisionados desde archivos.

Datasources:

```text
Prometheus -> métricas
Loki       -> logs
```

Dashboard principal:

```text
observability/grafana/provisioning/dashboards/json/medconnect-observability.json
```

Paneles incluidos:

| Panel | Fuente |
|---|---|
| API Gateway HTTP requests | Prometheus |
| API Gateway latency p95 | Prometheus |
| gRPC requests | Prometheus |
| gRPC latency p95 | Prometheus |
| Targets up | Prometheus |
| Logs recientes de MedConnect | Loki |
| Logs de API Gateway | Loki |

---

## 3. Decisión técnica principal

La decisión principal fue usar:

```text
Prometheus + Grafana + Loki + Alloy + X-Request-ID
```

La alternativa evaluada fue agregar tracing distribuido completo con Jaeger o Tempo.

Se descartó esa alternativa porque habría requerido introducir spans, exporters y propagación de contexto de tracing en todos los servicios. Para esta entrega, eso aumentaba bastante el alcance del cambio. El objetivo principal era poder observar el sistema sin modificar profundamente la lógica existente.

Con la decisión aplicada se cubren las preguntas más importantes para depurar el backend:

- cuántas requests llegan al API Gateway;
- cuánto demoran las requests HTTP;
- qué rutas responden con error;
- cuántas llamadas gRPC recibe cada servicio;
- qué métodos gRPC fallan;
- qué servicios están arriba o abajo;
- qué logs corresponden a una misma request.

El `X-Request-ID` reemplaza parcialmente la necesidad de tracing para esta entrega. No entrega un árbol de spans, pero permite correlacionar logs entre el gateway y los servicios internos.

---

## 4. Comportamiento ante fallos

### 4.1 Servicio backend caído

Si un servicio backend se cae, Prometheus deja de recolectar métricas desde ese target y el panel `Targets up` lo muestra como caído.

El API Gateway sigue funcionando, pero las rutas que dependen del servicio caído responden error. El log del gateway conserva el `request_id`, por lo que se puede buscar la request fallida en Loki.

Ejemplo:

```text
auth-service caído
  -> Prometheus marca auth-service como down
  -> requests de login/validación fallan
  -> api-gateway registra el error con request_id
```

### 4.2 Prometheus caído

Si Prometheus se cae, los servicios no dejan de funcionar. El impacto es que las métricas dejan de recolectarse mientras Prometheus esté detenido.

La degradación es parcial:

- el backend sigue atendiendo requests;
- Loki puede seguir recibiendo logs;
- Grafana no puede mostrar paneles basados en Prometheus hasta que vuelva.

### 4.3 Loki caído

Si Loki se cae, las métricas siguen funcionando porque Prometheus no depende de Loki.

El impacto es sobre los logs:

- Grafana no puede consultar logs recientes;
- Alloy no puede enviarlos correctamente mientras Loki esté caído;
- al recuperar Loki, la observabilidad de métricas no requiere cambios.

### 4.4 Alloy caído

Si Alloy se cae, los servicios siguen funcionando y Prometheus sigue recolectando métricas.

El impacto es que los logs dejan de llegar a Loki mientras Alloy esté detenido. Al reiniciar Alloy, vuelve a leer logs de los contenedores disponibles y los envía a Loki.

### 4.5 Grafana caído

Si Grafana se cae, no se puede usar el dashboard, pero Prometheus y Loki siguen funcionando.

Al reiniciar Grafana, los datasources y dashboards se cargan nuevamente desde los archivos provisionados.

### 4.6 Servidor de métricas de un servicio falla

En los servicios gRPC, el servidor de métricas se levanta separado del servidor principal. Si el servidor de métricas falla, el servicio puede seguir atendiendo llamadas gRPC.

Esto evita que un problema en observabilidad detenga la funcionalidad principal del servicio.

---

## 5. Casos de borde considerados

| Caso | Comportamiento |
|---|---|
| Request sin `X-Request-ID` | El gateway genera uno nuevo |
| Request con `X-Request-ID` | El gateway reutiliza el valor recibido |
| Handler no escribe status explícito | Se registra status `200` |
| Ruta con ID dinámico | Se normaliza antes de registrar la métrica |
| Ruta no conocida | Se registra como `not_found` |
| Llamada gRPC sin metadata `x-request-id` | El log usa `request_id=unknown` |
| Servicio no disponible | Prometheus lo refleja como target caído |

---

## 6. Trade-offs y limitaciones

### 6.1 Sin tracing distribuido completo

No se implementaron spans distribuidos con Jaeger o Tempo. La correlación se hace mediante `X-Request-ID` en logs.

Trade-off aceptado: se pierde detalle fino del recorrido interno de una request, pero se mantiene una implementación más simple y suficiente para seguir errores entre servicios.

### 6.2 Logs en texto

Los logs se escriben en formato texto con pares `key=value`, no como JSON estructurado.

Trade-off aceptado: se evita cambiar el sistema de logging de todos los servicios. Loki permite buscar por texto y por labels de contenedor, lo que es suficiente para esta entrega.

### 6.3 Sin alertas automáticas

No se configuraron alertas en Prometheus ni Grafana.

Trade-off aceptado: el bloque se enfocó en recolectar, visualizar y correlacionar información. Las alertas pueden agregarse como mejora posterior.

### 6.4 Sin métricas de consumo de recursos

No se agregaron métricas específicas de CPU, memoria o disco de contenedores.

Trade-off aceptado: se priorizaron métricas de aplicación, como requests, latencia, códigos de respuesta y estado de servicios.

### 6.5 Métricas transversales, no específicas de cada dominio

Las métricas agregadas son transversales al sistema. Por ejemplo, `availability-service` expone métricas gRPC generales, pero no métricas internas por shard.

Trade-off aceptado: se mantuvo la observabilidad desacoplada de la lógica interna de cada servicio. Esto reduce el riesgo de modificar comportamiento funcional existente.

---

## 7. Estado de validación

El stack se levanta con:

```bash
docker compose up -d --build api-gateway
```

Prometheus scrapea los siguientes targets:

```text
prometheus:9090
api-gateway:8080
auth-service:9090
booking-service:9090
availability-service:9090
payment-service:9090
```

El dashboard de Grafana queda disponible en:

```text
http://localhost:3000
```

Se validó que:

- el API Gateway expone métricas HTTP;
- los servicios internos exponen métricas gRPC;
- Prometheus recolecta métricas desde los servicios;
- Grafana carga datasources y dashboard desde archivos;
- Alloy envía logs de contenedores a Loki;
- los logs pueden buscarse usando `request_id`.

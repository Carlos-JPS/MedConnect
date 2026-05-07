# MedConnect

**Sistema de Gestión Clínica Hospitalaria Basado en Microservicios con Go y FastAPI**

MedConnect es una plataforma médica integral diseñada para optimizar la gestión clínica en entornos hospitalarios. Utilizando una arquitectura de microservicios moderna, el sistema permite la administración centralizada de pacientes, citas, historias clínicas, medicamentos, imágenes diagnósticas y pagos, todo ello soportado por una infraestructura robusta y escalable.

## 🚀 Características Principales

### 🏥 Gestión Clínica Integral
- **Pacientes**: Registro completo, historial médico y seguimiento personalizado.
- **Citas Médicas**: Programación, recordatorios y gestión de consultas.
- **Historias Clínicas**: Evolución médica digitalizada, diagnósticos y tratamientos.

### 📋 Administración Hospitalaria
- **Medicamentos**: Inventario y dispensación controlada.
- **Imágenes Diagnósticas**: Carga, visualización y gestión de estudios radiológicos.
- **Recursos Humanos**: Control de personal médico y administrativo.

### 💳 Sistema de Pagos
- **Pagos Integrados**: Módulo de pago seguro para servicios médicos.
- **Facturación**: Generación de facturas electrónicas.

## 🛠️ Arquitectura y Tecnologías

### Microservicios
El sistema está compuesto por múltiples servicios especializados, cada uno responsable de una funcionalidad específica:
- **Gateway Service**: Orquestación centralizada y API unificada (Go).
- **Patient Service**: Gestión de pacientes y registros médicos (FastAPI).
- **Appointment Service**: Administración de citas (FastAPI).
- **Billing Service**: Facturación y cobros (FastAPI).
- **Payment Service**: Procesamiento de pagos (FastAPI).

### Stack Tecnológico
- **Backend**: Go (Golang) y Python (FastAPI).
- **Orquestación**: Docker y Docker Compose.
- **Base de Datos**: PostgreSQL con soporte para PostgreSQL Native UUIDs.
- **RPC**: gRPC para comunicación inter-servicio.
- **API REST**: Endpoints RESTful para clientes web y móviles.

## 📦 Instalación y Ejecución

### Requisitos Previos
- Docker
- Docker Compose V2

### Ejecución del Sistema
Para iniciar todos los servicios con Docker Compose:
```bash
docker compose up --build -d
```

Para detener todos los servicios:
```bash
docker compose down
```

## 📂 Estructura del Proyecto

```
MedConnect/
├── docker-compose.yml              # Configuración de Docker Compose
├── .gitignore                      # Archivos ignorados por Git
├── .env                            # Variables de entorno (crear manualmente)
├── Insomnia_2026-05-07.yaml        # Colección de tests en Insomnia
├── README.md                       # Documentación del proyecto
├── README_ENG.md                   # English documentation
├── admin-service/                  # Servicio de administración (Go)
│   ├── ...
├── appointment-service/            # Servicio de citas (FastAPI)
│   ├── ...
├── billing-service/                # Servicio de facturación (FastAPI)
│   ├── ...
├── gateway-service/                # Gateway service (Go)
│   ├── ...
├── payment-service/                # Servicio de pagos (FastAPI)
│   ├── ...
└── patient-service/                # Servicio de pacientes (FastAPI)
    ├── ...
```

## 🤝 Contribuciones

Este proyecto es el resultado del trabajo colaborativo entre estudiantes de ingeniería informática.

## 📝 Licencia

Propiedad intelectual del proyecto desarrollado para la asignatura de Sistemas Distribuidos, Universidad de La Frontera (UFRO).
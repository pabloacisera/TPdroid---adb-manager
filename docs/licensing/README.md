# Sistema de Licencias TPDroid

DRM de 4 fases para el binario compilado TPDroid, usando Cloudflare Workers y Supabase.

## Arquitectura General

```
┌──────────────┐     POST /activar      ┌──────────────┐      REST      ┌──────────┐
│  activator   │ ──────────────────────→ │   Worker     │ ────────────→ │ Supabase │
│  (Win/Linux) │ ←────────────────────── │  cloudflare/ │ ←──────────── │          │
└──────────────┘     .lic (firmado)      └──────┬───────┘               └──────────┘
                                                │
                    POST /revalidar             │
┌──────────────┐ ──────────────────────────────→│
│   backend    │ ←──────────────────────────────│
│  (Go/Gin)    │     ok / error                 │
└──────┬───────┘                                │
       │                                        │
       │ 403 si no hay licencia                 │
       ▼                                        │
┌──────────────┐                                │
│ license-     │                                │
│ error.html   │                                │
└──────────────┘

┌──────────────────┐       ┌──────────────┐
│  Panel Admin     │ ────→ │   Supabase   │
│  (FastAPI)       │       │              │
└──────────────────┘       └──────────────┘
```

## Flujo de Activación

1. El desarrollador genera un código en el Panel Admin → se guarda en Supabase
2. El cliente ejecuta `TPDroid-Setup.exe` (o `install.sh` en Linux)
3. El instalador ejecuta `activator` que abre un formulario en el navegador
4. El usuario ingresa el código de licencia
5. `activator` obtiene el fingerprint del hardware (SHA-256)
6. Envía `{codigo, hw_id}` al Worker (`POST /activar`)
7. El Worker valida en Supabase: código existe y no está usado
8. Si ok: marca como usado, genera archivo `.lic` firmado con HMAC
9. `activator` guarda `.lic` en el sistema y retorna exit code 0
10. Si error: muestra mensaje, retorna exit code ≠ 0
11. El instalador verifica el exit code: 0 → continúa, ≠ 0 → aborta

## Flujo de Revalidación

1. El backend inicia y lee el archivo `.lic`
2. Envía el `.lic` + `current_hw_id` al Worker (`POST /revalidar`)
3. El Worker valida: HMAC, hw_id actual vs registrado, estado en Supabase
4. Si ok: el backend funciona normalmente
5. Si falla: todas las rutas `/api/*` devuelven 403, se sirve `license-error.html`

## Capas de Seguridad

1. **Primera capa** (instalador): `activator` evita la instalación si el código es inválido
2. **Segunda capa** (backend): revalidación contra el Worker cada vez que arranca la app
3. **Firma HMAC**: el `.lic` está firmado, no se puede falsificar sin el secreto del Worker
4. **Binding a hardware**: el `.lic` está vinculado al `hw_id`, no funciona en otro equipo

## Estructura de Carpetas

```
cloudflare/          → Worker + schema SQL
  schema.sql         → Tabla licencias en Supabase
  worker.js          → Endpoints /activar y /revalidar
  test-worker.js     → Tests unitarios del Worker

activator/           → Binario de activación (Win + Linux)
  main.go            → Servidor HTTP embebido + flujo de activación
  fingerprint.go     → Fingerprint cross-platform
  activate.html      → Formulario de activación (go:embed)

api_admin/           → Panel de administración
  main.py            → FastAPI: generar, listar, ver códigos
  requirements.txt   → Dependencias

backend/licensing/   → Paquete Go de validación (extraíble)
  client.go          → HTTP client para Worker
  middleware.go       → Gin middleware de bloqueo
```

## Variables de Entorno

### Cloudflare Worker
| Variable | Descripción |
|----------|-------------|
| `SUPABASE_URL` | URL del proyecto Supabase |
| `SUPABASE_SERVICE_KEY` | Service role key |
| `LICENSE_SECRET` | Secreto HMAC para firmar .lic |

### Backend (Go)
| Variable | Descripción |
|----------|-------------|
| `LICENSE_WORKER_URL` | URL del Worker (ej: https://licencias.mi-worker.workers.dev) |
| `LICENSE_PATH` | (opcional) Ruta al archivo .lic. Default: %APPDATA%/TPDroid/tpdroid.lic |

### Panel Admin (FastAPI)
| Variable | Descripción |
|----------|-------------|
| `SUPABASE_URL` | URL del proyecto Supabase |
| `SUPABASE_SERVICE_KEY` | Service role key |

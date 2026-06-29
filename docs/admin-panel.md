# Admin Panel de Licencias (FastAPI)

Panel web para administrar las licencias de TPDroid. Conecta directamente a Supabase via REST.

## Stack

- **Framework**: FastAPI
- **Auth**: API Key via header `X-API-Key`
- **Base de datos**: Supabase (PostgreSQL), misma tabla `licencias` que usa el Worker
- **Frontend**: HTML estático servido desde `api_admin/static/`

## Endpoints

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/codes` | Listar todos los códigos ordenados por fecha descendente |
| GET | `/api/codes/{codigo}` | Ver detalle de un código específico |
| POST | `/api/codes/generate` | Generar 1-100 códigos nuevos |
| DELETE | `/api/codes/{codigo}` | Revocar (si está en uso) o eliminar (si no se usó) |
| GET | `/api/stats` | Estadísticas: totales, activas, revocadas, disponibles |
| GET | `/` | Servir `static/index.html` |

## Configuración

Variables de entorno requeridas:

| Variable | Descripción |
|----------|-------------|
| `SUPABASE_URL` | URL del proyecto Supabase |
| `SUPABASE_SERVICE_KEY` | Service role key |
| `ADMIN_API_KEY` | Clave para autenticar requests (opcional — si no se setea, no hay auth) |

## Uso

```bash
cd api_admin
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8000
```

## Formato de códigos

`TPD-XXXX-XXXX-XXXX` donde cada X es alfanumérico mayúsculo (A-Z, 0-9).

## Estructura

```
api_admin/
  main.py              → API endpoints
  static/index.html    → Panel UI (HTML+JS)
  requirements.txt     → fastapi, uvicorn, httpx, python-dotenv
  .env.example         → Template de variables de entorno
  test_admin.py        → Tests
```

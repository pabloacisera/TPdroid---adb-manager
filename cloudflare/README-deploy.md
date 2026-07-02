# Deploy del Sistema de Licencias TPDroid

Guía paso a paso para deployar el sistema de licencias en Cloudflare Workers + Supabase.

---

## 1. Prerequisitos

- Node.js 18+
- `npm install -g wrangler`
- Cuenta en [Cloudflare](https://dash.cloudflare.com) (plan gratuito alcanza)
- Cuenta en [Supabase](https://supabase.com) (plan gratuito alcanza)
- `wrangler login` (abre el browser para autenticar tu cuenta de Cloudflare)

---

## 2. Paso 1 — Configurar Supabase

1. Crear un proyecto en [supabase.com](https://supabase.com)
2. Ir a **SQL Editor** y ejecutar el contenido de [`schema.sql`](./schema.sql)
3. Si ya tenés una base existente sin la columna `revocado`, ejecutar solo:

```sql
ALTER TABLE licencias ADD COLUMN IF NOT EXISTS revocado BOOLEAN NOT NULL DEFAULT false;
```

4. Obtener las credenciales necesarias:
   - Ir a **Project Settings → API**
   - Copiar **Project URL** (lo usás como `SUPABASE_URL`)
   - Copiar la **`service_role` key** (NO la `anon` key, necesitás la `service_role`)

---

## 3. Paso 2 — Configurar secrets en Cloudflare

Las secrets NUNCA van en el código. Se configuran con `wrangler secret put`.

```bash
cd cloudflare/

wrangler secret put SUPABASE_URL
# Pegar la Project URL de Supabase cuando lo pida (ej: https://xyz.supabase.co)

wrangler secret put SUPABASE_SERVICE_KEY
# Pegar la service_role key de Supabase

wrangler secret put LICENSE_SECRET
# Generar con: openssl rand -hex 32
# Pegar el resultado cuando lo pida

wrangler secret put ADMIN_SECRET
# Elegir una clave fuerte para proteger el endpoint /revocar
```

---

## 4. Paso 3 — Deploy

```bash
cd cloudflare/
wrangler deploy
```

La URL resultante tiene el formato `https://tpdroid-licencias.<tu-account>.workers.dev`.
**Anotala**, la vas a necesitar en los pasos siguientes.

---

## 5. Paso 4 — Verificar que funciona

Smoke tests con `curl`:

```bash
WORKER_URL="https://tpdroid-licencias.<tu-account>.workers.dev"
ADMIN_SECRET="<tu-admin-secret>"

# Test 1: activar con código inexistente → debe devolver 404
curl -s -X POST "$WORKER_URL/activar" \
  -H "Content-Type: application/json" \
  -d '{"codigo":"TPD-TEST-FAKE-0000","hw_id":"test-hw-id"}' | jq .

# Test 2: revocar sin header → debe devolver 401
curl -s -X POST "$WORKER_URL/revocar" \
  -H "Content-Type: application/json" \
  -d '{"codigo":"cualquiera"}' | jq .

# Test 3: revocar con código inexistente → debe devolver 404
curl -s -X POST "$WORKER_URL/revocar" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Secret: $ADMIN_SECRET" \
  -d '{"codigo":"TPD-TEST-FAKE-0000"}' | jq .
```

---

## 6. Paso 5 — Configurar variables de entorno locales

| Componente | Variable | Valor |
|---|---|---|
| `backend/` (tpdroid.exe) | `LICENSE_WORKER_URL` | URL del Worker |
| `api_admin/` | `SUPABASE_URL` | URL del proyecto |
| `api_admin/` | `SUPABASE_SERVICE_KEY` | service_role key |
| `api_admin/` | `ADMIN_API_KEY` | clave a elección |
| `activator/` (build) | `LICENSE_WORKER_URL` | URL del Worker (via ldflags) |

Además, el Worker Cloudflare necesita estas variables de entorno para los endpoints `GET /version` y `GET /definitions`:

```bash
# Configurar con wrangler secret put
wrangler secret put LATEST_VERSION    # ej: "0.2.0" — obligatorio post-release
wrangler secret put CHANGELOG         # Notas genéricas (fallback si no hay bilingües) — opcional
wrangler secret put NOTES_ES          # Notas en español — opcional
wrangler secret put NOTES_EN          # Notas en inglés — opcional
```

La URL de descarga (`download_url`) está hardcodeada en el Worker y apunta a `github.com/.../releases/latest`. No necesita configurarse.

Ejemplo de `.env` para el panel admin:

```bash
export SUPABASE_URL="https://xyz.supabase.co"
export SUPABASE_SERVICE_KEY="eyJ..."
export ADMIN_API_KEY="mi-clave-secreta"
```

Para iniciar el panel:

```bash
cd api_admin/
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
source .env   # o export manual de las variables
uvicorn main:app --host 127.0.0.1 --port 8000
# Abrir http://localhost:8000
```

---

## 7. Paso 6 — Compilar y distribuir

### Automático (CI/CD)

Las releases se generan al pushear un tag semántico (`v*`):

```bash
git tag v1.0.0
git push origin v1.0.0
```

El workflow de GitHub Actions:
- Compila backend + activator, empaqueta `dist/tpdroid-linux.tar.gz`
- Genera `dist/TPDroid-Setup.exe`
- Crea un **draft release** en GitHub con ambos archivos

La `LICENSE_WORKER_URL` se toma de `api_admin/.env`.

También podés dispararlo manualmente desde **GitHub → Actions → Build and Release → Run workflow**.

### Manual (local)

```bash
# Instalador Windows (requiere sudo apt install nsis en Linux)
export LICENSE_WORKER_URL="https://tpdroid-licencias.<account>.workers.dev"
make dist-windows
# Resultado: dist/TPDroid-Setup.exe

# Tarball Linux
make build-linux && make build-activator-linux
# Resultado: dist/tpdroid-linux + dist/activator-linux
```

---

## 8. Tests del sistema

```bash
# Worker (lógica HMAC)
node cloudflare/test-worker.js

# Activator (fingerprint)
cd activator && go test ./... -v

# Licensing (middleware + client)
cd backend && go test ./licensing/... -v

# Admin panel (endpoints + formato)
cd api_admin && python -m pytest test_admin.py -v
```

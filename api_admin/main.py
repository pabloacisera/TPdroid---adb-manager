"""Panel de Administración de Licencias TPDroid

Uso:
  export SUPABASE_URL=https://<project>.supabase.co
  export SUPABASE_SERVICE_KEY=<service-role-key>
  export ADMIN_API_KEY=<tu-clave-secreta>

  uvicorn main:app --host 0.0.0.0 --port 8000
"""

import os
import secrets
import string
from dotenv import load_dotenv
load_dotenv()

import httpx
from urllib.parse import quote as url_quote
from fastapi import Depends, FastAPI, Header, HTTPException, Query
from pydantic import BaseModel

app = FastAPI(title="TPDroid - Admin Licencias")

SUPABASE_URL = os.environ["SUPABASE_URL"]
SUPABASE_KEY = os.environ["SUPABASE_SERVICE_KEY"]
ADMIN_KEY = os.environ.get("ADMIN_API_KEY", "")
SUPABASE_HEADERS = {
    "Content-Type": "application/json",
    "apiKey": SUPABASE_KEY,
    "Authorization": f"Bearer {SUPABASE_KEY}",
    "Prefer": "return=representation",
}


# ─── Auth ────────────────────────────────────────────────

def verify_admin(x_api_key: str = Header(None)):
    if ADMIN_KEY and x_api_key != ADMIN_KEY:
        raise HTTPException(403, "Acceso denegado")


# ─── Helpers ─────────────────────────────────────────────

def generate_code() -> str:
    def block():
        return "".join(secrets.choice(string.ascii_uppercase + string.digits) for _ in range(4))
    return f"TPD-{block()}-{block()}-{block()}"


# ─── Models ──────────────────────────────────────────────

class CodeResponse(BaseModel):
    codigo: str
    usado: bool
    hw_id: str | None
    email: str | None
    fecha_activacion: str | None
    revocado: bool
    creado_en: str


class PaginatedCodesResponse(BaseModel):
    data: list[CodeResponse]
    total: int
    page: int
    limit: int
    total_pages: int


class GenerateRequest(BaseModel):
    cantidad: int = 1


class GenerateResponse(BaseModel):
    codigos: list[str]


class StatsResponse(BaseModel):
    total: int
    activas: int
    revocadas: int
    disponibles: int


# ─── Endpoints ───────────────────────────────────────────

@app.get("/api/codes")
def list_codes(
    email: str | None = None,
    status: str | None = Query(None, description="Filtrar por estado: disponible, activa, revocada"),
    page: int = Query(1, ge=1),
    limit: int = Query(50, ge=1, le=500),
    auth: None = Depends(verify_admin),
):
    base_url = f"{SUPABASE_URL}/rest/v1/licencias"

    filters = []
    if email:
        filters.append(f"email=ilike.*{url_quote(email)}*")

    # Map status to Supabase filters (applied at DB level, not post-filter)
    if status == "disponible":
        filters.append("usado=eq.false")
        filters.append("revocado=eq.false")
    elif status == "activa":
        filters.append("usado=eq.true")
        filters.append("revocado=eq.false")
    elif status == "revocada":
        filters.append("revocado=eq.true")

    filter_str = "&" + "&".join(filters) if filters else ""

    # Count total matching records (with same filters)
    count_url = f"{base_url}?select=codigo{filter_str}"
    count_resp = httpx.get(count_url, headers=SUPABASE_HEADERS)
    count_resp.raise_for_status()
    total = len(count_resp.json())

    # Fetch paginated data (with same filters applied at DB level)
    offset = (page - 1) * limit
    data_url = f"{base_url}?order=creado_en.desc&limit={limit}&offset={offset}{filter_str}"
    data_resp = httpx.get(data_url, headers=SUPABASE_HEADERS)
    data_resp.raise_for_status()
    data = data_resp.json()

    total_pages = max(1, (total + limit - 1) // limit)
    return {"data": data, "total": total, "page": page, "limit": limit, "total_pages": total_pages}


@app.get("/api/codes/{codigo}", response_model=CodeResponse)
def get_code(codigo: str, auth: None = Depends(verify_admin)):
    resp = httpx.get(
        f"{SUPABASE_URL}/rest/v1/licencias?codigo=eq.{codigo}&limit=1",
        headers=SUPABASE_HEADERS,
    )
    resp.raise_for_status()
    data = resp.json()
    if not data:
        raise HTTPException(404, "Código no encontrado")
    return data[0]


@app.post("/api/codes/generate", response_model=GenerateResponse)
def generate_codes(body: GenerateRequest, auth: None = Depends(verify_admin)):
    if body.cantidad < 1 or body.cantidad > 100:
        raise HTTPException(400, "La cantidad debe ser entre 1 y 100")

    codigos = []
    rows = []

    for _ in range(body.cantidad):
        code = generate_code()
        codigos.append(code)
        rows.append({"codigo": code})

    resp = httpx.post(
        f"{SUPABASE_URL}/rest/v1/licencias",
        headers=SUPABASE_HEADERS,
        json=rows,
    )
    resp.raise_for_status()

    return {"codigos": codigos}


@app.delete("/api/codes/{codigo}")
def delete_code(codigo: str, auth: None = Depends(verify_admin)):
    resp = httpx.get(
        f"{SUPABASE_URL}/rest/v1/licencias?codigo=eq.{codigo}&limit=1",
        headers=SUPABASE_HEADERS,
    )
    resp.raise_for_status()
    data = resp.json()
    if not data:
        raise HTTPException(404, "Código no encontrado")
    lic = data[0]
    if lic.get("revocado"):
        raise HTTPException(409, "La licencia ya estaba revocada")

    if lic.get("usado"):
        # En uso → revocar (marcar como revocada)
        patch_resp = httpx.patch(
            f"{SUPABASE_URL}/rest/v1/licencias?codigo=eq.{codigo}",
            headers=SUPABASE_HEADERS,
            json={"revocado": True},
        )
        patch_resp.raise_for_status()
        return {"ok": True, "accion": "revocar", "message": f"Licencia {codigo} revocada"}
    else:
        # No usado → eliminar registro
        del_resp = httpx.delete(
            f"{SUPABASE_URL}/rest/v1/licencias?codigo=eq.{codigo}",
            headers=SUPABASE_HEADERS,
        )
        del_resp.raise_for_status()
        return {"ok": True, "accion": "eliminar", "message": f"Código {codigo} eliminado"}


@app.get("/api/stats")
def stats(auth: None = Depends(verify_admin)):
    resp = httpx.get(
        f"{SUPABASE_URL}/rest/v1/licencias?select=codigo,usado,revocado",
        headers=SUPABASE_HEADERS,
    )
    resp.raise_for_status()
    data = resp.json()
    total = len(data)
    activas = sum(1 for d in data if d.get("usado") and not d.get("revocado"))
    revocadas = sum(1 for d in data if d.get("revocado"))
    disponibles = total - activas - revocadas
    return {
        "total": total,
        "activas": activas,
        "revocadas": revocadas,
        "disponibles": disponibles,
    }


# ─── Static UI ───────────────────────────────────────────

from fastapi.staticfiles import StaticFiles
from fastapi.responses import FileResponse
import os

STATIC_DIR = os.path.join(os.path.dirname(__file__), "static")
app.mount("/static", StaticFiles(directory=STATIC_DIR), name="static")


@app.get("/")
def index():
    return FileResponse(os.path.join(STATIC_DIR, "index.html"))

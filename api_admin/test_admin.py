"""Tests para el Panel Admin de Licencias.

Ejecutar:
  pip install -r requirements.txt
  python -m pytest test_admin.py -v
"""

import os

# Fijar env vars ANTES de que main.py se importe (load_dotenv usa override=False por defecto)
os.environ["SUPABASE_URL"] = "http://fake-supabase.test"
os.environ["SUPABASE_SERVICE_KEY"] = "fake-key"
os.environ["ADMIN_API_KEY"] = "test-api-key"

import re
import sys
from unittest.mock import patch

# ─── Tests de lógica interna ────────────────────────────

def test_generate_code_format():
    """Valida que el código generado tenga el formato TPD-XXXX-XXXX-XXXX"""
    sys.path.insert(0, ".")
    from main import generate_code

    code = generate_code()
    assert re.match(r"^TPD-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$", code), \
        f"Formato inválido: {code}"


def test_generate_code_unique():
    """Códigos generados deben ser distintos"""
    sys.path.insert(0, ".")
    from main import generate_code

    codes = {generate_code() for _ in range(100)}
    assert len(codes) == 100, "Códigos duplicados detectados"


def test_generate_code_no_lowercase():
    """Códigos deben ser solo mayúsculas y dígitos"""
    sys.path.insert(0, ".")
    from main import generate_code

    for _ in range(50):
        code = generate_code()
        # Solo TPD- + alfanumérico mayúscula
        for part in code.split("-")[1:]:
            assert re.match(r"^[A-Z0-9]{4}$", part), f"Caracter inválido en {code}"


from unittest.mock import MagicMock, patch
from fastapi.testclient import TestClient


def get_test_client():
    """Crea un TestClient con env vars de prueba."""
    import os
    os.environ.setdefault("SUPABASE_URL", "http://fake-supabase.test")
    os.environ.setdefault("SUPABASE_SERVICE_KEY", "fake-key")
    os.environ.setdefault("ADMIN_API_KEY", "test-api-key")
    from main import app
    return TestClient(app)


def test_revoke_requires_auth():
    """DELETE /api/codes/:codigo debe requerir X-API-Key."""
    client = get_test_client()
    resp = client.delete("/api/codes/TPD-TEST-0001-AAAA")
    assert resp.status_code == 403, f"Esperado 403, recibido {resp.status_code}"


def test_delete_not_found():
    """DELETE /api/codes/:codigo debe retornar 404 si el código no existe."""
    client = get_test_client()
    with patch("httpx.get") as mock_get:
        mock_resp = MagicMock()
        mock_resp.json.return_value = []
        mock_resp.raise_for_status = MagicMock()
        mock_get.return_value = mock_resp

        resp = client.delete(
            "/api/codes/TPD-FAKE-0000-ZZZZ",
            headers={"X-API-Key": "test-api-key"},
        )
        assert resp.status_code == 404, f"Esperado 404, recibido {resp.status_code}"


def test_delete_already_revoked():
    """DELETE /api/codes/:codigo debe retornar 409 si ya está revocada."""
    client = get_test_client()
    with patch("httpx.get") as mock_get:
        mock_resp = MagicMock()
        mock_resp.json.return_value = [{"codigo": "TPD-REVO-0001-KKKK", "usado": True, "revocado": True}]
        mock_resp.raise_for_status = MagicMock()
        mock_get.return_value = mock_resp

        resp = client.delete(
            "/api/codes/TPD-REVO-0001-KKKK",
            headers={"X-API-Key": "test-api-key"},
        )
        assert resp.status_code == 409, f"Esperado 409, recibido {resp.status_code}"


def test_delete_unused_code():
    """DELETE sobre código no usado debe eliminar el registro (DELETE a Supabase)."""
    client = get_test_client()
    with patch("httpx.get") as mock_get, patch("httpx.delete") as mock_del:
        mock_get_resp = MagicMock()
        mock_get_resp.json.return_value = [{"codigo": "TPD-NEW-0001-AAAA", "usado": False, "revocado": False}]
        mock_get_resp.raise_for_status = MagicMock()
        mock_get.return_value = mock_get_resp

        mock_del_resp = MagicMock()
        mock_del_resp.raise_for_status = MagicMock()
        mock_del.return_value = mock_del_resp

        resp = client.delete(
            "/api/codes/TPD-NEW-0001-AAAA",
            headers={"X-API-Key": "test-api-key"},
        )
        assert resp.status_code == 200, f"Esperado 200, recibido {resp.status_code}"
        data = resp.json()
        assert data["accion"] == "eliminar", f"Esperado 'eliminar', recibido {data['accion']}"
        mock_del.assert_called_once()


def test_revoke_active_code():
    """DELETE sobre código en uso debe revocar (PATCH a Supabase)."""
    client = get_test_client()
    with patch("httpx.get") as mock_get, patch("httpx.patch") as mock_patch:
        mock_get_resp = MagicMock()
        mock_get_resp.json.return_value = [{"codigo": "TPD-ACT-0001-BBBB", "usado": True, "revocado": False}]
        mock_get_resp.raise_for_status = MagicMock()
        mock_get.return_value = mock_get_resp

        mock_patch_resp = MagicMock()
        mock_patch_resp.raise_for_status = MagicMock()
        mock_patch.return_value = mock_patch_resp

        resp = client.delete(
            "/api/codes/TPD-ACT-0001-BBBB",
            headers={"X-API-Key": "test-api-key"},
        )
        assert resp.status_code == 200, f"Esperado 200, recibido {resp.status_code}"
        data = resp.json()
        assert data["accion"] == "revocar", f"Esperado 'revocar', recibido {data['accion']}"
        mock_patch.assert_called_once()


def test_stats_requires_auth():
    """GET /api/stats debe requerir X-API-Key."""
    client = get_test_client()
    resp = client.get("/api/stats")
    assert resp.status_code == 403, f"Esperado 403, recibido {resp.status_code}"


def test_stats_calculation():
    """GET /api/stats debe calcular correctamente activas, revocadas y disponibles."""
    client = get_test_client()
    mock_data = [
        {"codigo": "A", "usado": False, "revocado": False},
        {"codigo": "B", "usado": True, "revocado": False},
        {"codigo": "C", "usado": True, "revocado": True},
        {"codigo": "D", "usado": True, "revocado": False},
        {"codigo": "E", "usado": False, "revocado": True},
    ]
    with patch("httpx.get") as mock_get:
        mock_resp = MagicMock()
        mock_resp.json.return_value = mock_data
        mock_resp.raise_for_status = MagicMock()
        mock_get.return_value = mock_resp

        resp = client.get("/api/stats", headers={"X-API-Key": "test-api-key"})
        assert resp.status_code == 200, f"Esperado 200, recibido {resp.status_code}"
        data = resp.json()
        assert data["total"] == 5, f"total: esperado 5, recibido {data['total']}"
        assert data["activas"] == 2, f"activas: esperado 2, recibido {data['activas']}"
        assert data["revocadas"] == 2, f"revocadas: esperado 2, recibido {data['revocadas']}"
        assert data["disponibles"] == 1, f"disponibles: esperado 1, recibido {data['disponibles']}"


def test_list_codes_paginated():
    """GET /api/codes debe retornar respuesta paginada con data, total, page, limit."""
    client = get_test_client()
    mock_data = [
        {"codigo": "TPD-AAAA-BBBB-CCCC", "usado": True, "hw_id": "abc", "fecha_activacion": "2026-01-01", "creado_en": "2026-01-01", "revocado": True},
        {"codigo": "TPD-DDDD-EEEE-FFFF", "usado": True, "hw_id": "def", "fecha_activacion": "2026-01-02", "creado_en": "2026-01-01", "revocado": False},
        {"codigo": "TPD-GGGG-HHHH-IIII", "usado": False, "hw_id": None, "fecha_activacion": None, "creado_en": "2026-01-03", "revocado": False},
    ]
    with patch("httpx.get") as mock_get:
        mock_resp = MagicMock()
        mock_resp.json.return_value = mock_data
        mock_resp.raise_for_status = MagicMock()
        mock_get.return_value = mock_resp

        resp = client.get("/api/codes", headers={"X-API-Key": "test-api-key"})
        assert resp.status_code == 200, f"Esperado 200, recibido {resp.status_code}"
        body = resp.json()
        assert "data" in body
        assert "total" in body
        assert "page" in body
        assert "limit" in body
        assert "total_pages" in body
        assert body["total"] == 3
        assert body["page"] == 1
        assert body["limit"] == 50
        assert body["total_pages"] == 1
        for lic in body["data"]:
            assert "revocado" in lic, f"Falta revocado en {lic['codigo']}"
        assert body["data"][0]["revocado"] is True
        assert body["data"][1]["revocado"] is False
        assert body["data"][2]["revocado"] is False


def test_list_codes_status_filter_uses_supabase():
    """El filtro por status debe enviarse como filtro a Supabase, no post-filtrar en Python."""
    client = get_test_client()
    mock_data_activas = [
        {"codigo": "TPD-ACT-0001", "usado": True, "revocado": False, "creado_en": "2026-01-01"},
    ]
    called_urls = []

    def mock_side_effect(url, *args, **kwargs):
        called_urls.append(url)
        mock_resp = MagicMock()
        mock_resp.json.return_value = mock_data_activas
        mock_resp.raise_for_status = MagicMock()
        return mock_resp

    with patch("httpx.get") as mock_get:
        mock_get.side_effect = mock_side_effect
        resp = client.get("/api/codes?status=activa&page=1&limit=10", headers={"X-API-Key": "test-api-key"})
        assert resp.status_code == 200
        body = resp.json()
        assert body["total"] == 1
        assert len(body["data"]) == 1
        # Verify Supabase URLs contain the status filter
        assert any("usado=eq.true" in url and "revocado=eq.false" in url for url in called_urls), \
            f"Status filter not in Supabase URLs: {called_urls}"


def test_list_codes_paginated():
    """POST /api/codes/generate debe requerir X-API-Key."""
    client = get_test_client()
    resp = client.post("/api/codes/generate", json={"cantidad": 1})
    assert resp.status_code == 403, f"Esperado 403, recibido {resp.status_code}"


def test_generate_cantidad_limits():
    """POST /api/codes/generate debe rechazar cantidad fuera de rango."""
    client = get_test_client()
    for cantidad_invalida in [0, 101, -1]:
        resp = client.post(
            "/api/codes/generate",
            json={"cantidad": cantidad_invalida},
            headers={"X-API-Key": "test-api-key"},
        )
        assert resp.status_code == 400, \
            f"cantidad={cantidad_invalida}: esperado 400, recibido {resp.status_code}"

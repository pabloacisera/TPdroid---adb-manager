// Cloudflare Worker — Sistema de Licencias TPDroid
// Endpoints:
//   POST /activar    — Activa un código con un hw_id
//   POST /revalidar  — Revalida un archivo .lic existente

// Environment variables:
//   SUPABASE_URL        — https://<project>.supabase.co
//   SUPABASE_SERVICE_KEY — service_role key (admin)
//   LICENSE_SECRET      — secreto para HMAC

// ─── Rate limiting (in-memory sliding window) ──────────
// Nota: en producción con múltiples isolates esto es aproximado,
// pero sigue siendo una barrera efectiva contra fuerza bruta.
const rateLimitWindows = {
  '/activar':   { max: 5,  windowSec: 60 },
  '/revalidar': { max: 30, windowSec: 60 },
  '/revocar':   { max: 10, windowSec: 60 },
};

const rateLimitStore = new Map();

function cleanOldTimestamps(key, windowMs) {
  const now = Date.now();
  const timestamps = rateLimitStore.get(key) || [];
  const filtered = timestamps.filter(t => now - t < windowMs);
  if (filtered.length === 0) {
    rateLimitStore.delete(key);
  } else {
    rateLimitStore.set(key, filtered);
  }
  return filtered;
}

function checkRateLimit(path, clientIp) {
  const cfg = rateLimitWindows[path];
  if (!cfg) return true;
  const key = `${path}:${clientIp}`;
  const windowMs = cfg.windowSec * 1000;
  const recent = cleanOldTimestamps(key, windowMs);
  if (recent.length >= cfg.max) {
    return false;
  }
  recent.push(Date.now());
  rateLimitStore.set(key, recent);
  return true;
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const method = request.method;

    // CORS preflight
    if (method === 'OPTIONS') {
      return new Response(null, {
        headers: {
          'Access-Control-Allow-Origin': '*',
          'Access-Control-Allow-Methods': 'POST, OPTIONS',
          'Access-Control-Allow-Headers': 'Content-Type',
        },
      });
    }

    if (method !== 'POST') {
      return jsonResponse({ error: 'Method not allowed' }, 405);
    }

    // Rate limiting
    const clientIp = request.headers.get('CF-Connecting-IP') || request.headers.get('X-Forwarded-For') || 'unknown';
    if (!checkRateLimit(url.pathname, clientIp)) {
      return jsonResponse({ error: 'Demasiadas solicitudes. Intente nuevamente en un minuto.' }, 429);
    }

    try {
      switch (url.pathname) {
        case '/activar':
          return handleActivar(request, env);
        case '/revalidar':
          return handleRevalidar(request, env);
        case '/revocar':
          return handleRevocar(request, env);
        default:
          return jsonResponse({ error: 'Not found' }, 404);
      }
    } catch (err) {
      return jsonResponse({ error: err.message }, 500);
    }
  },
};

// ─── Handlers ──────────────────────────────────────────

async function handleActivar(request, env) {
  const { codigo, hw_id, email } = await request.json();

  if (!codigo || !hw_id) {
    return jsonResponse({ error: 'codigo y hw_id son requeridos' }, 400);
  }

  // 1. Buscar código en Supabase
  const licencia = await queryLicencia(env, codigo);
  if (!licencia) {
    return jsonResponse({ error: 'Código de licencia inválido' }, 404);
  }

  if (licencia.revocado) {
    return jsonResponse({ error: 'Esta licencia ha sido revocada' }, 403);
  }

  if (licencia.usado) {
    return jsonResponse({ error: 'Código de licencia ya utilizado' }, 409);
  }

  // 2. Marcar como usado
  const now = new Date().toISOString();
  const updateFields = {
    usado: true,
    hw_id: hw_id,
    fecha_activacion: now,
  };
  if (email) {
    updateFields.email = email;
  }
  const updated = await updateLicencia(env, codigo, updateFields);

  if (!updated) {
    return jsonResponse({ error: 'Error al activar la licencia' }, 500);
  }

  // 3. Generar .lic (firmado con HMAC)
  const licFile = await buildLicFile(codigo, hw_id, now, env.LICENSE_SECRET);

  return jsonResponse({
    success: true,
    message: 'Licencia activada correctamente',
    lic: licFile,
  });
}

async function handleRevalidar(request, env) {
  const { lic, current_hw_id } = await request.json();

  if (!lic || !current_hw_id) {
    return jsonResponse({ error: 'lic y current_hw_id son requeridos' }, 400);
  }

  // 1. Validar HMAC
  const expectedHmac = await computeHmac(
    lic.codigo + lic.hw_id + lic.issued,
    env.LICENSE_SECRET
  );

  if (lic.hmac !== expectedHmac) {
    return jsonResponse({ error: 'Firma de licencia inválida' }, 401);
  }

  // 2. Validar que el hw_id coincida con el actual
  if (lic.hw_id !== current_hw_id) {
    return jsonResponse({ error: 'Esta licencia no corresponde a este equipo' }, 403);
  }

  // 3. Validar contra Supabase
  const dbLic = await queryLicencia(env, lic.codigo);
  if (!dbLic) {
    return jsonResponse({ error: 'Licencia no encontrada en el servidor' }, 404);
  }

  if (dbLic.revocado) {
    return jsonResponse({ error: 'Esta licencia ha sido revocada' }, 403);
  }

  if (dbLic.hw_id !== lic.hw_id) {
    return jsonResponse({ error: 'La licencia fue registrada con otro equipo' }, 403);
  }

  return jsonResponse({
    success: true,
    message: 'Licencia válida',
    hw_id: lic.hw_id,
    issued: lic.issued,
  });
}

async function handleRevocar(request, env) {
  // Requiere header de autenticación para proteger el endpoint
  const adminSecret = request.headers.get('X-Admin-Secret');
  if (!adminSecret || adminSecret !== env.ADMIN_SECRET) {
    return jsonResponse({ error: 'No autorizado' }, 401);
  }

  const { codigo } = await request.json();
  if (!codigo) {
    return jsonResponse({ error: 'codigo es requerido' }, 400);
  }

  const licencia = await queryLicencia(env, codigo);
  if (!licencia) {
    return jsonResponse({ error: 'Código no encontrado' }, 404);
  }

  if (licencia.revocado) {
    return jsonResponse({ error: 'La licencia ya estaba revocada' }, 409);
  }

  const ok = await updateLicencia(env, codigo, { revocado: true });
  if (!ok) {
    return jsonResponse({ error: 'Error al revocar la licencia' }, 500);
  }

  return jsonResponse({ success: true, message: `Licencia ${codigo} revocada` });
}

// ─── Supabase REST helpers ─────────────────────────────

function supabaseHeaders(env) {
  return {
    'Content-Type': 'application/json',
    'apikey': env.SUPABASE_SERVICE_KEY,
    'Authorization': `Bearer ${env.SUPABASE_SERVICE_KEY}`,
    'Prefer': 'return=representation',
  };
}

async function queryLicencia(env, codigo) {
  const url = `${env.SUPABASE_URL}/rest/v1/licencias?codigo=eq.${encodeURIComponent(codigo)}&limit=1`;
  const res = await fetch(url, {
    headers: supabaseHeaders(env),
  });
  if (!res.ok) return null;
  const data = await res.json();
  return data && data.length > 0 ? data[0] : null;
}

async function updateLicencia(env, codigo, fields) {
  const url = `${env.SUPABASE_URL}/rest/v1/licencias?codigo=eq.${encodeURIComponent(codigo)}`;
  const res = await fetch(url, {
    method: 'PATCH',
    headers: supabaseHeaders(env),
    body: JSON.stringify(fields),
  });
  return res.ok;
}

// ─── HMAC ──────────────────────────────────────────────

async function buildLicFile(codigo, hw_id, issued, secret) {
  const payload = codigo + hw_id + issued;
  const hmac = await computeHmac(payload, secret);
  return {
    codigo: codigo,
    hw_id: hw_id,
    issued: issued,
    hmac: hmac,
  };
}

async function computeHmac(payload, secret) {
  const encoder = new TextEncoder();
  const key = await crypto.subtle.importKey(
    'raw',
    encoder.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );
  const signature = await crypto.subtle.sign('HMAC', key, encoder.encode(payload));
  return bytesToHex(new Uint8Array(signature));
}

function bytesToHex(bytes) {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

// ─── Response helper ───────────────────────────────────

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*',
    },
  });
}

-- Schema para Supabase — Tabla de licencias
-- Ejecutar en SQL Editor de Supabase
-- Última verificación contra producción: 2026-06-29

CREATE TABLE IF NOT EXISTS licencias (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  codigo VARCHAR(32) UNIQUE NOT NULL,
  usado BOOLEAN NOT NULL DEFAULT false,
  hw_id TEXT,
  email TEXT,
  fecha_activacion TIMESTAMPTZ,
  revocado BOOLEAN NOT NULL DEFAULT false,
  creado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_licencias_codigo ON licencias (codigo);
CREATE INDEX IF NOT EXISTS idx_licencias_hw_id ON licencias (hw_id);

-- Migraciones para bases existentes:
-- ALTER TABLE licencias ADD COLUMN IF NOT EXISTS revocado BOOLEAN NOT NULL DEFAULT false;
-- ALTER TABLE licencias ADD COLUMN IF NOT EXISTS email TEXT;

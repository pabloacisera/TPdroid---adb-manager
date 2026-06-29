#!/bin/bash
# test-install-flow.sh — Test completo del flujo de instalación de TPDroid
# 1. Compila el tar  →  2. Extrae  →  3. Ejecuta install.sh
# → 4. Activador corre  →  5. Cliente ingresa código+email
# → 6. Se crea licencia  →  7. Instalación continúa
# → 8. Se crea icono escritorio
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$PROJECT_DIR/dist"
TMPDIR="${TMPDIR:-/tmp}/tpdroid-test-$$"
MOCK_PORT=18787
MOCK_URL="http://127.0.0.1:$MOCK_PORT"
LOG="$TMPDIR/test.log"
PASS=0
FAIL=0
CODE="TPD-6OZQ-46VO-5XYJ"
EMAIL="cliente@ejemplo.com"

mkdir -p "$TMPDIR"

assert() {
  local desc="$1"
  shift
  if "$@"; then
    PASS=$((PASS + 1))
    echo "  [PASS] $desc"
  else
    FAIL=$((FAIL + 1))
    echo "  [FAIL] $desc"
  fi
}

# ─── 1. Mock Worker ─────────────────────────────────────────
echo "=== 1. Iniciando mock worker en $MOCK_URL ==="
python3 -c "
import http.server, json, hmac, hashlib, sys

class H(http.server.BaseHTTPRequestHandler):
    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin','*')
        self.send_header('Access-Control-Allow-Methods','POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers','Content-Type')
        self.end_headers()
    def do_POST(self):
        length = int(self.headers.get('Content-Length',0))
        body = json.loads(self.rfile.read(length))
        if self.path == '/activar':
            codigo, hw_id = body.get('codigo',''), body.get('hw_id','')
            issued = '2026-06-29T12:00:00Z'
            payload = codigo + hw_id + issued
            sig = hmac.new(b'test-secret', payload.encode(), hashlib.sha256).hexdigest()
            lic = {'codigo':codigo,'hw_id':hw_id,'issued':issued,'hmac':sig}
            resp = {'success':True,'message':'OK','lic':lic}
            self.send_response(200)
            self.send_header('Content-Type','application/json')
            self.send_header('Access-Control-Allow-Origin','*')
            self.end_headers()
            self.wfile.write(json.dumps(resp).encode())
        else:
            self.send_response(404)
            self.end_headers()
    def log_message(self,*a): pass

s = http.server.HTTPServer(('127.0.0.1',$MOCK_PORT), H)
s.serve_forever()
" &
MOCK_PID=$!
sleep 1

if kill -0 $MOCK_PID 2>/dev/null; then
  echo "  [OK] Mock worker PID $MOCK_PID"
else
  echo "  [FAIL] Mock worker no arrancó"
  exit 1
fi

# ─── 2. Build tar ────────────────────────────────────────────
echo ""
echo "=== 2. Construyendo tpdroid-linux.tar.gz ==="
export LICENSE_WORKER_URL="$MOCK_URL"
cd "$PROJECT_DIR"

# Compilar backend
cd "$PROJECT_DIR/backend"
GOOS=linux GOARCH=amd64 go build -o "$DIST_DIR/tpdroid" . 2>>"$LOG"
echo "  [OK] backend compilado"

# Compilar activator con ldflags
cd "$PROJECT_DIR/activator"
GOOS=linux GOARCH=amd64 go build \
  -ldflags="-X main.defaultWorkerURL=${MOCK_URL}" \
  -o "$DIST_DIR/activator-linux" . 2>>"$LOG"
echo "  [OK] activator compilado"

# Copiar ADB, install.sh, icono
cp "$PROJECT_DIR/adb-binaries/linux/adb" "$DIST_DIR/adb"
chmod +x "$DIST_DIR/adb"
cp "$PROJECT_DIR/install.sh" "$DIST_DIR/install.sh"
chmod +x "$DIST_DIR/install.sh"
cp "$PROJECT_DIR/tpdroid-icon-without-background.png" "$DIST_DIR/tpdroid-icon.png"

# Crear tar
cd "$PROJECT_DIR"
tar czf "$DIST_DIR/tpdroid-linux.tar.gz" \
  -C "$DIST_DIR" \
  tpdroid activator-linux adb install.sh tpdroid-icon.png

echo "  [OK] tpdroid-linux.tar.gz creado"
assert "existe tarball" test -f "$DIST_DIR/tpdroid-linux.tar.gz"

# ─── 3. Extraer tar ──────────────────────────────────────────
echo ""
echo "=== 3. Extrayendo tarball ==="
EXTRACT_DIR="$TMPDIR/extracted"
mkdir -p "$EXTRACT_DIR"
tar xzf "$DIST_DIR/tpdroid-linux.tar.gz" -C "$EXTRACT_DIR"
echo "  [OK] extraído en $EXTRACT_DIR"
ls -la "$EXTRACT_DIR"

for f in tpdroid activator-linux adb install.sh tpdroid-icon.png; do
  assert "existe $f en extraídos" test -f "$EXTRACT_DIR/$f"
done

# ─── 4. Ejecutar install.sh (background) ─────────────────────
echo ""
echo "=== 4. Ejecutando install.sh ==="
# Asegurar que no hay instalación previa
rm -f "$HOME/.config/tpdroid/license.lic"

# install.sh ejecuta activator-linux que abre un servidor web
# Lo corremos en background
cd "$EXTRACT_DIR"
HOME="$TMPDIR/home" \
XDG_CONFIG_HOME="$TMPDIR/home/.config" \
XDG_DATA_HOME="$TMPDIR/home/.local/share" \
bash install.sh &
INSTALL_PID=$!
echo "  install.sh PID: $INSTALL_PID"

# Esperar a que el activator inicie el servidor (buscar puerto)
ACTIVATOR_PORT=""
for i in $(seq 1 20); do
  # Buscar qué puerto abrió el activator
  ACTIVATOR_PORT=$(ss -tlnp 2>/dev/null | grep -oP '127\.0\.0\.1:\K\d+(?=.*activator)' || echo "")
  if [ -n "$ACTIVATOR_PORT" ]; then
    break
  fi
  # También puede estar en cualquier puerto alto (port 0)
  ACTIVATOR_PORT=$(ss -tlnp 2>/dev/null | awk '/activator/ {split($4,a,":"); print a[2]}' | head -1 || echo "")
  if [ -n "$ACTIVATOR_PORT" ]; then
    break
  fi
  sleep 0.5
done

if [ -z "$ACTIVATOR_PORT" ]; then
  echo "  [WARN] No se detectó puerto del activator por nombre. Buscando puerto 8787/18787..."
  # Podría estar escuchando en el puerto default. Verificar activamente curl.
  for try_port in 8787 18787; do
    if curl -s "http://127.0.0.1:$try_port/" >/dev/null 2>&1; then
      ACTIVATOR_PORT=$try_port
      break
    fi
  done
  # Buscar cualquier puerto local con respuesta HTTP
  for try_port in $(seq 49152 65535); do
    if curl -s --connect-timeout 0.2 "http://127.0.0.1:$try_port/" >/dev/null 2>&1; then
      ACTIVATOR_PORT=$try_port
      break
    fi
  done
fi

echo "  Activador detectado en puerto: ${ACTIVATOR_PORT:-DESCONOCIDO}"

# ─── 5. Enviar activación (simular el formulario) ────────────
echo ""
echo "=== 5. Simulando activación ==="
if [ -n "$ACTIVATOR_PORT" ]; then
  ACTIVATE_URL="http://127.0.0.1:$ACTIVATOR_PORT/activate"
  echo "  POST a $ACTIVATE_URL"
  ACTIVATE_RESP=$(curl -s -X POST "$ACTIVATE_URL" \
    -H "Content-Type: application/json" \
    -d "{\"codigo\":\"$CODE\",\"email\":\"$EMAIL\"}")
  echo "  Respuesta: $ACTIVATE_RESP"
  
  SUCCESS=$(echo "$ACTIVATE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success','false'))" 2>/dev/null || echo "false")
  assert "activación exitosa" test "$SUCCESS" = "True"
else
  echo "  [WARN] No se pudo detectar puerto del activator. Intentando servir mock local..."
  # Si no se detectó, el activator puede haber fallado
fi

# Esperar a que install.sh termine
echo ""
echo "  Esperando que install.sh termine..."
wait $INSTALL_PID 2>/dev/null || true
echo "  install.sh finalizó"

# ─── 6. Verificar licencia ───────────────────────────────────
echo ""
echo "=== 6. Verificando licencia ==="
LICENSE_PATH="$TMPDIR/home/.config/tpdroid/license.lic"
if [ -f "$LICENSE_PATH" ]; then
  echo "  Licencia encontrada en $LICENSE_PATH"
  cat "$LICENSE_PATH"
  assert "archivo .lic existe" test -f "$LICENSE_PATH"
  LIC_CODE=$(python3 -c "import json; print(json.load(open('$LICENSE_PATH')).get('codigo',''))" 2>/dev/null || echo "")
  assert "código en lic coincide" test "$LIC_CODE" = "$CODE"
else
  echo "  [FAIL] No se encontró licencia en $LICENSE_PATH"
  # Buscar en ubicación por defecto alternativa
  find "$TMPDIR" -name "*.lic" 2>/dev/null || echo "  No se encontró ningún .lic"
fi

# ─── 7. Verificar instalación ────────────────────────────────
echo ""
echo "=== 7. Verificando archivos instalados ==="
INSTALL_DIR="$TMPDIR/home/.local/share/tpdroid"
BIN_DIR="$TMPDIR/home/.local/bin"
DESKTOP_DIR="$TMPDIR/home/.local/share/applications"

ls -la "$INSTALL_DIR" 2>/dev/null || echo "  [WARN] $INSTALL_DIR no existe"
ls -la "$BIN_DIR" 2>/dev/null || echo "  [WARN] $BIN_DIR no existe"
ls -la "$DESKTOP_DIR" 2>/dev/null || echo "  [WARN] $DESKTOP_DIR no existe"

assert "tpdroid instalado" test -f "$INSTALL_DIR/tpdroid"
assert "ADB instalado" test -f "$INSTALL_DIR/adb-binaries/linux/adb"
assert "tpdroid.sh script" test -f "$INSTALL_DIR/tpdroid.sh"
assert "enlace simbólico" test -L "$BIN_DIR/tpdroid"

# ─── 8. Verificar icono escritorio ───────────────────────────
echo ""
echo "=== 8. Verificando icono de escritorio ==="
DESKTOP_FILE="$DESKTOP_DIR/tpdroid.desktop"
if [ -f "$DESKTOP_FILE" ]; then
  echo "  Desktop entry:"
  cat "$DESKTOP_FILE"
  assert "desktop entry existe" test -f "$DESKTOP_FILE"
  assert "Exec en desktop entry" grep -q "Exec=" "$DESKTOP_FILE"
  assert "Icon en desktop entry" grep -q "Icon=" "$DESKTOP_FILE"
  assert "Name en desktop entry" grep -q "Name=TPDroid" "$DESKTOP_FILE"
else
  echo "  [FAIL] No se encontró $DESKTOP_FILE"
fi

# ─── 9. Verificar binario tpdroid funciona ──────────────────
echo ""
echo "=== 9. Verificando binario ==="
if [ -f "$INSTALL_DIR/tpdroid" ]; then
  file "$INSTALL_DIR/tpdroid"
  assert "es ejecutable" test -x "$INSTALL_DIR/tpdroid"
else
  echo "  [SKIP] tpdroid no disponible"
fi

# ─── Resumen ─────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════"
echo "  RESULTADOS: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════════"

# Limpiar mock worker
kill $MOCK_PID 2>/dev/null || true

if [ "$FAIL" -gt 0 ]; then
  echo "  Log: $LOG"
  exit 1
else
  echo "  ¡Flujo de instalación completado exitosamente!"
  exit 0
fi

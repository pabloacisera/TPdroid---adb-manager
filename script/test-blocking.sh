#!/bin/bash
# test-blocking.sh — Verifica el pipeline completo de bloqueo de notificaciones
# Requiere: dispositivo conectado y servidor TPDroid corriendo en :8080
# Uso: ./script/test-blocking.sh
#
# Mejoras respecto a la versión anterior:
# - No usa com.android.shell (es system app → 403). Busca un package real no-system.
# - No usa `su -lp 2000` (innecesario para cmd notification post).
# - No usa `cmd notification dismiss` (no existe en SDK 34).
# - Verifica con appops directo para block-full.
# - Trap para restaurar estado aunque falle un test.

set -euo pipefail

BASE="http://localhost:8080"
PASS=0
FAIL=0
TOTAL=0

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TEST_PKG=""
TEST_CHANNELS="[]"

# Determinar ADB path
ADB=""
if [ -f "$SCRIPT_DIR/dist/tpdroid_fixed" ]; then
  case "$(uname -s)" in
    Linux)   ADB="$SCRIPT_DIR/adb-binaries/linux/adb" ;;
    Darwin)  ADB="$SCRIPT_DIR/adb-binaries/macos/adb" ;;
    *)       echo "Error: plataforma no soportada"; exit 1 ;;
  esac
elif command -v adb &>/dev/null; then
  ADB="adb"
else
  echo "Error: no se encuentra adb"
  exit 1
fi

if [ ! -x "$ADB" ]; then
  echo "Error: $ADB no es ejecutable"
  exit 1
fi

echo "=== ADB: $ADB ==="
echo ""

# ─── helpers ──────────────────────────────────────────────

check() {
  TOTAL=$((TOTAL + 1))
  local desc="$1"
  local result="$2"
  if [ "$result" = "ok" ]; then
    PASS=$((PASS + 1))
    echo "  ✅ $desc"
  else
    FAIL=$((FAIL + 1))
    echo "  ❌ $desc"
  fi
}

summary() {
  echo ""
  echo "═══════════════════════════════════════"
  if [ "$FAIL" -eq 0 ]; then
    echo "  ✅ $PASS/$TOTAL tests passed"
    return 0
  else
    echo "  ❌ $FAIL/$TOTAL tests failed"
    return 1
  fi
}

# Restaurar estado del package de prueba al finalizar
cleanup() {
  if [ -n "$TEST_PKG" ]; then
    echo "   Limpiando: restaurando $TEST_PKG..."
    curl -sf -X POST "$BASE/api/ads/unblock" \
      "${AUTH_ARGS[@]}" \
      -H "Content-Type: application/json" \
      -d "{\"package\":\"$TEST_PKG\",\"blocked_channels\":[],\"sdk_version\":\"$SDK\"}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

# ─── 1. Verificar dispositivo conectado ───────────────────

echo "--- 1. Verificando dispositivo ---"
SERIAL=$($ADB devices | awk 'NR==2 {print $1}')
if [ -z "$SERIAL" ] || [ "$SERIAL" = "" ]; then
  echo "❌ No hay dispositivo conectado via ADB"
  exit 1
fi
echo "   Serial: $SERIAL"

SDK=$($ADB -s "$SERIAL" shell getprop ro.build.version.sdk | tr -d '\r')
echo "   SDK: $SDK"
echo ""

# ─── 2. Verificar servidor ─────────────────────────────────

echo "--- 2. Verificando servidor ---"
if ! curl -sf "$BASE/api/status" >/dev/null 2>&1; then
  echo "❌ Servidor TPDroid no responde en $BASE"
  echo "   Ejecutá: ./script/dev.sh"
  exit 1
fi
echo "   OK"

echo "--- Obteniendo session token ---"
SESSION_TOKEN=$(curl -sf "$BASE/api/session-token" 2>/dev/null | python3 -c \
  "import sys,json; print(json.load(sys.stdin).get('token',''))" 2>/dev/null || echo "")
AUTH_ARGS=()
if [ -n "$SESSION_TOKEN" ]; then
  echo "   Token obtenido"
  AUTH_ARGS=(-H "X-Session-Token: $SESSION_TOKEN")
else
  echo "   ⚠️  No se pudo obtener session token — POST requests podrian fallar con 403"
fi
echo ""

# ─── 3. Scan para detectar package de prueba ───────────────

echo "--- 3. Test: GET /api/ads/scan ---"
SCAN=$(curl -sf "$BASE/api/ads/scan" 2>/dev/null || echo "[]")
PKG_COUNT=$(echo "$SCAN" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
check "Scan retorna lista (encontrados: $PKG_COUNT)" "$([ "$PKG_COUNT" -gt 0 ] 2>/dev/null && echo "ok" || echo "fail")"

# Buscar un package no-system con notif_channels
echo "   Buscando package no-system con canales de notificación..."
TEST_PKG=$(echo "$SCAN" | python3 -c "
import sys,json
data=json.load(sys.stdin)
for e in data:
    if not e.get('is_system_app', True) and e.get('notif_channels') and len(e['notif_channels']) > 0:
        print(e['package'])
        break
" 2>/dev/null || echo "")
TEST_CHANNELS=$(echo "$SCAN" | python3 -c "
import sys,json
data=json.load(sys.stdin)
for e in data:
    if not e.get('is_system_app', True) and e.get('notif_channels') and len(e['notif_channels']) > 0:
        print(json.dumps(e['notif_channels']))
        break
" 2>/dev/null || echo "[]")

if [ -z "$TEST_PKG" ]; then
  echo "⚠️  No se encontró package no-system con canales. Probando con com.google.android.youtube..."
  TEST_PKG="com.google.android.youtube"
  TEST_CHANNELS='["1","2"]'
fi
echo "   Package de prueba: $TEST_PKG"
echo "   Canales: $TEST_CHANNELS"
echo ""

# ─── 4. Postear notif bajo com.android.shell (solo demostrativo) ──

echo "--- Posteando notificación de prueba (bajo com.android.shell) ---"
$ADB -s "$SERIAL" shell cmd notification post -t "Test TPDroid" test_tpdroid_tag "Notificacion de prueba TPDroid" 2>/dev/null || true
sleep 1
SCAN2=$(curl -sf "$BASE/api/ads/scan" 2>/dev/null || echo "[]")
SHELL_FOUND=$(echo "$SCAN2" | python3 -c "
import sys,json
data=json.load(sys.stdin)
print(any(e.get('package','') == 'com.android.shell' for e in data))
" 2>/dev/null || echo "False")
check "Scan detecta notificación posteada (com.android.shell)" "$([ "$SHELL_FOUND" = "True" ] && echo "ok" || echo "fail")"
echo ""

# ─── 5. Test: bloqueo inteligente por canal ────────────────

echo "--- 5. Test: bloqueo por canal específico (smart block) ---"
BLOCK=$(curl -sf -X POST "$BASE/api/ads/block" \
  "${AUTH_ARGS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"package\":\"$TEST_PKG\",\"channels\":$TEST_CHANNELS,\"sdk_version\":\"$SDK\"}" 2>/dev/null || echo '{"success":false}')
BLOCK_OK=$(echo "$BLOCK" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success',False))" 2>/dev/null)
BLOCKED_CH=$(echo "$BLOCK" | python3 -c "import sys,json; d=json.load(sys.stdin); print(json.dumps(d.get('blocked_channels',[])))" 2>/dev/null)
FULL_BL=$(echo "$BLOCK" | python3 -c "import sys,json; print(json.load(sys.stdin).get('full_blocked',False))" 2>/dev/null)
check "Block respuesta success=true" "$([ "$BLOCK_OK" = "True" ] && echo "ok" || echo "fail")"
if [ "$SDK" -ge 33 ]; then
  # En SDK 33+, cmd notification disable-channel no existe → fallback a full block
  check "Block: fallback a full block (SDK $SDK no soporta channel block)" "$([ "$FULL_BL" = "True" ] && echo "ok" || echo "fail")"
else
  # En SDK 26-32, debería bloquear por canal
  check "Block: bloqueo por canal (blocked_channels no vacío)" "$([ "$BLOCKED_CH" != "[]" ] && [ "$BLOCKED_CH" != '""' ] && echo "ok" || echo "fail")"
  check "Block: full_blocked=false" "$([ "$FULL_BL" = "False" ] && echo "ok" || echo "fail")"
fi
echo "   blocked_channels: $BLOCKED_CH"
echo "   full_blocked: $FULL_BL"
echo ""

# ─── 6. Test: desbloqueo por canal ─────────────────────────

echo "--- 6. Test: desbloqueo por canal ---"
# Usar los blocked channels devueltos por el block (pueden ser null si fue full block)
if [ "$BLOCKED_CH" = "null" ] || [ "$BLOCKED_CH" = "[]" ]; then
  UNBLOCK_CH="[]"
else
  UNBLOCK_CH="$BLOCKED_CH"
fi
UNBLOCK=$(curl -sf -X POST "$BASE/api/ads/unblock" \
  "${AUTH_ARGS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"package\":\"$TEST_PKG\",\"blocked_channels\":$UNBLOCK_CH,\"sdk_version\":\"$SDK\"}" 2>/dev/null || echo '{"success":false}')
UNBLOCK_OK=$(echo "$UNBLOCK" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success',False))" 2>/dev/null)
check "Unblock respuesta success=true" "$([ "$UNBLOCK_OK" = "True" ] && echo "ok" || echo "fail")"
echo ""

# ─── 6. Test: desbloqueo por canal ─────────────────────────

echo "--- 6. Test: desbloqueo por canal ---"
UNBLOCK=$(curl -sf -X POST "$BASE/api/ads/unblock" \
  "${AUTH_ARGS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"package\":\"$TEST_PKG\",\"blocked_channels\":$BLOCKED_CH,\"sdk_version\":\"$SDK\"}" 2>/dev/null || echo '{"success":false}')
UNBLOCK_OK=$(echo "$UNBLOCK" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success',False))" 2>/dev/null)
check "Unblock respuesta success=true" "$([ "$UNBLOCK_OK" = "True" ] && echo "ok" || echo "fail")"
echo ""

# ─── 7. Test: bloqueo total (block-full) ──────────────────

echo "--- 7. Test: bloqueo total (block-full) ---"
FULL=$(curl -sf -X POST "$BASE/api/ads/block-full" \
  "${AUTH_ARGS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"package\":\"$TEST_PKG\"}" 2>/dev/null || echo '{"success":false}')
FULL_OK=$(echo "$FULL" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success',False))" 2>/dev/null)
FULL_BL2=$(echo "$FULL" | python3 -c "import sys,json; print(json.load(sys.stdin).get('full_blocked',False))" 2>/dev/null)
check "Block-full respuesta success=true" "$([ "$FULL_OK" = "True" ] && echo "ok" || echo "fail")"
check "Block-full full_blocked=true" "$([ "$FULL_BL2" = "True" ] && echo "ok" || echo "fail")"

APPOPS=$($ADB -s "$SERIAL" shell appops get "$TEST_PKG" POST_NOTIFICATION 2>/dev/null | tr -d '\r' || echo "")
check "appops POST_NOTIFICATION=bloqueado (deny|ignore)" "$(echo "$APPOPS" | grep -qE "deny|ignore" && echo "ok" || echo "fail")"
echo ""

# ─── 8. Test: restaurar después de bloqueo total ──────────

echo "--- 8. Test: restaurar después de bloqueo total ---"
RESTORE=$(curl -sf -X POST "$BASE/api/ads/unblock" \
  "${AUTH_ARGS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"package\":\"$TEST_PKG\",\"blocked_channels\":[],\"sdk_version\":\"$SDK\"}" 2>/dev/null || echo '{"success":false}')
RESTORE_OK=$(echo "$RESTORE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success',False))" 2>/dev/null)
check "Restore respuesta success=true" "$([ "$RESTORE_OK" = "True" ] && echo "ok" || echo "fail")"

APPOPS2=$($ADB -s "$SERIAL" shell appops get "$TEST_PKG" POST_NOTIFICATION 2>/dev/null | tr -d '\r' || echo "")
check "appops POST_NOTIFICATION=allow (restaurado)" "$(echo "$APPOPS2" | grep -q "allow" && echo "ok" || echo "fail")"
echo ""

# ─── 9. Test: guard de sistema (403 en 3 endpoints) ───────

echo "--- 9. Test: guard de sistema retorna 403 ---"
# block
SYS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/ads/block" \
  "${AUTH_ARGS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"package\":\"com.android.systemui\",\"channels\":[],\"sdk_version\":\"$SDK\"}" 2>/dev/null || echo "000")
check "POST /api/ads/block → systemui → 403" "$([ "$SYS" = "403" ] && echo "ok" || echo "fail")"

# block-full
SYS2=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/ads/block-full" \
  "${AUTH_ARGS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"package\":\"com.android.systemui\"}" 2>/dev/null || echo "000")
check "POST /api/ads/block-full → systemui → 403" "$([ "$SYS2" = "403" ] && echo "ok" || echo "fail")"

# unblock
SYS3=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/ads/unblock" \
  "${AUTH_ARGS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"package\":\"com.android.systemui\",\"blocked_channels\":[],\"sdk_version\":\"$SDK\"}" 2>/dev/null || echo "000")
check "POST /api/ads/unblock → systemui → 403" "$([ "$SYS3" = "403" ] && echo "ok" || echo "fail")"
echo ""

# ─── 10. Test: smart block sin canales → bloqueo total ────

echo "--- 10. Test: smart block sin canales → bloqueo total (fallback) ---"
BLOCK_NOCH=$(curl -sf -X POST "$BASE/api/ads/block" \
  "${AUTH_ARGS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"package\":\"$TEST_PKG\",\"channels\":[],\"sdk_version\":\"$SDK\"}" 2>/dev/null || echo '{"success":false}')
BLOCK_NOCH_OK=$(echo "$BLOCK_NOCH" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success',False))" 2>/dev/null)
BLOCK_NOCH_FULL=$(echo "$BLOCK_NOCH" | python3 -c "import sys,json; print(json.load(sys.stdin).get('full_blocked',False))" 2>/dev/null)
check "Block sin channels → success=true" "$([ "$BLOCK_NOCH_OK" = "True" ] && echo "ok" || echo "fail")"
check "Block sin channels → full_blocked=true" "$([ "$BLOCK_NOCH_FULL" = "True" ] && echo "ok" || echo "fail")"

APPOPS3=$($ADB -s "$SERIAL" shell appops get "$TEST_PKG" POST_NOTIFICATION 2>/dev/null | tr -d '\r' || echo "")
check "appops POST_NOTIFICATION=bloqueado (deny|ignore)" "$(echo "$APPOPS3" | grep -qE "deny|ignore" && echo "ok" || echo "fail")"
echo ""

# cleanup via trap

# ─── Resultado ─────────────────────────────────────────────

summary

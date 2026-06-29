#!/bin/bash
# build-macos.sh — Genera tpdroid-macos.tar.gz
# Puede ejecutarse desde macOS nativo o desde Linux (cross-compile).
# Uso: ./script/build-macos.sh
#
# Desde Linux (CI):  GOOS=darwin se pasa automáticamente
# Desde macOS:       compila nativo, el resultado es igual
#
# Prerequisito (solo para quien compila):
#   go 1.21+ instalado  →  https://go.dev/dl/
#
# El usuario final NO necesita Go. Solo descarga el tarball y ejecuta install-macos.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$SCRIPT_DIR/dist"
WORKER_URL="${LICENSE_WORKER_URL:-https://tpdroid-licencias.testdeveloperrandom.workers.dev}"

command -v go >/dev/null 2>&1 || { echo "Error: go no instalado"; exit 1; }
mkdir -p "$DIST_DIR"

# ── Compilar backend ─────────────────────────────────
echo "Compilando tpdroid para macOS (amd64)..."
cd "$SCRIPT_DIR/backend"
GOOS=darwin GOARCH=amd64 go build -o "$DIST_DIR/tpdroid-macos-amd64" .
echo "-> $DIST_DIR/tpdroid-macos-amd64"

echo "Compilando tpdroid para macOS (arm64 / Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -o "$DIST_DIR/tpdroid-macos-arm64" .
echo "-> $DIST_DIR/tpdroid-macos-arm64"

# ── Universal binary (lipo) — solo si se compila desde macOS ──────────────
if command -v lipo >/dev/null 2>&1; then
  echo "Creando universal binary (lipo)..."
  lipo -create -output "$DIST_DIR/tpdroid-macos" \
    "$DIST_DIR/tpdroid-macos-amd64" \
    "$DIST_DIR/tpdroid-macos-arm64"
  rm "$DIST_DIR/tpdroid-macos-amd64" "$DIST_DIR/tpdroid-macos-arm64"
  echo "-> $DIST_DIR/tpdroid-macos  (universal: amd64 + arm64)"
else
  # En CI (Linux) no hay lipo — distribuir los dos binarios por separado
  echo "lipo no disponible (CI Linux): se distribuyen dos binarios separados"
fi

# ── Compilar activator para macOS ────────────────────
echo "Compilando activator-macos..."
cd "$SCRIPT_DIR/activator"
GOOS=darwin GOARCH=amd64 go build \
  -ldflags="-X main.defaultWorkerURL=${WORKER_URL}" \
  -o "$DIST_DIR/activator-macos-amd64" .

GOOS=darwin GOARCH=arm64 go build \
  -ldflags="-X main.defaultWorkerURL=${WORKER_URL}" \
  -o "$DIST_DIR/activator-macos-arm64" .

if command -v lipo >/dev/null 2>&1; then
  lipo -create -output "$DIST_DIR/activator-macos" \
    "$DIST_DIR/activator-macos-amd64" \
    "$DIST_DIR/activator-macos-arm64"
  rm "$DIST_DIR/activator-macos-amd64" "$DIST_DIR/activator-macos-arm64"
fi
echo "-> activator-macos compilado"

# ── Copiar ADB binary para macOS ─────────────────────
ADB_SRC="$SCRIPT_DIR/adb-binaries/macos/adb"
if [ ! -f "$ADB_SRC" ]; then
  echo "Error: falta adb-binaries/macos/adb"
  exit 1
fi
cp "$ADB_SRC" "$DIST_DIR/adb-macos"
chmod +x "$DIST_DIR/adb-macos"
echo "-> adb-macos copiado"

# ── Copiar install-macos.sh ───────────────────────────
cp "$SCRIPT_DIR/install-macos.sh" "$DIST_DIR/install-macos.sh"
chmod +x "$DIST_DIR/install-macos.sh"

# ── Crear tarball ─────────────────────────────────────
cd "$SCRIPT_DIR"

# Si hay lipo (macOS nativo) → un solo binario universal
if [ -f "$DIST_DIR/tpdroid-macos" ]; then
  tar czf "$DIST_DIR/tpdroid-macos.tar.gz" \
    -C "$DIST_DIR" \
    tpdroid-macos activator-macos adb-macos install-macos.sh
else
  # CI Linux → dos tarballs separados por arquitectura
  tar czf "$DIST_DIR/tpdroid-macos-amd64.tar.gz" \
    -C "$DIST_DIR" \
    tpdroid-macos-amd64 activator-macos-amd64 adb-macos install-macos.sh

  tar czf "$DIST_DIR/tpdroid-macos-arm64.tar.gz" \
    -C "$DIST_DIR" \
    tpdroid-macos-arm64 activator-macos-arm64 adb-macos install-macos.sh

  echo "-> $DIST_DIR/tpdroid-macos-amd64.tar.gz"
  echo "-> $DIST_DIR/tpdroid-macos-arm64.tar.gz"
fi

echo "OK: Archivos macOS listos en $DIST_DIR/"
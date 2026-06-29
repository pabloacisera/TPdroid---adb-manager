#!/bin/bash
# build-linux.sh — Genera tpdroid-linux.tar.gz
# Uso: ./script/build-linux.sh
# Requisito: go instalado

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$SCRIPT_DIR/dist"
WORKER_URL="${LICENSE_WORKER_URL:-https://tpdroid-licencias.testdeveloperrandom.workers.dev}"

command -v go >/dev/null 2>&1 || { echo "Error: go no instalado"; exit 1; }
mkdir -p "$DIST_DIR"

# Compilar backend
echo "Compilando tpdroid para Linux..."
cd "$SCRIPT_DIR/backend"
GOOS=linux GOARCH=amd64 go build -o "$DIST_DIR/tpdroid" .
echo "-> $DIST_DIR/tpdroid"

# Compilar activator
echo "Compilando activator-linux..."
cd "$SCRIPT_DIR/activator"
GOOS=linux GOARCH=amd64 go build \
  -ldflags="-X main.defaultWorkerURL=${WORKER_URL}" \
  -o "$DIST_DIR/activator-linux" .
echo "-> $DIST_DIR/activator-linux"

# Copiar ADB binary
cp "$SCRIPT_DIR/adb-binaries/linux/adb" "$DIST_DIR/adb"
chmod +x "$DIST_DIR/adb"

# Copiar install.sh e icono
cp "$SCRIPT_DIR/install.sh" "$DIST_DIR/install.sh"
chmod +x "$DIST_DIR/install.sh"
cp "$SCRIPT_DIR/tpdroid-icon-without-background.png" "$DIST_DIR/tpdroid-icon.png"

# Crear tarball
cd "$SCRIPT_DIR"
tar czf "$DIST_DIR/tpdroid-linux.tar.gz" \
  -C "$DIST_DIR" \
  tpdroid activator-linux adb install.sh tpdroid-icon.png

echo "-> $DIST_DIR/tpdroid-linux.tar.gz"
echo "OK: Archivo listo para distribuir."

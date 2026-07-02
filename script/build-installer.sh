#!/bin/bash
# build-installer.sh — Genera TPDroid-Setup.exe desde Linux
# Requisito: sudo apt install nsis
# Uso: ./script/build-installer.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$SCRIPT_DIR/dist"

# 1. Verificar prerequisitos
command -v makensis >/dev/null 2>&1 || { echo "Error: nsis no instalado. Ejecutá: sudo apt install nsis"; exit 1; }
command -v go >/dev/null 2>&1 || { echo "Error: go no instalado"; exit 1; }

# 2. Verificar que existe adb.exe para Windows
if [ ! -f "$SCRIPT_DIR/adb-binaries/windows/adb.exe" ]; then
  echo "Error: falta adb-binaries/windows/adb.exe"
  exit 1
fi

# 3. Crear dist/ si no existe
mkdir -p "$DIST_DIR"

VERSION="${VERSION:-dev}"

# 4. Compilar binario Windows
echo "Compilando tpdroid.exe para Windows (v${VERSION})..."
cd "$SCRIPT_DIR/backend"
GOOS=windows GOARCH=amd64 go build -ldflags="-X main.Version=${VERSION}" -o "$DIST_DIR/tpdroid.exe" .
echo "-> $DIST_DIR/tpdroid.exe"

# 5. Compilar activator.exe
echo "Compilando activator.exe..."
cd "$SCRIPT_DIR/activator"
GOOS=windows GOARCH=amd64 go build \
  -ldflags="-X main.defaultWorkerURL=${LICENSE_WORKER_URL:-https://tpdroid-licencias.testdeveloperrandom.workers.dev}" \
  -o "$DIST_DIR/activator.exe" .
echo "-> $DIST_DIR/activator.exe"

# 6. Generar instalador NSIS
echo "Generando TPDroid-Setup.exe..."
cd "$SCRIPT_DIR"
makensis installer/tpdroid.nsi
echo "-> $DIST_DIR/TPDroid-Setup.exe"
echo "OK: Instalador listo en $DIST_DIR/TPDroid-Setup.exe"

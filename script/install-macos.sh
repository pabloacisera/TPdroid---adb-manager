#!/bin/bash
# install-macos.sh — Instalador de TPDroid para macOS
# Se ejecuta desde el tarball tpdroid-macos.tar.gz
# Uso: ./install-macos.sh

set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-$HOME/Library/Application Support/tpdroid}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"

echo "=== Instalador TPDroid para macOS ==="
echo ""

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Detectar arquitectura y buscar el binario correcto
ARCH="$(uname -m)"
if [ -f "$SCRIPT_DIR/tpdroid-macos" ]; then
  TPDROID_BIN="tpdroid-macos"
  ACTIVATOR_BIN="activator-macos"
elif [ "$ARCH" = "arm64" ] && [ -f "$SCRIPT_DIR/tpdroid-macos-arm64" ]; then
  TPDROID_BIN="tpdroid-macos-arm64"
  ACTIVATOR_BIN="activator-macos-arm64"
else
  TPDROID_BIN="tpdroid-macos-amd64"
  ACTIVATOR_BIN="activator-macos-amd64"
fi

# Verificar que los binarios existen
for f in "$TPDROID_BIN" "$ACTIVATOR_BIN" adb-macos; do
  if [ ! -f "$SCRIPT_DIR/$f" ]; then
    echo "Error: falta $f en el directorio de instalación"
    exit 1
  fi
done

# ── Activación ───────────────────────────────────────────
echo "Paso 1: Activación de licencia"
echo "Se abrirá el navegador para ingresar el código de licencia."
chmod +x "$SCRIPT_DIR/$ACTIVATOR_BIN"
"$SCRIPT_DIR/$ACTIVATOR_BIN"
ACTIVATOR_EXIT=$?

if [ $ACTIVATOR_EXIT -ne 0 ]; then
  echo ""
  echo "Error: Activación fallida. La instalación no puede continuar."
  exit 1
fi

echo ""
echo "Paso 2: Instalando archivos..."

# Crear directorios
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/adb-binaries/macos"

# Solicitar permisos para /usr/local/bin si es necesario
if [ ! -w "$BIN_DIR" ]; then
  echo "Se necesitan permisos de administrador para instalar el comando 'tpdroid'."
  sudo mkdir -p "$BIN_DIR"
fi

# Copiar binarios
cp "$SCRIPT_DIR/$TPDROID_BIN" "$INSTALL_DIR/tpdroid"
cp "$SCRIPT_DIR/adb-macos" "$INSTALL_DIR/adb-binaries/macos/adb"
chmod +x "$INSTALL_DIR/tpdroid"
chmod +x "$INSTALL_DIR/adb-binaries/macos/adb"

# Quitar cuarentena de macOS (Gatekeeper)
echo "Paso 3: Quitando restricción de Gatekeeper..."
xattr -dr com.apple.quarantine "$INSTALL_DIR/tpdroid" 2>/dev/null || true
xattr -dr com.apple.quarantine "$INSTALL_DIR/adb-binaries/macos/adb" 2>/dev/null || true

# Crear script de inicio
cat > "$INSTALL_DIR/tpdroid.sh" << 'SCRIPT'
#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"
export ADB_PATH="$DIR/adb-binaries/macos/adb"
exec "$DIR/tpdroid"
SCRIPT
chmod +x "$INSTALL_DIR/tpdroid.sh"

# Enlace simbólico en /usr/local/bin
if [ -w "$BIN_DIR" ]; then
  ln -sf "$INSTALL_DIR/tpdroid.sh" "$BIN_DIR/tpdroid"
else
  sudo ln -sf "$INSTALL_DIR/tpdroid.sh" "$BIN_DIR/tpdroid"
fi

echo ""
echo "=== Instalación completada ==="
echo ""
echo "Para ejecutar TPDroid:"
echo "  $ tpdroid"
echo "  Abrir http://localhost:8080 en el navegador"
echo ""
echo "Nota: la primera vez macOS puede pedir confirmación de seguridad."
echo "Si aparece una advertencia, ir a: Preferencias → Seguridad y privacidad → Permitir."
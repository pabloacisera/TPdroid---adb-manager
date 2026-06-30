#!/bin/bash
# install.sh — Instalador de TPDroid para Linux
# Se ejecuta desde el tarball tpdroid-linux.tar.gz
# Uso: ./install.sh

set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/share/tpdroid}"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
DESKTOP_DIR="${DESKTOP_DIR:-$HOME/.local/share/applications}"

echo "=== Instalador TPDroid ==="
echo ""

# Obtener directorio donde está este script
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Verificar que los binarios existen
for f in tpdroid activator-linux adb tpdroid-icon.png; do
  if [ ! -f "$SCRIPT_DIR/$f" ]; then
    echo "Error: falta $f en el directorio de instalación"
    exit 1
  fi
done

# ── Activación ───────────────────────────────────────────
echo "Paso 1: Activación de licencia"
echo "Se abrirá el navegador para ingresar el código de licencia."
chmod +x "$SCRIPT_DIR/activator-linux"
"$SCRIPT_DIR/activator-linux"
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
mkdir -p "$INSTALL_DIR/adb-binaries/linux"
mkdir -p "$BIN_DIR"
mkdir -p "$DESKTOP_DIR"

# Copiar binarios
cp "$SCRIPT_DIR/tpdroid" "$INSTALL_DIR/"
cp "$SCRIPT_DIR/adb" "$INSTALL_DIR/adb-binaries/linux/"
cp "$SCRIPT_DIR/tpdroid-icon.png" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/tpdroid"
chmod +x "$INSTALL_DIR/adb-binaries/linux/adb"

# Crear script de inicio
cat > "$INSTALL_DIR/tpdroid.sh" << 'SCRIPT'
#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"
export ADB_PATH="$DIR/adb-binaries/linux/adb"
"$DIR/tpdroid" &
sleep 2
xdg-open "http://localhost:8080" >/dev/null 2>&1
wait
SCRIPT
chmod +x "$INSTALL_DIR/tpdroid.sh"

# Enlace simbólico
ln -sf "$INSTALL_DIR/tpdroid.sh" "$BIN_DIR/tpdroid"

# Crear entrada de escritorio
cat > "$DESKTOP_DIR/tpdroid.desktop" << DESKTOP
[Desktop Entry]
Name=TPDroid
Comment=Android Process & Notification Manager
Exec=$INSTALL_DIR/tpdroid.sh
Icon=$INSTALL_DIR/tpdroid-icon.png
Terminal=false
Type=Application
Categories=Utility;
DESKTOP

echo ""
echo "=== Instalación completada ==="
echo ""
echo "Para ejecutar TPDroid:"
echo "  $ tpdroid"
echo "  Abrir http://localhost:8080 en el navegador"
echo ""
echo "O desde el menú de aplicaciones: TPDroid"

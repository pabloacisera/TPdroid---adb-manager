#!/bin/bash

# Nos aseguramos de estar en la raíz del proyecto
SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$SCRIPT_DIR"

# Excluimos usando rutas relativas desde la raíz
zip -r app.zip . \
  -x "api_admin/venv/*" \
  -x "*/__pycache__/*" \
  -x ".git/*"

#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="/tmp/android-manager-server"

cleanup() {
  echo ""
  echo "Deteniendo servidor..."
  if [ -n "$BACKEND_PID" ]; then
    kill $BACKEND_PID 2>/dev/null
    wait $BACKEND_PID 2>/dev/null
  fi
  rm -f "$BINARY"
  echo "Puerto 8080 liberado."
}

trap cleanup EXIT INT TERM

if lsof -i :8080 -sTCP:LISTEN &>/dev/null 2>&1; then
  echo "Error: el puerto 8080 ya está en uso."
  echo "Ejecutá: kill \$(lsof -ti :8080 -sTCP:LISTEN)"
  exit 1
fi

echo "Compilando..."
cd "$SCRIPT_DIR/backend" && go build -o "$BINARY" .
if [ $? -ne 0 ]; then
  echo "Error: compilación fallida"
  exit 1
fi

echo "Iniciando servidor en http://localhost:8080..."
"$BINARY" &
BACKEND_PID=$!

for i in $(seq 1 15); do
  if curl -s http://localhost:8080/api/status >/dev/null 2>&1; then
    echo "Servidor listo después de ${i}s"
    break
  fi
  if ! kill -0 $BACKEND_PID 2>/dev/null; then
    echo "Error: el servidor falló al iniciar"
    exit 1
  fi
  sleep 1
done

if command -v xdg-open &>/dev/null; then
  xdg-open "http://localhost:8080"
elif command -v open &>/dev/null; then
  open "http://localhost:8080"
else
  echo "Abrí http://localhost:8080 en tu navegador"
fi

echo "Servidor corriendo (PID: $BACKEND_PID). Presiona Ctrl+C para detener."
wait $BACKEND_PID

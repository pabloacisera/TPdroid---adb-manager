#!/bin/sh

# Iniciar tailscaled
tailscaled --state=/var/lib/tailscale/tailscaled.state --socket=/var/run/tailscale/tailscaled.sock &

echo "Esperando a que tailscaled inicie..."
for i in $(seq 1 10); do
    if [ -S /var/run/tailscale/tailscaled.sock ]; then
        echo "tailscaled listo"
        break
    fi
    sleep 1
done

if [ -n "$TAILSCALE_AUTHKEY" ]; then
    echo "Autenticando con Tailscale..."
    tailscale up --reset \
                 --authkey="$TAILSCALE_AUTHKEY" \
                 --hostname="${TAILSCALE_HOSTNAME:-TPDROID-ADMIN}" \
                 --accept-dns=false \
                 --accept-routes \
                 --timeout 15s || echo "tailscale up no completó (pendiente de aprobación)"
    echo "IP Tailscale: $(tailscale ip -4 2>/dev/null || echo 'no asignada')"
else
    echo "No se encontró TAILSCALE_AUTHKEY"
fi

# Ejecutar la aplicación
exec uvicorn main:app --host 0.0.0.0 --port 8000

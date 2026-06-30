#!/bin/sh

# Detener y eliminar (NO borrar el volumen de tailscale, o perderás la auth)
docker stop tpdroid-admin 2>/dev/null || true
docker rm tpdroid-admin 2>/dev/null || true

# Verificar .env existe
ls -la .env

# Stage 1 - construimos la imagen
# -- t: referencia un nombre "tpdroid-admin"
# -- "." le dice donde esta el Dockerfile
docker build -t tpdroid-admin .

# Stage 2 - Levanta el contenedor a partir de la imagen
# -- le damos permiso al contendor para que crear un red de tailscale
# -- le damos acceso al tunel virtual que usa la vpn de tailscale
# Cargar variables de entorno
# Volumenes persistentes de tailscale, main.py y static/
# Obtenemos la ruta absoluta de donde está el script
DIR="$(pwd)"
docker run -d \
    --name tpdroid-admin \
    --cap-add=NET_ADMIN \
    --device=/dev/net/tun \
    --env-file "$DIR/.env" \
    -v tailscale-state:/var/lib/tailscale \
    -v "$DIR/main.py:/app/main.py" \
    -v "$DIR/static:/app/static" \
    tpdroid-admin

# Ver logs en tiempo real
docker logs -f tpdroid-admin

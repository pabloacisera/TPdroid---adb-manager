FROM python:3.12-slim

# Actualizamos paquetes
# Instalamos curl
# Instalar cliente de tailscale
# Limpia paquetes que ya no se usarán
# Limpiar otros paquetes
# Borrar listas apt
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates \
    && curl -fsSL https://tailscale.com/install.sh | sh \
    && apt-get purge -y curl \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

# Nos paramos en app
WORKDIR /app

# Copiar requirements
COPY requirements.txt .

# Instalar dependencias sin guardar cache
RUN pip install --no-cache-dir -r requirements.txt

# Copiamos todo el proyecto dentro de app
COPY . .

# Copiamos el entry-point que arranca tailscale y uvicorn
COPY entry-point.sh /entry-point.sh

# Otorgamos permiso de ejecucion al script
RUN chmod +x entry-point.sh

# Abrir puerto 8000
EXPOSE 8000

# Arrancar el proceso
CMD ["/entry-point.sh"]

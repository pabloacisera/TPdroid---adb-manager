# Activator — Binario de Activación (Win/Linux)

El activator es un ejecutable cross-platform que maneja el formulario de activación de licencia. Se ejecuta durante la instalación y al iniciar el backend.

## Cómo funciona

1. Inicia un servidor HTTP en `127.0.0.1` (puerto aleatorio)
2. Abre el navegador predeterminado con el formulario de activación
3. El usuario ingresa código de licencia (y opcionalmente email)
4. Toma el fingerprint del hardware (SHA-256 de machine-id / MAC)
5. Envía `POST /activar` al Worker de Cloudflare
6. Si ok: guarda archivo `.lic` firmado y retorna exit code 0
7. Si error: muestra mensaje, exit code ≠ 0
8. Timeout automático a los 5 minutos

## Fingerprint cross-platform

| SO | Fuente |
|----|--------|
| Windows | `wmic csproduct get uuid` |
| Linux | `/etc/machine-id` (fallback: `hostname` + MAC) |
| Fallback global | MAC address de la primera interfaz no-loopback |

## Archivo .lic

Se guarda en:

| SO | Ruta |
|----|------|
| Windows | `%APPDATA%/TPDroid/tpdroid.lic` |
| Linux | `$XDG_CONFIG_HOME/tpdroid/license.lic` (~/.config/tpdroid/) |

## Build

```bash
# Windows (desde Linux/macOS con cross-compile)
GOOS=windows GOARCH=amd64 go build -o activator.exe ./activator/

# Linux
GOOS=linux GOARCH=amd64 go build -o activator ./activator/
```

## Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `LICENSE_WORKER_URL` | `http://localhost:8787` | URL del Worker de licencias |
| `ACTIVATOR_PORT` | `0` (aleatorio) | Puerto fijo para el servidor HTTP |

## Integración con instalador

- **NSIS (Windows)**: `tpdroid.nsi` ejecuta `activator.exe`, aborta instalación si exit code ≠ 0
- **install.sh (Linux)**: ejecuta `./activator`, aborta si falla
- **Makefile**: targets `build-activator-windows` y `build-activator-linux` con ldflags para inyectar `LICENSE_WORKER_URL`

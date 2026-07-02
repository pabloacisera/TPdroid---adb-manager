# TPdroid — Android Process & Notification Manager

[![GitHub](https://img.shields.io/badge/GitHub-pabloacisera/TPdroid--adb--manager-blue?logo=github)](https://github.com/pabloacisera/TPdroid---adb-manager.git)

Aplicación de escritorio web que se conecta a dispositivos Android vía USB (ADB embebido) para gestionar procesos en ejecución y apps instaladas. Permite forzar detención de procesos, deshabilitar/activar notificaciones, y aplicar bloqueo profundo en juegos.

El frontend está embebido en el binario Go via `go:embed`. Un solo binario, cero dependencias externas.

## Features

- **Ver procesos en ejecución** — PID, nombre, UID, estado (R/S/I/Z/T/D)
- **Forzar detención** de procesos de usuario con verificación post-stop
- **Gestión de notificaciones** — deshabilitar/activar notificaciones por app
- **Detección de juegos** — identifica juegos por package name, engines SDKs y permisos
- **Triple-Lock en juegos** (Android 13+) — `appops deny` + `pm revoke` + flags de permiso
- **Filtro de solo procesos activos** (Running)
- **Búsqueda en tiempo real** con debounce en procesos y apps
- **Paginación client-side** — 25 procesos / 20 apps por página
- **Heartbeat cada 2s** con detección de desconexión y banner con countdown
- **Page Visibility API** — pausa polling cuando la pestaña no está visible
- **Bilingüe** — Español e Inglés con toggle en la UI
- **Notificación de actualizaciones** — campanita en la navbar que consulta versión remota cada 30 min
- **Definiciones remotas** — listas de detección de juegos actualizables desde Worker sin recompilar
- **Landing dinámica** — enlaces de descarga resueltos contra versión remota
- **Protección server-side** de procesos del sistema (HTTP 403)

## Cómo usar la app

### 1. Preparar el teléfono

1. **Ajustes → Acerca del teléfono** → tocar **Número de compilación** 7 veces
2. **Ajustes → Opciones de desarrollador** → activar **Depuración USB**
3. Mantener el teléfono desbloqueado

### 2. Iniciar TPdroid
```bash
git clone https://github.com/pabloacisera/TPdroid---adb-manager.git && cd TPdroid---adb-manager
./dev.sh
# Se abre http://localhost:8080
```

### 2.1 Usando CLI

Instalación desde el paquete distribuible

Si descargaste el tarball tpdroid-linux.tar.gz o tpdroid-macos.tar.gz:

Linux
```
# 1. Descomprimir
tar -xzvf tpdroid-linux.tar.gz
# 2. Ejecutar el instalador
cd tpdroid-linux
./install.sh
# 3. Ejecutar la aplicación
tpdroid
# Abrir http://localhost:8080 en el navegador
```

macOS
```
# 1. Descomprimir
tar -xzvf tpdroid-macos.tar.gz
# 2. Ejecutar el instalador
cd tpdroid-macos
./install-macos.sh
# 3. Ejecutar la aplicación
tpdroid
# Abrir http://localhost:8080 en el navegador
```

Desinstalación
```
# Linux
rm -rf ~/.local/share/tpdroid ~/.local/bin/tpdroid ~/.local/share/applications/tpdroid.desktop

# macOS
sudo rm -f /usr/local/bin/tpdroid
rm -rf ~/Library/Application\ Support/tpdroid
```

Notas:

    El instalador valida la licencia antes de continuar.

    En Linux, el acceso directo aparecerá en el menú de aplicaciones.

    En macOS, puede aparecer una advertencia de seguridad. Ir a Preferencias → Seguridad y privacidad → Permitir.

### 3. Conectar

- Conectar el teléfono por USB
- La app detecta el dispositivo automáticamente (polling cada 2s, timeout 60s)

### 4. Autorizar

- Revisar la pantalla del teléfono y pulsar **"Permitir depuración USB"**
- Marcar **"Recordar siempre"** para no repetirlo

### 5. Dashboard

Una vez conectado, verás dos pestañas:

#### Processes
- Lista de procesos con PID, nombre, UID, estado
- Usar **"Running only"** para filtrar solo activos
- Buscar por nombre en tiempo real
- Botón **Force Stop** para detener procesos de usuario
- La app verifica que el proceso realmente se detuvo post-comando

#### Apps & Notifications
- Lista de apps de usuario con paquete, etiqueta y permisos
- Buscar por nombre/paquete en tiempo real
- Botón para **Desactivar/Activar notificaciones**
- En juegos: se muestra badge 🎮 y se aplica **Triple-Lock** profundo
- Permisos truncados (max 2) con botón **"ver"** para listar todos en modal

## Game Detection System

TPdroid detecta juegos automáticamente usando tres estrategias en orden:

### 1. Package name analysis
Busca segmentos como `.game`, `.games`, `.gaming`, `game.`, `games.` y sufijos `game`/`games`.

### 2. Game engines & Ad SDKs
Detecta packages que contengan: `unity`, `unreal`, `cocos`, `ironsource`, `applovin`, `vungle`, `admob`, `chartboost`, `tapjoy`.

### 3. Permission combo
Si una app tiene `VIBRATE` + `INTERNET`, se considera juego (alta probabilidad de serlo).

### Exclusion list (previene falsos positivos)

Muchas apps no-juego tienen `VIBRATE+INTERNET` (WhatsApp, Facebook, Chrome, Gmail, etc.). La **exclusion list** contiene ~400 prefijos de packages conocidos no-juego. Si un package coincide, se descarta como juego inmediatamente, antes de cualquier otra verificación.

Categorías cubiertas:
| Categoría | Ejemplos |
|-----------|----------|
| Social & Communication | WhatsApp, Facebook, Instagram, Twitter, LinkedIn, Snapchat, TikTok, Telegram, Discord, Slack, Reddit, Signal |
| Browsers | Chrome, Firefox, Edge, Opera, Brave, Samsung Internet, DuckDuckGo |
| Google Apps | Gmail, Maps, Drive, Photos, Calendar, Keep, YouTube, Play Store, Play Services, Assistant, News, Translate |
| Microsoft | Office (Word/Excel/PPT), Outlook, OneNote, OneDrive, Teams, Bing |
| Adobe | Acrobat Reader, Photoshop Express, Lightroom, Adobe Scan |
| Streaming | Spotify, Netflix, Prime Video, Disney+, HBO Max, Hulu, Peacock, Paramount+, Apple Music, Deezer, SoundCloud |
| Banking & Finance | PayPal, Venmo, Cash App, Chase, Bank of America, Wells Fargo, Citi, USAA, Coinbase, Binance |
| Shopping | Amazon, eBay, Walmart, Etsy, AliExpress, Shopee, MercadoLibre, Uber, Lyft |
| Food Delivery | Uber Eats, DoorDash, Grubhub, Postmates, Deliveroo |
| Productivity | Evernote, Trello, Asana, Todoist, LastPass, 1Password, Bitwarden, Dropbox |
| Health & Fitness | Google Fit, Samsung Health, MyFitnessPal, Strava, Fitbit, Nike Run Club, Headspace, Calm |
| Maps & Travel | Google Maps, Waze, Airbnb, Booking.com, Expedia, TripAdvisor, Kayak, Skyscanner |
| Utilities | Files by Google, MX Player, VLC, Speedtest, Wikipedia, IMDb |
| VPN & Remote | Cloudflare 1.1.1.1, NordVPN, ExpressVPN, ProtonVPN, WireGuard, TeamViewer, AnyDesk |
| Amazon Apps | Shopping, Prime Video, Music, Kindle, Alexa, Photos |
| Samsung | One UI Home, Health, Pay, Members, Notes, Internet, Calendar, Contacts, Messages |
| Xiaomi (MIUI) | Security Center, Cleaner, Notes, Gallery, Player, Browser, Store |
| Huawei / OnePlus / Oppo / Vivo / Realme | System apps de cada fabricante |
| Other OEMs | Sony, LG, Motorola, HTC, Nokia, Lenovo, Asus, Acer, ZTE, TCL |

### Triple-Lock para juegos (Android 13+)

Cuando se deshabilitan notificaciones de un juego, se aplican 4 comandos:

1. `appops set POST_NOTIFICATION deny` — bloquea runtime
2. `pm revoke POST_NOTIFICATIONS` — revoca el permiso
3. `pm set-permission-flags user-set` — marca como decisión de usuario
4. `pm clear-permission-flags user-fixed` — permite re-habilitar después

Al re-habilitar:
1. `appops set POST_NOTIFICATION allow` — desbloquea runtime
2. `pm grant POST_NOTIFICATIONS` — concede el permiso
3. `pm set-permission-flags user-set`
4. `pm clear-permission-flags user-fixed`

Los errores de `pm` se silencian (Android 12- no soporta estos flags). La UI muestra un toast confirmando el bloqueo profundo.

## Prerrequisitos

- Go 1.21+
- Dispositivo Android con Depuración USB habilitada
- Cable USB
- No requiere ADB instalado por separado

## Cómo Ejecutar en Modo Dev

```bash
./script/dev.sh
```

O manualmente:

```bash
cd backend && go run main.go
# Abrir http://localhost:8080
```

## Build por Plataforma

```bash
make build-linux              # → dist/tpdroid-linux.tar.gz (VERSION=dev)
make build-linux VERSION=0.2.0  # → con versión específica
make build-windows            # → dist/TPDroid-Setup.exe
make build-macos              # → dist/tpdroid-macos.tar.gz
make build-all VERSION=0.2.0  # → todas las plataformas
```

## Release Pipeline (CI/CD)

Ver [`docs/release-process.md`](docs/release-process.md) para la guía completa con todos los pasos.

Las releases se generan **solo cuando el desarrollador decide versionar**, no en cada push.

### Resumen rápido

```bash
# 1. Taggear y pushear
git tag vX.Y.Z
git push origin vX.Y.Z

# 2. Publicar el draft en GitHub → Releases → Drafts

# 3. Actualizar Worker
cd cloudflare/
wrangler secret put LATEST_VERSION "X.Y.Z"
wrangler secret put DOWNLOAD_URL "https://github.com/pabloacisera/TPdroid---adb-manager/releases/download/vX.Y.Z"
wrangler secret put CHANGELOG "..."
wrangler secret put NOTES_ES "..."
wrangler secret put NOTES_EN "..."
```

## ADB Binaries

Descargar desde [Android SDK Platform Tools](https://developer.android.com/studio/releases/platform-tools) y colocar en:

- `adb-binaries/windows/adb.exe`
- `adb-binaries/linux/adb`
- `adb-binaries/macos/adb`

## Protección de Procesos del Sistema

Un proceso se considera del sistema si:
- Su UID es `system`, `root` o `shell`, O
- Su package empieza con `com.android.`, `android.` o es `android`

Los botones de acción aparecen deshabilitados en la UI y el backend retorna HTTP 403. La protección se aplica siempre server-side (defense in depth).

## Estructura del Proyecto

```
├── backend/
│   ├── adb/
│   │   ├── client.go      # Wrappers de comandos ADB + game detection
│   │   ├── definitions.go # Remote definitions fetch + fallback local
│   │   └── types.go       # Structs de datos
│   ├── cors/
│   │   └── cors.go        # Custom CORS middleware
│   ├── version/
│   │   └── client.go      # Version cache con background poll
│   ├── handlers/
│   │   ├── handler.go     # Handler struct, ResolveSerial
│   │   ├── status.go      # GET /api/status
│   │   ├── device.go      # GET /api/device
│   │   ├── processes.go   # GET /api/processes, POST force-stop
│   │   ├── apps.go        # GET /api/apps, POST disable/enable-notification
│   │   ├── version.go     # GET /api/version
│   │   └── definitions.go # GET /api/definitions
│   ├── ui/                # Frontend embebido (go:embed)
│   │   ├── index.html     # UI con Tailwind CDN
│   │   ├── css/styles.css # Estilos custom (sin @apply)
│   │   └── js/
│   │       ├── app.js     # Lógica principal, steps, dashboard
│   │       ├── api.js     # Cliente HTTP /api
│   │       ├── ui.js      # Renderizado de tablas, modales, toast
│   │       └── i18n.js    # Traducciones EN/ES
│   └── main.go            # Entry point, rutas Gin, go:embed
├── adb-binaries/          # ADB por plataforma (no incluido en repo)
├── dist/                  # Binarios compilados
├── Makefile
├── dev.sh
└── README.md
```

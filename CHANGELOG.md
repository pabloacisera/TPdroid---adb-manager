# Changelog

## v0.2.0 (2026-06-30)

### Features

- **Update Bell**: Notification bell in the navbar that checks for new versions against a Cloudflare Worker. Shows current version, latest version, changelog, and download link. Polls every 30 minutes in background. Never blocks startup.
- **Remote Definitions**: Game detection lists (known non-game prefixes, ad keywords, game segments/engines) are now fetched from the Worker at startup with automatic fallback to hardcoded local lists. Thread-safe via `atomic.Value`.
- **Version API**: New `GET /api/version` endpoint exposing current and latest version info consumed by the frontend.
- **Definitions API**: New `GET /api/definitions` endpoint exposing remote definitions for external consumers.
- **Landing Dynamic Downloads**: Download links on the landing page are now dynamically resolved via the Worker's `/version` endpoint, falling back to hardcoded v0.1.1 links.
- **LDFLAGS Injection**: Version string is injected at build time via `-ldflags=-X main.Version=$(VERSION)`. Default `"dev"` when not specified.

### Security

- **CORS Middleware Replaced**: Removed `gin-contrib/cors` dependency. Created a custom CORS middleware (`backend/cors/cors.go`) with a restricted whitelist (`http://localhost:8080`, `http://127.0.0.1:8080`). `Origin: null` is explicitly rejected to prevent CORS bypass attacks from sandboxed iframes. `file://` removed as it was a remnant from the pre-go:embed architecture.
- **Session Token (defense in depth)**: All POST endpoints now require a `X-Session-Token` header generated at server startup via `crypto/rand` and fetched by the frontend via `GET /api/session-token`. Provides protection against DNS rebinding attacks even if CORS were bypassed. Comparison uses `crypto/subtle.ConstantTimeCompare`.
- **test-blocking.sh**: Updated to fetch session token and include `X-Session-Token` header in all 10 POST curl calls, preventing 403 errors from the new middleware.

### Features

- **Landing page repositioned**: Hero badges changed from technical stack (`Go/Gin`, `Android ADB`, etc.) to benefit-oriented messaging (`Sin Root`, `Bloqueo de Ads`, `Multi-dispositivo`, `Plug & Play USB`). Subtitle rewritten to speak to non-technical users and B2B audiences.
- **PowerShell fingerprint fallback**: Windows hardware fingerprint now falls back to PowerShell (`Get-CimInstance Win32_ComputerSystemProduct`) if `wmic` fails, before falling back to MAC address. Ensures compatibility with Windows 11 24H2+ where `wmic` is deprecated.

### Worker Changes

- Worker now accepts GET requests for `/version` and `/definitions` endpoints in addition to existing POST endpoints.
- CORS preflight now allows both GET and POST methods.
- Rate limiting applies only to POST endpoints.
- `DOWNLOAD_URL` env var eliminada — `download_url` ahora está hardcodeado en el Worker apuntando a `github.com/.../releases/latest`. La URL de descarga no depende del número de versión ni necesita configurarse post-release.

### Build System

- `Makefile`: Added `VERSION` variable (default `dev`) and `LDFLAGS` for all `build-*` targets.
- `script/build-installer.sh`: Passes `VERSION` via ldflags.
- `script/build-linux.sh`: Passes `VERSION` via ldflags.
- `script/build-macos.sh`: Passes `VERSION` via ldflags.
- `.github/workflows/release.yml`: Extracts version from git tag and sets `VERSION` env.

### Documentation

- `CHANGELOG.md` — new
- `docs/update-notifications.md` — new
- `docs/remote-definitions.md` — new
- `docs/roadmap.md` — new

## v0.1.1 (2026-06-29)

### Features
- Email durante activación (campo opcional en formulario)
- Username persistente con localStorage y modal
- Filtro de estado de notificaciones en Apps tab

### Docs
- docs/game-detection.md
- docs/admin-panel.md
- docs/activator.md
- docs/feature-ads-scanner.md

## v0.1.0 (2026-06-26)

### Features
- Release inicial con licencias DRM (Cloudflare Worker + Supabase)
- Activador cross-platform (Win + Linux)
- Panel admin FastAPI
- Backend licensing middleware
- Build + distribución (Makefile, NSIS, tarballs)
- Tests HMAC, Go y Python

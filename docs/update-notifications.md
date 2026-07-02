# Feature: Update Notifications (Campanita)

## El problema

Los usuarios no saben cuándo hay una nueva versión disponible. Tienen que revisar manualmente el repositorio o esperar a que alguien les avise.

## Solución

Una campanita en la navbar (entre el username y el toggle de idioma) que consulta periódicamente el Worker de Cloudflare para detectar versiones nuevas. Si hay update, muestra un dot rojo y un popover con la info.

## Arquitectura

```
┌──────────────┐   GET /version    ┌──────────────┐
│   Frontend   │ ───────────────→  │   Worker     │
│  (campanita) │ ←───────────────  │  cloudflare/  │
└──────────────┘   { latest,       └──────────────┘
                    download_url,
                    changelog }
       │
       │ GET /api/version
       ▼
┌──────────────┐
│   Backend    │
│ (Go/Gin)     │
│ version.Cache│
└──────────────┘
```

### Componentes

1. **Worker endpoint** (`GET /version`): devuelve `{ latest, download_url, changelog, notes_es, notes_en }`. `download_url` está hardcodeado apuntando a `github.com/.../releases/latest`. `latest` y las notas se configuran con env vars.

2. **Backend cache** (`backend/version/client.go`):
   - `Cache` struct con `sync.RWMutex` para acceso thread-safe
   - `NewCache(workerURL, currentVersion)` crea el cache y arranca background checker
   - `refresh()` fetchea el Worker cada 6 horas
   - `HasUpdate(info)` compara semver vía `CompareSemver(current, latest)`
   - Nunca bloquea el startup — la goroutine corre en background, log warning si falla

3. **Handler** (`GET /api/version`): expone `{ current, latest, download_url, changelog, notes_es, notes_en, update_available, check_failed }`.

4. **Frontend**: polling cada 30 minutos, dot rojo si `latest > current`, popover con detalles y botón de descarga. Usa `notes_es` o `notes_en` según el idioma activo, con fallback a `changelog`.

## Worker env vars

| Variable | Obligatoria | Contenido | Se muestra cuando |
|---|---|---|---|
| `LATEST_VERSION` | Sí (post-release) | Última versión (ej: `0.2.0`) | Backend compara contra `current` para decidir campana; landing reemplaza hrefs |
| `CHANGELOG` | No | Notas genéricas (un solo texto, sin idioma) | Fallback si no hay `NOTES_ES` ni `NOTES_EN` |
| `NOTES_ES` | No | Notas detalladas en español | Usuario tiene el idioma ES |
| `NOTES_EN` | No | Notas detalladas en inglés | Usuario tiene el idioma EN |

`download_url` está hardcodeado en el Worker apuntando a `github.com/.../releases/latest`. No necesita env var.

## Archivos creados/modificados

| Archivo | Cambio |
|---------|--------|
| `backend/version/client.go` | NUEVO: VersionCache, CompareSemver, HasUpdate |
| `backend/handlers/version.go` | NUEVO: GET /api/version |
| `backend/main.go` | + VersionCache init + ruta |
| `backend/ui/index.html` | + campanita en navbar + popover |
| `backend/ui/js/api.js` | + getVersionInfo() |
| `backend/ui/js/i18n.js` | + claves update.* EN/ES |
| `backend/ui/js/app.js` | + version polling 30min, toggle popover |
| `backend/ui/js/ui.js` | + renderUpdatePopover() |
| `backend/ui/css/styles.css` | + estilos update-bell, dot, popover |
| `cloudflare/worker.js` | + handleGetVersion() |
| `landing/index.html` | + dynamic version fetch para download links |

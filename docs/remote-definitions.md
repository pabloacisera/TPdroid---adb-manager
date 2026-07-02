# Feature: Remote Definitions

## El problema

Las listas de exclusión de juegos (~400 prefijos), keywords de ads, segmentos de game detection y engines conocidos están hardcodeadas en `backend/adb/client.go`. Actualizarlas requiere recompilar y redistribuir el binario.

## Solución

Las definiciones se fetchan desde el Worker de Cloudflare al arrancar y se cachean en memoria con `atomic.Value`. Si el Worker no responde o la respuesta está vacía, se usa el fallback local hardcodeado.

## Arquitectura

```
┌──────────────┐   GET /definitions  ┌──────────────┐
│   Backend    │ ────────────────→   │   Worker     │
│  (Go/Gin)    │ ←────────────────  │  cloudflare/  │
└──────┬───────┘   JSON {           └──────────────┘
       │            known_non_game_prefixes: [...],
       │            ad_keywords: [...],          ┌──────────────────┐
       │            game_segments: [...],         │  Hardcoded lists │
       │            game_engines: [...] }         │  (fallback)      │
       │                                          │  client.go       │
       └─────────────────────────────────────→   │                  │
            if Worker fails or empty              └──────────────────┘
```

### Thread safety

Usamos `sync/atomic` con `atomic.Value` para almacenar el puntero a `RemoteDefinitions`. La carga y almacenamiento son lock-free. Nunca se escribe y lee concurrentemente sin sincronización.

### Getters con fallback

Cada getter verifica si el remote tiene datos; si no, devuelve la lista hardcodeada:

- `GetKnownNonGamePrefixes()` → remote o `knownNonGamePrefixes`
- `GetAdKeywords()` → remote o `adKeywords`
- `GetGameSegments()` → remote o `gameSegments`
- `GetGameEngines()` → remote o `gameEngines`
- `GetSystemUIDs()` → remote o `nil`
- `GetSystemPrefixes()` → remote o `nil`

### Worker endpoint

`GET /definitions` devuelve un JSON plano con las 6 claves del struct `RemoteDefinitions` (`known_non_game_prefixes`, `ad_keywords`, `game_segments`, `game_engines`, `system_uids`, `system_prefixes`).

## Archivos creados/modificados

| Archivo | Cambio |
|---------|--------|
| `backend/adb/definitions.go` | NUEVO: RemoteDefinitions, atomic.Value, FetchAndApplyRemoteDefinitions, getters |
| `backend/adb/client.go` | MODIFICADO: gameSegments/gameEngines extraídos a package-level vars, funciones usan getters |
| `backend/handlers/definitions.go` | NUEVO: GET /api/definitions |
| `backend/main.go` | + FetchAndApplyRemoteDefinitions() en goroutine al startup |
| `cloudflare/worker.js` | + handleGetDefinitions() |

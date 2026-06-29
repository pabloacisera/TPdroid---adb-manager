# Feature: Ad & Intrusive Notification Scanner

## El problema que resuelve

El bloqueo de notificaciones por package (`appops set POST_NOTIFICATION deny`) no es suficiente porque:
- Una app legítima (ej: radio) puede contener un SDK de publicidad que genera notificaciones o overlays
- El navegador puede tener permisos de push web otorgados por sitios publicitarios
- SDKs de ads pueden registrar alarmas que se disparan automáticamente aunque la app esté cerrada

Esta feature detecta lo que está pasando **en tiempo real** en el dispositivo, no lo que *podría* pasar según los permisos.

## Fuentes de detección (las tres señales)

| Señal | Comando ADB | Qué detecta |
|---|---|---|
| `active_notif` | `dumpsys notification --noredact` | Notificaciones activas ahora mismo por package |
| `web_push` | `dumpsys notification --noredact` | Notificaciones push de navegador (canal "browser", "web", "push") |
| `ad_alarm` | `dumpsys alarm` | Alarmas repetitivas con tags de SDKs publicitarios |
| `overlay_window` | `dumpsys window windows` | Ventanas de tipo TYPE_APPLICATION_OVERLAY activas ahora |

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/ads/scan` | Ejecuta los 3 dumpsys en paralelo, parsea y devuelve lista unificada |
| POST | `/api/ads/block` | Revoca overlay + bloquea notificaciones + force-stop |
| POST | `/api/ads/unblock` | Restaura overlay + reactiva notificaciones |

## Qué hace exactamente el bloqueo

```
appops set <pkg> SYSTEM_ALERT_WINDOW deny
appops set <pkg> POST_NOTIFICATION deny
pm revoke <pkg> android.permission.POST_NOTIFICATIONS
am force-stop <pkg>
```

## Caso Chrome / Web Push

Si el usuario ve notificaciones de publicidad que vienen del navegador, la entrada aparece como `pkg=com.android.chrome` con razón `web_push`. Al bloquearlo, se revoca el permiso de notificaciones de Chrome completo. Si el usuario quiere ser más selectivo (solo bloquear ciertos sitios), debe ir a Settings → Chrome → Site Settings → Notifications. Esa granularidad requiere interacción humana en la UI del navegador y está fuera del alcance de ADB.

## Archivos creados/modificados

| Archivo | Tipo de cambio |
|---|---|
| `backend/adb/types.go` | + struct `AdEntry` |
| `backend/adb/client.go` | + `GetActiveNotifications`, `parseActiveNotifications`, `GetAdAlarms`, `parseAdAlarms`, `GetActiveOverlays`, `parseActiveOverlays`, `ScanAdSources`, `BlockAdSource`, `UnblockAdSource` |
| `backend/handlers/ads.go` | Archivo nuevo: handlers `GetAdScan`, `BlockAdSource`, `UnblockAdSource` |
| `backend/main.go` | + 3 rutas bajo `/api/ads/` |
| `backend/frontend/js/api.js` | + `scanAds()`, `blockAd()`, `unblockAd()` |
| `backend/frontend/js/i18n.js` | + clave `tabs.ads` y claves `ads.*` en EN y ES |
| `backend/frontend/js/ui.js` | + `renderAdTable()` |
| `backend/frontend/js/app.js` | + `renderAdTable` al import existente de ui.js; + lógica del tab ads |
| `backend/frontend/index.html` | + tab button `tab-ads` y panel `tab-ads-content` |
| `backend/frontend/css/styles.css` | + estilos `ads-badge-*`, `ads-channel`, `ads-alarm-tag`, `ads-blocked-row` |

## Limitaciones conocidas

- `dumpsys notification --noredact` requiere ADB en modo debug (que ya es requisito del proyecto).
- En Android 12+ algunas notificaciones tienen el texto redactado por política de privacidad; aun así el `pkg=` siempre está presente.
- Las alarmas de ads que no usan tags explícitos con palabras clave pueden no detectarse. El heurístico de "alarma repetitiva de app desconocida" cubre este caso parcialmente.
- El bloqueo de Chrome bloquea TODAS las notificaciones push web, no solo las publicitarias.
- Después de un force-stop, si el usuario vuelve a abrir la app, el SDK puede re-registrar alarmas. El bloqueo de notificaciones persiste, pero las alarmas en sí se registran de nuevo. Para eliminar el SDK, la única solución permanente es desinstalar la app.

## Comportamiento de bloqueo en browsers

Chrome (`com.android.chrome`) y otros browsers instalables (ej: `com.android.browser`)
son bloqueables aunque usen el prefijo `com.android.*`. La función
`IsSystemProcess()` en `backend/adb/client.go` tiene una **allowlist explícita**
que los excluye de la protección de sistema:

```
browserAllowlist = { "com.android.chrome", "com.android.browser" }
```

Esto significa que:
- **Chrome**: se puede force-stoppear, revocar overlays, y bloquear notificaciones
- **Play Store** (`com.android.vending`): permanece protegido como app de sistema
- **Settings** (`com.android.settings`): permanece protegido

La verificación de UID (`system`/`root`/`shell`) ocurre **antes** de la allowlist,
por lo que si Chrome corriera con UID `system` (no ocurre en Android standard),
seguiría protegido.

## Actualización: Bloqueo inteligente por canal (SDK-aware)

### El problema que corregía la versión anterior
El bloqueo original usaba `DisableNotifications()` que deniega POST_NOTIFICATION a nivel de package
completo. Para Chrome, esto bloqueaba también sus notificaciones legítimas.

### Solución implementada

El sistema detecta la versión del SDK del dispositivo en el momento de la conexión y adapta
la estrategia de bloqueo:

| SDK (Android) | Estrategia |
|---|---|
| ≥ 26 (Android 8+) | `cmd notification disable-channel <pkg> <channelId>` — bloquea solo el canal intrusivo |
| < 26 (Android 7 y menor) | `appops set POST_NOTIFICATION deny` — bloqueo total (única opción disponible) |

### Flujo completo

1. Al conectar el teléfono, el frontend captura `sdk_version` de `GET /api/device`
2. Al escanear, cada `AdEntry` incluye `notif_channels` con los IDs de canal activos
3. Al hacer clic en "Bloquear esta notificación":
   - Se envía al backend: `{ package, channels, sdk_version }`
   - Backend llama a `BlockAdSourceSmart()` que elige la estrategia según el SDK
   - Si el bloqueo por canal falla (canal no encontrado, etc.), cae automáticamente en bloqueo total
   - La respuesta incluye `blocked_channels` y `full_blocked` para que el frontend sepa qué pasó
4. Al desbloquear: se envían los `blocked_channels` guardados → desbloqueo simétrico

### Botón "Bloquear todo" (☢)

Disponible como segunda opción cuando el dispositivo soporta canales.
- Para browsers: muestra un `confirm()` de advertencia antes de ejecutar
- Llama a `POST /api/ads/block-full` → siempre usa bloqueo de package completo
- El estado se muestra en verde (distinto del azul del bloqueo por canal)

### Endpoints actualizados

| Endpoint | Cambio |
|---|---|
| `POST /api/ads/block` | Ahora acepta `channels` y `sdk_version`; responde con `blocked_channels` y `full_blocked` |
| `POST /api/ads/unblock` | Ahora acepta `blocked_channels` y `sdk_version`; desbloquea simétricamente |
| `POST /api/ads/block-full` | NUEVO — bloqueo total explícito, ignora canales y SDK |

# Feature: Game Notification Triple-Lock

## El problema

Los juegos en Android usan múltiples capas para generar notificaciones. Un simple `appops set POST_NOTIFICATION deny` no es suficiente porque muchos juegos registran canales de notificación a nivel de sistema que persisten incluso después del bloqueo básico.

## Triple-Lock

Para apps detectadas como juegos, se aplican 4 comandos en lugar de 1:

| Comando | Propósito |
|---------|-----------|
| `appops set <pkg> POST_NOTIFICATION deny` | Bloqueo nivel appops |
| `pm revoke <pkg> android.permission.POST_NOTIFICATIONS` | Revocar permiso runtime |
| `pm set-permission-flags <pkg> android.permission.POST_NOTIFICATIONS user-set` | Marcar como "user-set" para que el SO no lo reactive |
| `pm clear-permission-flags <pkg> android.permission.POST_NOTIFICATIONS user-fixed` | Limpiar "user-fixed" para permitir re-activación futura |

Los errores de `pm` en Android < 13 se ignoran silenciosamente.

## Detección de juegos

`IsGameApp()` en `backend/adb/client.go` usa 3 estrategias en orden:

1. **Exclusion list** (~400 prefijos conocidos como `com.google.`, `com.microsoft.`, `com.facebook.`, `com.whatsapp.`). Si el package coincide con algún prefijo, se descarta inmediatamente como no-juego.

2. **Segmentos en el package name**: busca `.game`, `.games`, `.gaming`, `game.`, `games.` en cualquier posición, o sufijo `game`/`games`.

3. **SDKs de juegos/ads conocidos**: detecta engines como `unity`, `unreal`, `cocos`, y ad-SDKs como `ironsource`, `applovin`, `vungle`, `admob`, `chartboost`, `tapjoy` en el package name.

## Frontend

- La columna Package muestra un badge `🎮 Game` (púrpura) cuando `is_game === true`
- Los botones de notificaciones incluyen `data-is-game`
- Al desactivar, el toast incluye "— deep lock applied" / "— bloqueo profundo aplicado"
- El badge usa clase `.badge-game` con color `#7c3aed`

## Archivos involucrados

| Archivo | Cambio |
|---------|--------|
| `backend/adb/client.go` | `IsGameApp()`, `isKnownNonGame()`, modificación de `DisableNotifications`/`EnableNotifications` para aceptar `isGame` |
| `backend/handlers/apps.go` | Handlers leen `is_game` del body |
| `backend/ui/js/api.js` | `disableNotification()`/`enableNotification()` envían `is_game` |
| `backend/ui/js/ui.js` | `renderAppsTable()` muestra badge + `data-is-game` |
| `backend/ui/js/app.js` | Click handlers pasan `isGame` a la API |
| `backend/ui/js/i18n.js` | Claves `badges.game`, `toast.gameLocked` |
| `backend/ui/css/styles.css` | `.badge-game` |

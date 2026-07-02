# Release Process

Guía completa para publicar una nueva versión de TPDroid.

## Resumen del pipeline

| Paso | Acción | Automático |
|---|---|---|
| 1. GitHub Release | `git tag vX.Y.Z && git push origin vX.Y.Z` | Trigger del workflow |
| 2. Build + subir artifacts | GitHub Action compila Linux/Windows/macOS y crea draft | Sí |
| 3. Publicar release en GitHub | Ir a Releases → Drafts → Publish | No (manual) |
| 4. Actualizar Worker | `wrangler secret put` con los nuevos valores | No (manual) |
| 5. Landing actualiza links | Script fetchea Worker y reemplaza hrefs | Sí |
| 6. Update bell en usuarios | Backend compara versión actual vs latest del Worker | Sí |

---

## 1. GitHub Release

```bash
git tag v0.3.0          # Elegir la versión
git push origin v0.3.0  # → Dispara GitHub Actions
```

El workflow [`release.yml`](../.github/workflows/release.yml):
- Compila backend + activator para Linux, Windows y macOS
- Empaqueta `tpdroid-linux.tar.gz`, `TPDroid-Setup.exe`, `tpdroid-macos-*.tar.gz`
- Crea un **draft release** en GitHub con los artifacts

La `LICENSE_WORKER_URL` se toma del secret del repositorio.

## 2. Publicar la release

1. Ir a **GitHub → Releases → Drafts**
2. Revisar los artifacts, editar descripción
3. Click en **"Publish release"**

## 3. Actualizar el Worker

Después de publicar la release, actualizar `LATEST_VERSION` para que la landing y el update bell apunten a la nueva versión.

```bash
cd cloudflare/

wrangler secret put LATEST_VERSION "0.3.0"
```

`LATEST_VERSION` es la única env var necesaria. Se usa para:
- **Landing**: reemplaza el número de versión en los hrefs de descarga
- **Update bell**: el backend compara `latest` vs `current` para decidir si mostrar la campana

La URL de descarga (`download_url`) está hardcodeada en el Worker y siempre apunta a `github.com/.../releases/latest`. No necesita configurarse.

### Opcional: notas de versión en el popover

Si querés que los usuarios vean las notas al hacer click en la campanita, podés setear:

```bash
wrangler secret put CHANGELOG  "• Correcciones de seguridad\n• Nuevo sistema de fingerprint\n• Mejoras en CORS"
wrangler secret put NOTES_ES   "• Se corrigió error de autenticación\n• Nueva campanita de actualización\n• Mejoras en detección de juegos"
wrangler secret put NOTES_EN   "• Fixed authentication error\n• New update notification bell\n• Improved game detection"
```

| Variable | Contenido | Cuándo se muestra |
|---|---|---|
| `CHANGELOG` | Notas genéricas (un solo texto, sin distinción de idioma) | Fallback si no hay `NOTES_ES` ni `NOTES_EN` |
| `NOTES_ES` | Notas detalladas en español | Usuario tiene el idioma ES |
| `NOTES_EN` | Notas detalladas en inglés | Usuario tiene el idioma EN |

Si no se setean, el popover solo muestra la versión y el botón de descarga — la campana igual funciona.

## 4. Landing

La landing (`landing/index.html`) tiene un script inline que fetchea `GET /version` del Worker y reemplaza los hrefs de descarga automáticamente. No requiere deploy adicional.

La landing se sirve desde Cloudflare Pages y se deploya automáticamente en cada push a `main`.

## 5. Update bell (campanita de actualización)

El backend ejecuta un `version.Cache` que:
- Al arrancar, fetchea `GET /version` del Worker
- Re-consulta cada 6 horas
- Compara la versión actual (inyectada via ldflags) con `latest`

Si `latest > current`, el frontend muestra el punto rojo en la campanita. Al hacer click, el popover muestra:
- Versión actual vs latest
- Botón "Descargar" que lleva a la página de releases de GitHub
- Notas de versión en el idioma del usuario (`NOTES_ES` / `NOTES_EN` / `CHANGELOG` como fallback)

## Notas importantes

- `LATEST_VERSION` tiene default `"0.2.0"` en el Worker. Para releases siguientes **hay que actualizarlo** con `wrangler secret put`.
- `CHANGELOG`, `NOTES_ES` y `NOTES_EN` son opcionales. Si no se setean, la campana igual funciona — solo no muestra notas.
- `download_url` está hardcodeado en el Worker, no necesita env var.
- El backend se compila con `-ldflags="-X main.Version=${VERSION}"`. Si se buildéa sin setear `VERSION`, queda como `"dev"` y nunca compara semver correctamente.

# Deploy de la Landing Page TPDroid

La landing page es un sitio estático (HTML + CSS inline + JS vanilla) en el directorio `landing/`. Sin frameworks, sin build steps.

## Deploy en Render.com

### Opción 1: Static Site (recomendada)

1. Ir a [Render Dashboard → New → Static Site](https://dashboard.render.com/select-repo?type=static)
2. Conectar el repositorio
3. Configurar:

   | Campo | Valor |
   |-------|-------|
   | **Name** | `tpdroid-landing` |
   | **Root Directory** | `landing` |
   | **Build Command** | _vacío_ (no necesita build) |
   | **Publish Directory** | `.` (el root es `landing/`) |

4. Hacer clic en **Create Static Site**

Render deploya automáticamente en cada push a la rama principal.

### Opción 2: Cloudflare Pages (alternativa)

1. Ir a [Cloudflare Dashboard → Workers & Pages → Create → Pages](https://dash.cloudflare.com/)
2. Conectar el repositorio
3. Configurar:

   | Campo | Valor |
   |-------|-------|
   | **Project name** | `tpdroid-landing` |
   | **Production branch** | `main` |
   | **Build command** | _vacío_ |
   | **Build output directory** | `landing` |

4. Hacer clic en **Save and Deploy**

## Personalizar

- Los textos están en el objeto `i18n` dentro del `<script>` al final del `index.html`.
- La paleta de colores se define en las variables del `<style>` (fondo `#111827`, acento `#60A5FA`, etc.).
- Los enlaces de descarga (`href="#"`) apuntan a las releases de GitHub. Reemplazar con las URLs reales de los artifacts cuando estén publicados.

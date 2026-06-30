# Deploy de la Landing Page TPDroid

La landing page es un sitio estático (HTML + CSS inline + JS vanilla) en el directorio `landing/`. Sin frameworks, sin build steps.

## Deploy en Cloudflare Pages

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

Cloudflare Pages deploya automáticamente en cada push a `main`.

## Personalizar

- Los textos están en el objeto `i18n` dentro del `<script>` al final del `index.html`.
- La paleta de colores se define en las variables del `<style>` (fondo `#111827`, acento `#60A5FA`, etc.).
- Los enlaces de descarga (`href="#"`) apuntan a las releases de GitHub. Reemplazar con las URLs reales de los artifacts cuando estén publicados.

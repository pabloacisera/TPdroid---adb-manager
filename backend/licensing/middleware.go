package licensing

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed license-error.html tpdroid-icon.png tpdroid-icon-without-background.png
var errorPage embed.FS

func BlockMiddleware() gin.HandlerFunc {
	errorHTML, err := fs.ReadFile(errorPage, "license-error.html")
	if err != nil {
		return func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Licencia inválida",
			})
		}
	}

	return func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Licencia inválida. Adquiera una licencia para usar TPDroid.",
			})
			return
		}
		// Serve embedded icon files so the error page can display them
		if c.Request.URL.Path == "/tpdroid-icon.png" || c.Request.URL.Path == "/tpdroid-icon-without-background.png" {
			iconData, err := fs.ReadFile(errorPage, c.Request.URL.Path[1:])
			if err == nil {
				c.Data(http.StatusOK, "image/png", iconData)
				c.Abort()
				return
			}
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusForbidden, string(errorHTML))
	}
}

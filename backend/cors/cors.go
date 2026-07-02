package cors

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var allowedOrigins = []string{
	"http://localhost:8080",
	"http://127.0.0.1:8080",
}

func originAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	for _, allowed := range allowedOrigins {
		if strings.HasPrefix(origin, allowed) {
			return true
		}
	}
	return false
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if originAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

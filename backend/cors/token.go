package cors

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

var sessionToken atomic.Value

func init() {
	token := generateToken()
	sessionToken.Store(token)
}

func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func GetSessionToken() string {
	v := sessionToken.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

func SessionTokenMiddleware() gin.HandlerFunc {
	serverToken := GetSessionToken()
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		clientToken := c.GetHeader("X-Session-Token")
		if clientToken == "" || len(clientToken) != len(serverToken) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid session token"})
			return
		}
		if subtle.ConstantTimeCompare([]byte(clientToken), []byte(serverToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid session token"})
			return
		}
		c.Next()
	}
}

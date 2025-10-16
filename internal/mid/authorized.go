package mid

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hamidoujand/jumble/internal/auth"
)

func Authorized(a *auth.Auth, roles map[string]struct{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, ok := c.Get("claims")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": http.StatusText(http.StatusUnauthorized)})
			c.Abort()
			return
		}

		claim, ok := val.(auth.Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": http.StatusText(http.StatusUnauthorized)})
			c.Abort()
			return
		}

		err := a.Authorized(claim, roles)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": http.StatusText(http.StatusUnauthorized)})
			c.Abort()
			return
		}

		c.Next()
	}
}

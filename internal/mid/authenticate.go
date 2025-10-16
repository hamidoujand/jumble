package mid

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/pkg/logger"
)

func Authenticate(log logger.Logger, a *auth.Auth, usrBus *bus.Bus) gin.HandlerFunc {
	return func(c *gin.Context) {
		// using a 5 seconds ctx to hit the db
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second*5)
		defer cancel()

		token := c.Request.Header.Get("authorization")

		claims, err := a.VerifyToken(ctx, token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		if claims.Subject == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "you are not authorized to take this action"})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("invalid subject: %s", claims.Subject)})
			c.Abort()
			return
		}

		//fetch the user from db
		usr, err := usrBus.QueryByID(ctx, userID)
		if errors.Is(err, bus.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": http.StatusText(http.StatusUnauthorized)})
			c.Abort()
			return
		}

		if err != nil {
			log.Error(c.Request.Context(), "queryByID", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
			c.Abort()
			return
		}

		if !usr.Enabled {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user is disabled"})
			c.Abort()
			return
		}

		c.Set("claims", claims)
		c.Set("user", usr)

		c.Next()
	}
}

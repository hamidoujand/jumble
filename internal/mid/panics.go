package mid

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/hamidoujand/jumble/internal/metrics"
	"github.com/hamidoujand/jumble/pkg/logger"
)

func Panic(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				val, ok := c.Get("metrics")
				if ok {
					m := val.(*metrics.Metrics)
					m.AddPanic()
				}

				log.Error(c.Request.Context(), "PANIC", rec, "STACK", string(stack))
				c.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
				c.Abort()
				return
			}
		}()

		c.Next()
	}
}

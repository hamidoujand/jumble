package mid

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hamidoujand/jumble/pkg/logger"
)

func Logger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()

		//full path with queries
		p := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			p = fmt.Sprintf("%s?%s", p, c.Request.URL.RawQuery)
		}

		log.Info(c.Request.Context(), "request started", "method", c.Request.Method, "path", p, "remoteAddr", c.Request.RemoteAddr)
		c.Next()

		statusCode := c.Writer.Status()
		took := time.Since(startedAt)

		log.Info(c.Request.Context(), "request completed", "method", c.Request.Method, "path", p, "remoteAddr", c.Request.RemoteAddr, "statusCode", statusCode, "took", took)
	}
}

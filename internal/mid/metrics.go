package mid

import (
	"github.com/gin-gonic/gin"
	"github.com/hamidoujand/jumble/internal/metrics"
)

func Metrics(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqs := m.AddRequest()
		if reqs%1000 == 0 {
			m.AddGoroutine()
		}

		c.Set("metrics", m)

		c.Next()

		if c.Writer.Status() >= 400 {
			m.AddError()
		}
	}
}

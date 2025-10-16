package mid

import (
	"github.com/gin-gonic/gin"
	"github.com/hamidoujand/jumble/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func Telemetry(tracer trace.Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start span for HTTP request
		ctx, span := tracer.Start(
			c.Request.Context(),
			"http.request",
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.route", c.FullPath()),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
				attribute.String("net.host.name", c.Request.Host),
			),
		)
		defer span.End()

		//set the traceID
		if span.SpanContext().HasTraceID() {
			traceID := span.SpanContext().TraceID().String()
			ctx = telemetry.SetTraceID(ctx, traceID)
		}

		c.Request = c.Request.WithContext(ctx)

		//process the request
		c.Next()

		//get response data
		// Set response attributes
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// Record errors
		if c.Writer.Status() >= 400 {
			span.SetStatus(codes.Error, "request failed")
		}
	}
}

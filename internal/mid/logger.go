package mid

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hamidoujand/jumble/pkg/logger"
	"github.com/hamidoujand/jumble/pkg/mux"
)

func Logger(log logger.Logger) mux.Middleware {
	return func(next mux.HandlerFunc) mux.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			startedAt := mux.GetReqStartedAt(ctx)

			//full path with queries
			p := r.URL.Path
			if r.URL.RawQuery != "" {
				p = fmt.Sprintf("%s?%s", p, r.URL.RawQuery)
			}

			log.Info(ctx, "request started", "method", r.Method, "path", p, "remoteAddr", r.RemoteAddr)
			err := next(ctx, w, r)

			statusCode := mux.GetStatusCode(ctx)
			took := time.Since(startedAt)

			log.Info(ctx, "request completed", "method", r.Method, "path", p, "remoteAddr", r.RemoteAddr, "statusCode", statusCode, "took", took)
			return err
		}
	}
}

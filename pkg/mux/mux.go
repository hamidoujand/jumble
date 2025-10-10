// Package mux is going to provide a custom mux with middleware support.
package mux

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HandlerFunc represents a new custom handler that handles requests with our custom mux.
type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// Mux represents a custom http.ServeMux with modified features.
type Mux struct {
	*http.ServeMux
	log  logger.Logger
	mids []Middleware
}

// New creates a new Mux and returns it.
func New(log logger.Logger, mids ...Middleware) *Mux {
	mux := http.NewServeMux()

	return &Mux{
		ServeMux: mux,
		log:      log,
		mids:     mids,
	}
}

// HandleFunc sets a handler to an http.Method and given path.
func (m *Mux) HandleFunc(method string, version string, path string, handlerFunc HandlerFunc, mids ...Middleware) {
	//first apply the route specific mids
	wrappedHandler := registerMiddlewares(handlerFunc, mids)
	wrappedHandler = registerMiddlewares(wrappedHandler, m.mids)

	h := func(w http.ResponseWriter, r *http.Request) {
		//original context from req.
		ctx := r.Context()

		rm := RequestMeta{
			startedAt: time.Now(),
			requestID: uuid.New(),
		}

		ctx = SetReqMetadata(ctx, &rm)

		wrappedHandler(ctx, w, r)
	}

	if version != "" {
		path = "/" + version + path
	}

	pattern := fmt.Sprintf("%s %s", method, path)
	otelHandler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(h))
	m.ServeMux.Handle(pattern, otelHandler)
}

// HandlerFuncNoMid is for routes that you do not want to go through middleware chain.
// such rounts are like: readiness and liveness probes.
func (m *Mux) HandleFuncNoMid(method string, version string, path string, handlerFunc HandlerFunc) {
	h := func(w http.ResponseWriter, r *http.Request) {
		//original context from req.
		ctx := r.Context()

		rm := RequestMeta{
			startedAt: time.Now(),
			requestID: uuid.New(),
		}

		ctx = SetReqMetadata(ctx, &rm)

		handlerFunc(ctx, w, r)
	}

	if version != "" {
		path = "/" + version + path
	}

	pattern := fmt.Sprintf("%s %s", method, path)
	otelHandler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(h))
	m.ServeMux.Handle(pattern, otelHandler)
}

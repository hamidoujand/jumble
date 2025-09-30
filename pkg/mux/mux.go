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
	"go.opentelemetry.io/otel/trace"
)

// HandlerFunc represents a new custom handler that handles requests with our custom mux.
type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// Mux represents a custom http.ServeMux with modified features.
type Mux struct {
	mux    *http.ServeMux
	otlMux http.Handler
	log    logger.Logger
	mids   []Middleware
}

// New creates a new Mux and returns it.
func New(log logger.Logger, tracer trace.Tracer, mids ...Middleware) *Mux {
	mux := http.NewServeMux()

	// Wrap the standard mux with OpenTelemetry handler
	// This creates the first span for incoming HTTP requests
	/*
		Creates a span for every incoming HTTP request
		Extracts trace context from headers (if coming from another service)

		Sets span attributes like:

		HTTP method (http.method)
		URL path (http.route)
		Status code (http.status_code)
		User agent (http.user_agent)
	*/

	return &Mux{
		mux:    mux,
		log:    log,
		otlMux: otelhttp.NewHandler(mux, "request"),
		mids:   mids,
	}
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.otlMux.ServeHTTP(w, r)
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

		if err := wrappedHandler(ctx, w, r); err != nil {
			//if you have an err in here you only need to log it
			m.log.Error(ctx, "error while handling request", "err", err.Error())
			return
		}
	}

	if version != "" {
		path = "/" + version + path
	}

	pattern := fmt.Sprintf("%s %s", method, path)

	m.mux.HandleFunc(pattern, h)
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

		if err := handlerFunc(ctx, w, r); err != nil {
			//if you have an err in here you only need to log it
			m.log.Error(ctx, "error while handling request", "err", err.Error())
			return
		}
	}

	if version != "" {
		path = "/" + version + path
	}

	pattern := fmt.Sprintf("%s %s", method, path)

	m.mux.HandleFunc(pattern, h)
}

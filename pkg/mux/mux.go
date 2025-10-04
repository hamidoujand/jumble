// Package mux is going to provide a custom mux with middleware support.
package mux

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/pkg/logger"
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

		if err := wrappedHandler(ctx, w, r); err != nil {
			m.log.Error(ctx, "error while handling request", "err", err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "internal server error",
			})
			return
		}
	}

	if version != "" {
		path = "/" + version + path
	}

	pattern := fmt.Sprintf("%s %s", method, path)

	m.ServeMux.HandleFunc(pattern, h)
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
			m.log.Error(ctx, "error while handling request", "err", err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "internal server error",
			})
			return
		}
	}

	if version != "" {
		path = "/" + version + path
	}

	pattern := fmt.Sprintf("%s %s", method, path)

	m.ServeMux.HandleFunc(pattern, h)
}

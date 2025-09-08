// Package mux is going to provide a custom mux with middleware support.
package mux

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hamidoujand/jumble/pkg/logger"
)

// HandlerFunc represents a new custom handler that handles requests with our custom mux.
type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// Mux represents a custom http.ServeMux with modified features.
type Mux struct {
	//since Mux follows the "is a" rules which means "Mux is a Custom ServeMux" we use embedding.
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

	m.ServeMux.HandleFunc(pattern, h)
}

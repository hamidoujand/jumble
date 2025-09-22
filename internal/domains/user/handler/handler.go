// Package handlers provides endpoints to intract with users domain.
package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hamidoujand/jumble/pkg/mux"
)

type handler struct {
}

func (h *handler) CreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	msg := map[string]string{
		"msg":     "Hello World!",
		"traceID": mux.GetTraceID(ctx).String(),
	}

	if err := mux.Respond(ctx, w, http.StatusOK, msg); err != nil {
		return fmt.Errorf("failed to respond: %w", err)
	}

	return nil
}

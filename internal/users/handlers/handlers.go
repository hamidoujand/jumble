// Package handlers provides endpoints to intract with users domain.
package handlers

import (
	"context"
	"fmt"
	"net/http"
)

type handler struct {
}

func (h *handler) CreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	fmt.Fprintln(w, "successfull hit!")
	return nil
}

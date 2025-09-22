package handler

import (
	"net/http"

	"github.com/hamidoujand/jumble/pkg/mux"
)

// RegisterRoutes takes the mux and register endpoints on it.
func RegisterRoutes(mux *mux.Mux) {
	const version = "v1"

	usr := handler{}

	mux.HandleFunc(http.MethodGet, version, "/users", usr.CreateUser)
}

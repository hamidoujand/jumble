package handler

import (
	"net/http"

	"github.com/hamidoujand/jumble/pkg/logger"
	"github.com/hamidoujand/jumble/pkg/mux"
	"github.com/jmoiron/sqlx"
)

type Conf struct {
	Mux   *mux.Mux
	DB    *sqlx.DB
	Log   logger.Logger
	Build string
}

func RegisterRoutes(cfg Conf) *http.ServeMux {
	mux := http.NewServeMux()

	h := handler{
		db:    cfg.DB,
		log:   cfg.Log,
		build: cfg.Build,
	}

	mux.HandleFunc("/v1/readiness", h.readiness)
	mux.HandleFunc("/v1/liveness", h.liveness)
	return mux
}

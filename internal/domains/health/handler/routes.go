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

func RegisterRoutes(cfg Conf) {
	const version = "v1"

	h := handler{
		db:    cfg.DB,
		log:   cfg.Log,
		build: cfg.Build,
	}

	cfg.Mux.HandleFuncNoMid(http.MethodGet, version, "/readiness", h.readiness)
	cfg.Mux.HandleFuncNoMid(http.MethodGet, version, "/liveness", h.liveness)
}

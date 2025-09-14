package handler

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/hamidoujand/jumble/internal/errs"
	"github.com/hamidoujand/jumble/internal/sqldb"
	"github.com/hamidoujand/jumble/pkg/logger"
	"github.com/hamidoujand/jumble/pkg/mux"
	"github.com/jmoiron/sqlx"
)

type handler struct {
	db    *sqlx.DB
	log   logger.Logger
	build string
}

func (h *handler) readiness(ctx context.Context, w http.ResponseWriter, _ *http.Request) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	if err := sqldb.ConnCheck(ctx, h.db); err != nil {
		h.log.Error(ctx, "readiness failed", "err", err.Error())
		return errs.Newf(http.StatusInternalServerError, "readiness failed")
	}

	return mux.Respond(ctx, w, http.StatusOK, nil)
}

func (h *handler) liveness(ctx context.Context, w http.ResponseWriter, _ *http.Request) error {
	//host name from kernel
	host, err := os.Hostname()
	if err != nil {
		host = "unavailable"
	}

	info := Info{
		Status:     "running",
		Build:      h.build,
		Host:       host,
		Name:       os.Getenv("KUBERNETES_NAME"),
		PodIP:      os.Getenv("KUBERNETES_POD_IP"),
		Node:       os.Getenv("KUBERNETES_NODE_NAME"),
		Namespace:  os.Getenv("KUBERNETES_NAMESPACE"),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
	}

	return mux.Respond(ctx, w, http.StatusOK, info)
}

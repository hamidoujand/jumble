package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/hamidoujand/jumble/internal/sqldb"
	"github.com/hamidoujand/jumble/pkg/logger"
	"github.com/jmoiron/sqlx"
)

type handler struct {
	db    *sqlx.DB
	log   logger.Logger
	build string
}

func (h *handler) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	if err := sqldb.ConnCheck(ctx, h.db); err != nil {
		h.log.Error(ctx, "readiness failed", "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"msg": "ok"})
}

func (h *handler) liveness(w http.ResponseWriter, r *http.Request) {
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

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(info)
}

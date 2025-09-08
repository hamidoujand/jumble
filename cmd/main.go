package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/hamidoujand/jumble/internal/debug"
	userHandlers "github.com/hamidoujand/jumble/internal/users/handlers"
	"github.com/hamidoujand/jumble/pkg/logger"
	"github.com/hamidoujand/jumble/pkg/mux"
)

//TODOS: add TLS support.

var build = "development"

func main() {
	traceIDFn := func(ctx context.Context) string {
		return "00000000-0000-0000-0000-000000000000"
	}

	ctx := context.Background()

	var env logger.Environment

	if build == "development" {
		env = logger.EnvironmentDev
	} else {
		env = logger.EnvironmentProd
	}

	log := logger.New(os.Stdout, logger.LevelDebug, env, "jumble", traceIDFn)

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "main failed to execute run", "err", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, log logger.Logger) error {
	log.Info(ctx, "run", "build", build, "GOMAXPROCS", runtime.GOMAXPROCS(0))

	//configuration
	cfg := struct {
		Web struct {
			ReadTimeout    time.Duration `conf:"default:10s"`
			WriteTimeout   time.Duration `conf:"default:30s"`
			IdleTimeout    time.Duration `conf:"default:120s"`
			ShutdownTimout time.Duration `conf:"default:120s"`
			DebugHost      string        `conf:"default:0.0.0.0:3000"`
			APIHost        string        `conf:"default:0.0.0.0:8000"`
		}
	}{}

	const prefix = "JUMBLE"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			//print help
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing conf: %w", err)
	}

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("conf to string: %w", err)
	}

	log.Info(ctx, "app configuration", "cfg", out)

	//==========================================================================
	//Debug Server
	go func() {
		log.Info(ctx, "debug server starting", "host", cfg.Web.DebugHost)
		if err := http.ListenAndServe(cfg.Web.DebugHost, debug.Register()); err != nil {
			log.Error(ctx, "failed to start debug server", "host", cfg.Web.DebugHost, "err", err.Error())
			return
		}
	}()

	//==========================================================================
	// Mux init
	m := mux.New(log)

	userHandlers.RegisterRoutes(m)

	//==========================================================================
	// API Server
	server := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      m,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     logger.NewStdLogger(log, logger.LevelError),
	}

	serverErrs := make(chan error, 1)
	shutdown := make(chan os.Signal, 1)

	//os.Interrupt is going to be platform independent for example on UNIX it mapped to "syscall.SIGINT" on Windows to
	//something else, so we need a flexibility in here so we use os.Interrupt as well.
	//NOTE: you can skip "syscall.SIGINT" and only use os.Interrupt.
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info(ctx, "API server starting", "host", cfg.Web.APIHost)
		if err := server.ListenAndServe(); err != nil {
			serverErrs <- fmt.Errorf("listenAndServe: %w", err)
		}
	}()

	select {
	case err := <-serverErrs:
		//something went wrong when starting the server
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Info(ctx, "server received a shutdown signal", "signal", sig)
		defer log.Info(ctx, "server completed the shutdown process", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			server.Close()
			return fmt.Errorf("failed to gracefully shutdown the server: %w", err)
		}
	}
	return nil
}

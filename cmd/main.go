package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/hamidoujand/jumble/pkg/logger"
)

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
			ReadTimeout       time.Duration `conf:"default:10s"`
			ReadHeaderTimeout time.Duration `conf:"default:5s"`
			WriteTimeout      time.Duration `conf:"default:30s"`
			IdleTimeout       time.Duration `conf:"default:120s"`
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

	shutdown := make(chan os.Signal, 1)

	//os.Interrupt is going to be platform independent for example on UNIX it mapped to "syscall.SIGINT" on Windows to
	//something else, so we need a flexibility in here so we use os.Interrupt as well.
	//NOTE: you can skip "syscall.SIGINT" and only use os.Interrupt.
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-shutdown
	return nil
}

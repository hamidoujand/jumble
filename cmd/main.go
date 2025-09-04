package main

import (
	"context"
	"os"

	"github.com/hamidoujand/jumble/pkg/logger"
)

func main() {
	traceIDFn := func(ctx context.Context) string {
		return "00000000-0000-0000-0000-000000000000"
	}

	ctx := context.Background()

	logger := logger.New(os.Stdout, logger.LevelDebug, logger.EnvironmentDev, "jumble", traceIDFn)

	if err := run(ctx, logger); err != nil {
		logger.Error(ctx, "main failed to execute run", "err", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, log logger.Logger) error {
	log.Info(ctx, "run", "hello", "world!")
	return nil
}

package metrics

import (
	"context"
	"expvar"
	"runtime"
)

// can be accessed concurrently thanks to expvar package.
type metrics struct {
	goroutines *expvar.Int
	requests   *expvar.Int
	errors     *expvar.Int
	panics     *expvar.Int
}

var m metrics

func init() {
	m = metrics{
		goroutines: expvar.NewInt("goroutines"),
		requests:   expvar.NewInt("requests"),
		errors:     expvar.NewInt("errors"),
		panics:     expvar.NewInt("panics"),
	}
}

type key int

const metricsKey key = 1

func Set(ctx context.Context) context.Context {
	return context.WithValue(ctx, metricsKey, &m)
}

func AddGoroutine(ctx context.Context) int {
	m, ok := ctx.Value(metricsKey).(*metrics)
	if !ok {
		return 0
	}

	gs := runtime.NumGoroutine()
	m.goroutines.Add(int64(gs))
	return gs
}

func AddRequest(ctx context.Context) int {
	m, ok := ctx.Value(metricsKey).(*metrics)
	if !ok {
		return 0
	}

	m.requests.Add(1)
	return int(m.requests.Value())
}

func AddPanic(ctx context.Context) int {
	m, ok := ctx.Value(metricsKey).(*metrics)
	if !ok {
		return 0
	}
	m.panics.Add(1)
	return int(m.panics.Value())
}

func AddError(ctx context.Context) int {
	m, ok := ctx.Value(metricsKey).(*metrics)
	if !ok {
		return 0
	}

	m.errors.Add(1)
	return int(m.errors.Value())
}

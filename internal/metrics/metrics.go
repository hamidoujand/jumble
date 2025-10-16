package metrics

import (
	"expvar"
	"runtime"
)

// can be accessed concurrently thanks to expvar package.
type Metrics struct {
	goroutines *expvar.Int
	requests   *expvar.Int
	errors     *expvar.Int
	panics     *expvar.Int
}

func New() *Metrics {
	m := Metrics{
		goroutines: expvar.NewInt("goroutines"),
		requests:   expvar.NewInt("requests"),
		errors:     expvar.NewInt("errors"),
		panics:     expvar.NewInt("panics"),
	}

	return &m
}

func (m *Metrics) AddGoroutine() int {
	gs := runtime.NumGoroutine()
	m.goroutines.Add(int64(gs))
	return gs
}

func (m *Metrics) AddRequest() int {
	m.requests.Add(1)
	return int(m.requests.Value())
}

func (m *Metrics) AddPanic() int {
	m.panics.Add(1)
	return int(m.panics.Value())
}

func (m *Metrics) AddError() int {
	m.errors.Add(1)
	return int(m.errors.Value())
}

// Package metrics constructs the metrics the application will track.
package metrics

import (
	"context"
	"expvar"
)

var m *metrics

type metrics struct {
	gouroutines *expvar.Int
	requests    *expvar.Int
	errors      *expvar.Int
	panics      *expvar.Int
}

func init() {
	m = &metrics{
		gouroutines: expvar.NewInt("goroutines"),
		requests:    expvar.NewInt("requests"),
		errors:      expvar.NewInt("errors"),
		panics:      expvar.NewInt("panics"),
	}
}

type ctxKey int

const key ctxKey = 1

func Set(ctx context.Context) context.Context {
	return context.WithValue(ctx, key, m)
}

func AddGoroutines(ctx context.Context) {
	if v, ok := ctx.Value(key).(*metrics); ok {
		if v.requests.Value()%100 == 0 {
			v.gouroutines.Add(1)
		}
	}
}

func AddRequests(ctx context.Context) {
	if v, ok := ctx.Value(key).(*metrics); ok {
		v.requests.Add(1)
	}

}

func AddErrors(ctx context.Context) {
	if v, ok := ctx.Value(key).(*metrics); ok {
		v.errors.Add(1)
	}

}
func AddPanics(ctx context.Context) {
	if v, ok := ctx.Value(key).(*metrics); ok {
		v.panics.Add(1)
	}

}

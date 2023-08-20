package web

import (
	"context"
	"errors"
)

type ctxKey int

const key ctxKey = 1

type Values struct {
	TraceID    string
	StatusCode int
}

func GetValues(ctx context.Context) (*Values, error) {
	v, ok := ctx.Value(key).(*Values)
	if !ok {
		return nil, errors.New("web values missing from context")
	}
	return v, nil
}

func GetTraceID(ctx context.Context) string {
	v, ok := ctx.Value(key).(*Values)

	if !ok {
		return "00000000-0000-0000-0000-000000000000"

	}
	return v.TraceID
}

func SetStatusCode(ctx context.Context, statusCode int) error {
	v, ok := ctx.Value(key).(*Values)

	if !ok {
		return errors.New("web values missing from context")
	}
	v.StatusCode = statusCode
	return nil
}

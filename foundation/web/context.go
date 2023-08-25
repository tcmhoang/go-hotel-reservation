package web

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey int

const key ctxKey = 1

type Values struct {
	TraceID    string
	StatusCode int
	Tracer     trace.Tracer
}

func SetValues(ctx context.Context, v *Values) context.Context {
	return context.WithValue(ctx, key, v)
}

func GetValues(ctx context.Context) *Values {

	v, ok := ctx.Value(key).(*Values)
	if !ok {
		return &Values{
			TraceID: "00000000-0000-0000-0000-000000000000",
			Tracer:  trace.NewNoopTracerProvider().Tracer(""),
		}
	}

	return v
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

func AddSpan(ctx context.Context, spanName string, keyValues ...attribute.KeyValue) (context.Context, trace.Span) {
	v, ok := ctx.Value(key).(*Values)
	if !ok || v.Tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := v.Tracer.Start(ctx, spanName)
	for _, kv := range keyValues {
		span.SetAttributes(kv)
	}

	return ctx, span
}

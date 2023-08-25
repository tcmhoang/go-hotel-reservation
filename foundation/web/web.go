// Package web contains a small web framework extension.
package web

import (
	"context"
	"net/http"
	"os"
	"syscall"

	"github.com/dimfeld/httptreemux/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

type App struct {
	mux      *httptreemux.ContextMux
	otmux    http.Handler
	shutdown chan os.Signal
	mvs      []Middleware
	tracer   trace.Tracer
}

func NewApp(shutdown chan os.Signal, tracer trace.Tracer, mvs ...Middleware) *App {
	mux := httptreemux.NewContextMux()

	return &App{
		mux:      mux,
		shutdown: shutdown,
		mvs:      mvs,
		otmux:    otelhttp.NewHandler(mux, "request"),
		tracer:   tracer,
	}
}

func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.otmux.ServeHTTP(w, r)
}

func (a *App) Handle(method string, group string, path string, handler Handler, mvs ...Middleware) {
	fpath := path
	if group != "" {
		fpath = "/" + group + path
	}

	// first wrap the arg
	handler = withMiddleware(handler, mvs...)
	// then wrap the app mvs
	handler = withMiddleware(handler, a.mvs...)

	h := func(w http.ResponseWriter, r *http.Request) {

		ctx, span := a.startSpan(w, r)
		defer span.End()

		v := Values{
			TraceID: span.SpanContext().TraceID().String(),
		}
		ctx = context.WithValue(ctx, key, &v)

		if err := handler(ctx, w, r); err != nil {
			a.SignalShutdown()
			return
		}

	}

	a.mux.Handle(method, fpath, h)
}

func (a *App) startSpan(w http.ResponseWriter, r *http.Request) (context.Context, trace.Span) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	if a.tracer != nil {
		ctx, span = a.tracer.Start(ctx, "pkg.web.handle")
		span.SetAttributes(attribute.String("endpoint", r.RequestURI))
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(w.Header()))

	return ctx, span
}

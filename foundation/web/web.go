package web

import (
	"context"
	"net/http"
	"os"
	"syscall"

	"github.com/dimfeld/httptreemux/v5"
)

type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

type App struct {
	*httptreemux.ContextMux
	shutdown chan os.Signal
	mvs      []Middleware
}

func NewApp(shutdown chan os.Signal, mvs ...Middleware) *App {
	return &App{
		httptreemux.NewContextMux(),
		shutdown,
		mvs,
	}
}

func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
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

		if err := handler(r.Context(), w, r); err != nil {
			return
		}

	}

	a.ContextMux.Handle(method, fpath, h)
}

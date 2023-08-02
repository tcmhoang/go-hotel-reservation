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
}

func NewApp(shutdown chan os.Signal) *App {
	return &App{
		httptreemux.NewContextMux(),
		shutdown,
	}
}

func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
}

func (a *App) Handle(method string, group string, path string, hander Handler) {
	fpath := path
	if group != "" {
		fpath = "/" + group + path
	}

	h := func(w http.ResponseWriter, r *http.Request) {

		if err := hander(r.Context(), w, r); err != nil {
			return
		}

	}

	a.ContextMux.Handle(method, fpath, h)
}

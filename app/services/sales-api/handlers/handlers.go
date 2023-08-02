// Contains the full set of handler functions and routes
// Which is supported by the web api.
package handlers

import (
	"expvar"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/dimfeld/httptreemux/v5"
	chkgrp "github.com/tcmhoang/sservices/app/services/sales-api/handlers/debug"
	"github.com/tcmhoang/sservices/app/services/sales-api/handlers/v1/testgrp"
	"go.uber.org/zap"
)

func debugStdLibMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/vars", expvar.Handler())

	return mux
}

func DebugMux(build string, log *zap.SugaredLogger) http.Handler {
	mux := debugStdLibMux()

	cgh := chkgrp.Handlers{
		Build: build,
		Log:   log,
	}

	mux.HandleFunc("/debug/readiness", cgh.Readiness)
	mux.HandleFunc("/debug/liveness", cgh.Liveness)

	return mux
}

type APIMuxConfig struct {
	Shutdown chan os.Signal
	Log      *zap.SugaredLogger
}

func APIMux(cfg APIMuxConfig) *httptreemux.ContextMux {
	mux := httptreemux.NewContextMux()

	tgh := testgrp.Handlers{
		Log: cfg.Log,
	}

	mux.Handle(http.MethodGet, "/v1/test", tgh.Test)
	return mux

}

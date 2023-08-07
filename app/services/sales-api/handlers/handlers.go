// Contains the full set of handler functions and routes
// Which is supported by the web api.
package handlers

import (
	"expvar"
	"net/http"
	"net/http/pprof"
	"os"

	chkgrp "github.com/tcmhoang/sservices/app/services/sales-api/handlers/debug"
	"github.com/tcmhoang/sservices/app/services/sales-api/handlers/v1/testgrp"
	"github.com/tcmhoang/sservices/business/sys/auth"
	"github.com/tcmhoang/sservices/business/web/mids"
	"github.com/tcmhoang/sservices/foundation/web"
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
	Auth     *auth.Auth
}

func APIMux(cfg APIMuxConfig) *web.App {
	app := web.NewApp(
		cfg.Shutdown,
		mids.Logger(cfg.Log),
		mids.Errors(cfg.Log),
		mids.Metrics(),
		mids.Pacnics(),
	)

	v1(app, cfg)

	return app

}

func v1(app *web.App, cfg APIMuxConfig) {
	const ver = "v1"

	tgh := testgrp.Handlers{
		Log: cfg.Log,
	}

	app.Handle(http.MethodGet, ver, "/test", tgh.Test)
	app.Handle(http.MethodGet, ver, "/testauth",
		tgh.Test,
		mids.Authenticate(cfg.Auth),
		mids.Authorize(auth.Admin),
	)

}

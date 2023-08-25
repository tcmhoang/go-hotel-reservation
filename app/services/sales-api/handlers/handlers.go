// Package handlers contains the full set of handler functions and routes
// Which is supported by the web api.
package handlers

import (
	"expvar"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/jmoiron/sqlx"
	chkgrp "github.com/tcmhoang/sservices/app/services/sales-api/handlers/debug"
	"github.com/tcmhoang/sservices/app/services/sales-api/handlers/v1/testgrp"
	"github.com/tcmhoang/sservices/app/services/sales-api/handlers/v1/usergrp"
	usercore "github.com/tcmhoang/sservices/business/core/user"
	"github.com/tcmhoang/sservices/business/sys/auth"
	"github.com/tcmhoang/sservices/business/web/mids"
	"github.com/tcmhoang/sservices/foundation/web"
	"go.opentelemetry.io/otel/trace"
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

func DebugMux(build string, log *zap.SugaredLogger, db *sqlx.DB) http.Handler {
	mux := debugStdLibMux()

	cgh := chkgrp.Handlers{
		Build: build,
		Log:   log,
		DB:    db,
	}

	mux.HandleFunc("/debug/readiness", cgh.Readiness)
	mux.HandleFunc("/debug/liveness", cgh.Liveness)

	return mux
}

type APIMuxConfig struct {
	Shutdown chan os.Signal
	Log      *zap.SugaredLogger
	Auth     *auth.Auth
	DB       *sqlx.DB
	Tracer   trace.Tracer
}

func APIMux(cfg APIMuxConfig) *web.App {
	app := web.NewApp(
		cfg.Shutdown,
		cfg.Tracer,
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

	ugh := usergrp.New(usercore.NewCore(cfg.Log, cfg.DB), cfg.Auth)
	app.Handle(http.MethodGet, ver, "/users/token", ugh.Token)
	app.Handle(http.MethodGet, ver, "/users", ugh.Query, mids.Authenticate(cfg.Auth), mids.Authorize(auth.Admin))
	app.Handle(http.MethodGet, ver, "/users/:user_id", ugh.QueryByID, mids.Authenticate(cfg.Auth))
	app.Handle(http.MethodPost, ver, "/users", ugh.Create, mids.Authenticate(cfg.Auth), mids.Authorize(auth.Admin))
	app.Handle(http.MethodPut, ver, "/users/:user_id", ugh.Update, mids.Authenticate(cfg.Auth))
	app.Handle(http.MethodDelete, ver, "/users/:user_id", ugh.Delete, mids.Authenticate(cfg.Auth))

}

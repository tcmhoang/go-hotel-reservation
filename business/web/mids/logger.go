package mids

import (
	"context"
	"net/http"
	"time"

	"github.com/tcmhoang/sservices/foundation/web"
	"go.uber.org/zap"
)

func Logger(log *zap.SugaredLogger) web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

			v, err := web.GetValues(ctx)
			if err != nil {
				return err
			}

			log.Infow(
				"Request started",
				"traceid", v.TraceID,
				"method", r.Method,
				"path", r.URL.Path,
				"remoteaddr", r.RemoteAddr,
			)

			err = handler(ctx, w, r)

			log.Infow(
				"Request completed",
				"traceid", v.TraceID,
				"method", r.Method,
				"path", r.URL.Path,
				"remoteaddr", r.RemoteAddr,
				"statuscode", v.StatusCode,
				"since", time.Since(v.Now),
			)
			return err

		}
	}
}

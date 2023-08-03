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
			// TODO: Need to figure out how to implement trace id
			traceID := "000000000000000000000000000"
			// TODO: Status code based on the return value of the handler
			statuscode := http.StatusOK
			now := time.Now()

			log.Infow("Request started",
				"traceid", traceID,
				"method", r.Method,
				"path", r.URL.Path,
				"remoteaddr", r.RemoteAddr,
			)
			err := handler(ctx, w, r)

			log.Infow("Request completed",
				"traceid", traceID,
				"method", r.Method,
				"path", r.URL.Path,
				"remoteaddr", r.RemoteAddr,
				"statuscode", statuscode,
				"since", time.Since(now),
			)
			return err

		}
	}
}

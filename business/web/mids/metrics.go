package mids

import (
	"context"
	"net/http"

	"github.com/tcmhoang/sservices/business/sys/metrics"
	"github.com/tcmhoang/sservices/foundation/web"
)

func Metrics() web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			ctx = metrics.Set(ctx)

			err := handler(ctx, w, r)

			metrics.AddRequests(ctx)
			metrics.AddGoroutines(ctx)

			if err != nil {
				metrics.AddErrors(ctx)
			}
			return err
		}
	}

}

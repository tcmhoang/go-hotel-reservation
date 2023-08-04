package mids

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/tcmhoang/sservices/foundation/web"
)

func Pacnics() web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					trace := debug.Stack()

					err = fmt.Errorf("PANIC [%v]: STACK:\n%s", rec, string(trace))
				}
			}()
			return handler(ctx, w, r)
		}
	}
}

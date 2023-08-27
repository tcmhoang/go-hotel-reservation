package mids

import (
	"context"
	"net/http"

	"github.com/tcmhoang/sservices/business/sys/validation"
	"github.com/tcmhoang/sservices/foundation/web"
	"go.uber.org/zap"
)

func Errors(log *zap.SugaredLogger) web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			v := web.GetValues(ctx)

			if err := handler(ctx, w, r); err != nil {
				log.Errorw("ERROR", "traceid", v.TraceID, "ERROR", err)

				var er validation.ErrorResponse
				var statuscode int
				switch act := validation.Cause(err).(type) {
				case validation.FieldErrors:
					er = validation.ErrorResponse{
						Error:  "Data validation error",
						Fields: act.Fields(),
					}
					statuscode = http.StatusBadRequest
				case *validation.RequestError:
					er = validation.ErrorResponse{
						Error: act.Error(),
					}
					statuscode = act.Status
				default:
					er = validation.ErrorResponse{
						Error: http.StatusText(http.StatusInternalServerError),
					}
					statuscode = http.StatusInternalServerError

				}

				if err := web.Respond(ctx, w, er, statuscode); err != nil {
					return err
				}

				if web.IsShutdownErr(err) {
					return err
				}

			}

			return nil

		}
	}
}

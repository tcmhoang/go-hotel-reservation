// Package testgrp contains handlers for testing purposes
package testgrp

import (
	"context"
	"net/http"

	"github.com/tcmhoang/sservices/foundation/web"
	"go.uber.org/zap"
)

type Handlers struct {
	Log *zap.SugaredLogger
}

// Test handler is for dev
func (h Handlers) Test(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	statusCode := http.StatusOK

	return web.Respond(
		ctx,
		w,
		struct{ Status string }{Status: "OK"},
		statusCode,
	)
}

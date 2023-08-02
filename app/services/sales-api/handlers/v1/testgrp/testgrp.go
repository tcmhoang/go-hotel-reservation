package testgrp

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type Handlers struct {
	Log *zap.SugaredLogger
}

// Test handler is for dev
func (h Handlers) Test(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(struct {
		Status string
	}{Status: "OK"})

	statusCode := http.StatusOK
	h.Log.Infow("readiness", "statuscode", statusCode, "method", r.Method, "path", r.URL.Path, "remoteaddr", r.RemoteAddr)
}

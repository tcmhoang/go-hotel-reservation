// Package chkgrp maintains the groupof handlers for health checking
package chkgrp

import (
	"encoding/json"
	"net/http"
	"os"

	"go.uber.org/zap"
)

type Handlers struct {
	Build string
	Log   *zap.SugaredLogger
}

func (h Handlers) Readiness(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Status string `json:"status"`
	}{
		Status: "OK",
	}
	statusCode := http.StatusOK

	if err := response(w, statusCode, data); err != nil {
		h.Log.Errorw("readiness", "ERROR", err)
	}

	h.Log.Infow("readiness", "statuscode", statusCode, "method", r.Method, "path", r.URL.Path, "remoteaddr", r.RemoteAddr)
}

func (h Handlers) Liveness(w http.ResponseWriter, r *http.Request) {
	host, err := os.Hostname()
	if err != nil {
		host = "unavailable"
	}

	data := struct {
		Status    string `json:"status,omitempty"`
		Build     string `json:"build,omitempty"`
		Host      string `json:"host,omitempty"`
		Pod       string `json:"pod,omitempty"`
		PodIP     string `json:"pod_ip,omitempty"`
		Node      string `json:"node,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	}{
		Status:    "up",
		Build:     h.Build,
		Host:      host,
		Pod:       os.Getenv("KUBERNETES_PODNAME"),
		PodIP:     os.Getenv("KUBERNETES_NAMESPACE_POD_ID"),
		Node:      os.Getenv("KUBERNETES_NODENAME"),
		Namespace: os.Getenv("KUBERNETES_NAMESPACE"),
	}

	statusCode := http.StatusOK

	if err := response(w, statusCode, data); err != nil {
		h.Log.Errorw("liveness", "ERROR", err)
	}

	h.Log.Infow("liveness", "statuscode", statusCode, "method", r.Method, "path", r.URL.Path, "remoteaddr", r.RemoteAddr)
}

func response[A any](w http.ResponseWriter, statusCode int, data A) error {

	jsoned, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if _, err := w.Write(jsoned); err != nil {
		return err
	}

	return nil

}

package web

import (
	"context"
	"encoding/json"
	"net/http"
)

func Respond[A any](ctx context.Context, w http.ResponseWriter, data A, statusCode int) error {

	SetStatusCode(ctx, statusCode)

	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return nil
	}

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

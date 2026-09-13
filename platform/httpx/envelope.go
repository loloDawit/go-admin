// Package httpx supplies the wire shape for HTTP responses. It owns the
// envelope, never the catalogue of errors: each service maps its own sentinels
// in internal/httperr.
package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Message must be safe to show a client: no SQL, driver text, host names, or
// credentials. Log the cause instead.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Code: code, Message: message})
}

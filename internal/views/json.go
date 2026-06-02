package views

import (
	"encoding/json"
	"net/http"

	"github.com/alvor-technologies/iag-platform-go/apierr"
)

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Cache-Control", "no-store")
	code := statusToCode(status)
	apierr.WriteHTTP(w, status, code, message)
}

func statusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return apierr.CodeBadRequest
	case http.StatusUnauthorized:
		return apierr.CodeUnauthorized
	case http.StatusForbidden:
		return apierr.CodeForbidden
	case http.StatusNotFound:
		return apierr.CodeNotFound
	case http.StatusConflict:
		return apierr.CodeConflict
	case http.StatusTooManyRequests:
		return apierr.CodeTooManyRequests
	case http.StatusServiceUnavailable:
		return apierr.CodeServiceUnavailable
	default:
		return apierr.CodeInternal
	}
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

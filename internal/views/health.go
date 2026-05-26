package views

import "net/http"

func Health(w http.ResponseWriter) {
	JSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "iag-acp-api",
		"version": "v1",
	})
}

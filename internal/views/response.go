package views

import (
	"net/http"
	"strings"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

// WriteError maps model errors to HTTP responses (view layer).
func WriteError(w http.ResponseWriter, err error) {
	switch err {
	case models.ErrNotFound:
		Error(w, http.StatusNotFound, "not found")
	case models.ErrConflict:
		Error(w, http.StatusConflict, "conflict")
	case models.ErrValidation:
		Error(w, http.StatusBadRequest, err.Error())
	case models.ErrForbidden:
		Error(w, http.StatusForbidden, "forbidden")
	case models.ErrUnauthorized:
		Error(w, http.StatusUnauthorized, "unauthorized")
	case models.ErrPersistFailed:
		Error(w, http.StatusInternalServerError, "failed to save data")
	default:
		if err != nil && strings.Contains(err.Error(), "validation error") {
			Error(w, http.StatusBadRequest, err.Error())
			return
		}
		Error(w, http.StatusInternalServerError, "internal server error")
	}
}

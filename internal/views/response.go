package views

import (
	"net/http"
	"strings"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

// WriteError maps model errors to HTTP responses (view layer).
func WriteError(w http.ResponseWriter, err error) {
	switch err {
	case models.ErrNotFound:
		apierr.WriteHTTP(w, http.StatusNotFound, apierr.CodeNotFound, "resource not found")
	case models.ErrConflict:
		apierr.WriteHTTP(w, http.StatusConflict, apierr.CodeConflict, "resource conflict")
	case models.ErrValidation:
		apierr.WriteHTTP(w, http.StatusBadRequest, apierr.CodeValidation, err.Error())
	case models.ErrForbidden:
		apierr.WriteHTTP(w, http.StatusForbidden, apierr.CodeForbidden, "access denied")
	case models.ErrUnauthorized:
		apierr.WriteHTTP(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "authentication required")
	case models.ErrPersistFailed:
		apierr.WriteHTTP(w, http.StatusInternalServerError, apierr.CodeInternal, "failed to save data")
	default:
		if err != nil && strings.Contains(err.Error(), "validation error") {
			apierr.WriteHTTP(w, http.StatusBadRequest, apierr.CodeValidation, err.Error())
			return
		}
		apierr.WriteHTTP(w, http.StatusInternalServerError, apierr.CodeInternal, "internal server error")
	}
}

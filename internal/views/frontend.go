package views

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

func Frontend(w http.ResponseWriter, fe models.FrontendStore) {
	JSON(w, http.StatusOK, fe)
}

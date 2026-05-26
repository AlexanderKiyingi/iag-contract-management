package views

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

func Workspace(w http.ResponseWriter, ws models.Workspace) {
	JSON(w, http.StatusOK, ws)
}

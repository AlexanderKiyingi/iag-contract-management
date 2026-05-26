package views

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

func ContractList(w http.ResponseWriter, contracts []models.Contract) {
	JSON(w, http.StatusOK, contracts)
}

func Contract(w http.ResponseWriter, status int, c models.Contract) {
	JSON(w, status, c)
}

package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// SupplyContractController serves the coffee off-take contract domain. It reuses
// the contracts.* permission set (a supply contract is a contract) and is kept
// distinct from the construction-oriented governance contracts.
type SupplyContractController struct {
	model  *models.Store
	supply *persistence.SupplyStore
}

func NewSupplyContractController(model *models.Store, supply *persistence.SupplyStore) *SupplyContractController {
	return &SupplyContractController{model: model, supply: supply}
}

func (s *SupplyContractController) handleErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, persistence.ErrSupplyNotFound) {
		views.Error(w, http.StatusNotFound, "supply contract not found")
		return true
	}
	views.WriteError(w, err)
	return true
}

func (s *SupplyContractController) List(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), s.model, w, "contracts.read") {
		return
	}
	list, err := s.supply.ListContracts(r.Context())
	if s.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (s *SupplyContractController) Get(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), s.model, w, "contracts.read") {
		return
	}
	c, err := s.supply.GetContract(r.Context(), pathSegmentAfter(r, "supply-contracts"))
	if s.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, c)
}

func (s *SupplyContractController) Create(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), s.model, w, "contracts.create") {
		return
	}
	var in models.SupplyContractInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.FarmerBusinessID) == "" {
		views.Error(w, http.StatusBadRequest, "farmerId is required")
		return
	}
	status := in.Status
	if status == "" {
		status = models.SupplyDraft
	}
	if !status.Valid() {
		views.Error(w, http.StatusBadRequest, "invalid status")
		return
	}
	c, err := s.supply.CreateContract(r.Context(), models.SupplyContract{
		ID:                  models.NewGovID("SUP"),
		FarmerBusinessID:    strings.TrimSpace(in.FarmerBusinessID),
		FarmerName:          in.FarmerName,
		Variety:             in.Variety,
		CommittedWeightKg:   in.CommittedWeightKg,
		BasePricePerKgUgx:   in.BasePricePerKgUgx,
		QualityBonusRateUgx: in.QualityBonusRateUgx,
		ContractText:        in.ContractText,
		Status:              status,
	})
	if s.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusCreated, c)
}

func (s *SupplyContractController) Sign(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), s.model, w, "contracts.update") {
		return
	}
	var in struct {
		Signature string `json:"signature"`
	}
	_ = decodeJSON(r, &in)
	c, err := s.supply.Sign(r.Context(), pathSegmentAfter(r, "supply-contracts"), strings.TrimSpace(in.Signature))
	if s.handleErr(w, err) {
		return
	}
	views.JSON(w, http.StatusOK, c)
}

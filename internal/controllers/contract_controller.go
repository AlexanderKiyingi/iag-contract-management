package controllers

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/events"
	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

type ContractController struct {
	model  *models.Store
	events *events.Bus
}

func NewContractController(model *models.Store, bus *events.Bus) *ContractController {
	return &ContractController{model: model, events: bus}
}

func (c *ContractController) List(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "contracts.read") {
		return
	}
	list := c.model.ListContractsForSession(r.Context())
	page, pageSize, paginate := models.ParsePageQuery(r, 50, 500)
	if paginate {
		views.JSON(w, http.StatusOK, models.PaginateSlice(list, page, pageSize))
		return
	}
	views.ContractList(w, list)
}

func (c *ContractController) Get(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "contracts.read") {
		return
	}
	contract, err := c.model.GetContractForSession(r.Context(), lastPathSegment(r))
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.Contract(w, http.StatusOK, contract)
}

func (c *ContractController) Create(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "contracts.create") {
		return
	}
	var in models.ContractInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	contract, err := c.model.CreateContract(in)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	events.PublishContractCreated(r.Context(), c.events, contract)
	views.Contract(w, http.StatusCreated, contract)
}

func (c *ContractController) Patch(w http.ResponseWriter, r *http.Request) {
	no := lastPathSegment(r)
	if !c.model.CanEditContractCtx(r.Context(), no) {
		views.WriteError(w, models.ErrForbidden)
		return
	}
	var patch models.ContractPatch
	if err := decodeJSON(r, &patch); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var previous models.ContractStatus
	if patch.Status != nil {
		if existing, err := c.model.FindContract(no); err == nil {
			previous = existing.Status
		}
	}
	contract, err := c.model.PatchContract(no, patch)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	events.PublishContractUpdated(r.Context(), c.events, contract)
	if patch.Status != nil && *patch.Status != previous {
		events.PublishContractStatusChanged(r.Context(), c.events, contract, previous)
	}
	views.Contract(w, http.StatusOK, contract)
}

func (c *ContractController) Delete(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "contracts.delete") {
		return
	}
	no := lastPathSegment(r)
	existing, _ := c.model.FindContract(no)
	if err := c.model.DeleteContract(no); err != nil {
		views.WriteError(w, err)
		return
	}
	if existing.No != "" {
		events.PublishContractDeleted(r.Context(), c.events, existing)
	}
	views.NoContent(w)
}

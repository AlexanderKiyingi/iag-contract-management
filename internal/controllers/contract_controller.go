package controllers

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

type ContractController struct {
	model *models.Store
}

func NewContractController(model *models.Store) *ContractController {
	return &ContractController{model: model}
}

func (c *ContractController) List(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "contracts.read") {
		return
	}
	list := c.model.ListContracts()
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
	contract, err := c.model.FindContract(lastPathSegment(r))
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
	contract, err := c.model.PatchContract(no, patch)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.Contract(w, http.StatusOK, contract)
}

func (c *ContractController) Delete(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), c.model, w, "contracts.delete") {
		return
	}
	if err := c.model.DeleteContract(lastPathSegment(r)); err != nil {
		views.WriteError(w, err)
		return
	}
	views.NoContent(w)
}

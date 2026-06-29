package controllers

import (
	"net/http"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

// The contractor portal: read-only governance endpoints that are always scoped
// to the contractor linked to the caller's platform user (gov_contractors.
// platform_user_id == JWT subject). No module permissions are required — access
// is self-scoped by ownership, so a contractor sees only their own contracts.

// resolvePortalContractor returns the contractor bound to the caller, or writes
// an error response and returns nil.
func (g *GovernanceController) resolvePortalContractor(w http.ResponseWriter, r *http.Request) *models.GovContractor {
	sess := g.model.SessionFromRequest(r.Context())
	if sess.UserID == "" && sess.Email == "" {
		views.Error(w, http.StatusUnauthorized, "authentication required")
		return nil
	}
	c, err := g.gov.GetContractorForUser(r.Context(), sess.UserID, sess.Email)
	if err != nil || c == nil {
		views.Error(w, http.StatusForbidden, "no contractor profile is linked to your account")
		return nil
	}
	return c
}

// ownedContract loads the :id contract and verifies it belongs to contractorID.
func (g *GovernanceController) ownedContract(w http.ResponseWriter, r *http.Request, contractorID string) *models.GovContract {
	c, err := g.gov.GetContract(r.Context(), pathSegmentAfter(r, "contracts"))
	if g.handleErr(w, err) {
		return nil
	}
	if c.ContractorID != contractorID {
		views.Error(w, http.StatusForbidden, "contract not available")
		return nil
	}
	return c
}

// PortalMe returns the contractor profile linked to the caller.
func (g *GovernanceController) PortalMe(w http.ResponseWriter, r *http.Request) {
	c := g.resolvePortalContractor(w, r)
	if c == nil {
		return
	}
	views.JSON(w, http.StatusOK, c)
}

// PortalContracts lists the caller-contractor's contracts.
func (g *GovernanceController) PortalContracts(w http.ResponseWriter, r *http.Request) {
	c := g.resolvePortalContractor(w, r)
	if c == nil {
		return
	}
	all, err := g.gov.ListContracts(r.Context())
	if err != nil {
		views.WriteError(w, err)
		return
	}
	out := make([]models.GovContract, 0)
	for _, k := range all {
		if k.ContractorID == c.ID {
			out = append(out, k)
		}
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": out})
}

// PortalContract returns a single owned contract.
func (g *GovernanceController) PortalContract(w http.ResponseWriter, r *http.Request) {
	c := g.resolvePortalContractor(w, r)
	if c == nil {
		return
	}
	con := g.ownedContract(w, r, c.ID)
	if con == nil {
		return
	}
	views.JSON(w, http.StatusOK, con)
}

// PortalMilestones lists milestones for an owned contract.
func (g *GovernanceController) PortalMilestones(w http.ResponseWriter, r *http.Request) {
	c := g.resolvePortalContractor(w, r)
	if c == nil {
		return
	}
	con := g.ownedContract(w, r, c.ID)
	if con == nil {
		return
	}
	list, err := g.gov.ListMilestones(r.Context(), con.ID)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// PortalVariations lists variations for an owned contract.
func (g *GovernanceController) PortalVariations(w http.ResponseWriter, r *http.Request) {
	c := g.resolvePortalContractor(w, r)
	if c == nil {
		return
	}
	con := g.ownedContract(w, r, c.ID)
	if con == nil {
		return
	}
	list, err := g.gov.ListVariations(r.Context(), con.ID)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// PortalDocURL returns a presigned download URL for a document on an owned
// contract — the portal equivalent of PresignDownloadDoc, scoped by ownership
// (contractors lack contracts.read, so they cannot use the admin endpoint).
func (g *GovernanceController) PortalDocURL(w http.ResponseWriter, r *http.Request) {
	c := g.resolvePortalContractor(w, r)
	if c == nil {
		return
	}
	con := g.ownedContract(w, r, c.ID)
	if con == nil {
		return
	}
	if !g.storageReady(w) {
		return
	}
	docID := pathSegmentAfter(r, "documents")
	for _, d := range con.Documents {
		if d.ID == docID {
			if d.Key == "" {
				views.Error(w, http.StatusNotFound, "no file attached to this document")
				return
			}
			views.JSON(w, http.StatusOK, map[string]string{
				"url": g.docs.PresignGet(d.Key, docPresignExpiry),
			})
			return
		}
	}
	views.Error(w, http.StatusNotFound, "document not found")
}

// PortalReports lists progress reports for an owned contract.
func (g *GovernanceController) PortalReports(w http.ResponseWriter, r *http.Request) {
	c := g.resolvePortalContractor(w, r)
	if c == nil {
		return
	}
	con := g.ownedContract(w, r, c.ID)
	if con == nil {
		return
	}
	list, err := g.gov.ListProgressReports(r.Context(), con.ID)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]any{"items": list})
}

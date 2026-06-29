package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

const docPresignExpiry = 15 * time.Minute

type presignUploadInput struct {
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        string `json:"size"`
}

type presignUploadResult struct {
	DocID     string `json:"docId"`
	Key       string `json:"key"`
	UploadURL string `json:"uploadUrl"`
}

// safeName reduces a filename to a key-safe token (keeps the extension).
func safeName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "file"
	}
	return out
}

func (g *GovernanceController) storageReady(w http.ResponseWriter) bool {
	if g.docs.IsEnabled() {
		return true
	}
	views.Error(w, http.StatusServiceUnavailable, "document storage is not configured")
	return false
}

// PresignContractDoc issues a presigned PUT URL for uploading a document to a
// contract. The client uploads directly to the bucket, then calls AddContractDoc.
func (g *GovernanceController) PresignContractDoc(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.update") {
		return
	}
	if !g.storageReady(w) {
		return
	}
	cid := pathSegmentAfter(r, "contracts")
	if _, err := g.gov.GetContract(r.Context(), cid); g.handleErr(w, err) {
		return
	}
	var in presignUploadInput
	if err := decodeJSON(r, &in); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		views.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	docID := models.NewGovID("GDOC")
	key := fmt.Sprintf("governance/contracts/%s/%s-%s", cid, docID, safeName(in.Name))
	views.JSON(w, http.StatusOK, presignUploadResult{
		DocID:     docID,
		Key:       key,
		UploadURL: g.docs.PresignPut(key, docPresignExpiry),
	})
}

// AddContractDoc records uploaded-document metadata on the contract (the file
// itself already lives in the bucket under doc.Key).
func (g *GovernanceController) AddContractDoc(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.update") {
		return
	}
	cid := pathSegmentAfter(r, "contracts")
	c, err := g.gov.GetContract(r.Context(), cid)
	if g.handleErr(w, err) {
		return
	}
	var doc models.GovDoc
	if err := decodeJSON(r, &doc); err != nil {
		views.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(doc.Name) == "" {
		views.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if doc.ID == "" {
		doc.ID = models.NewGovID("GDOC")
	}
	if doc.Date == "" {
		doc.Date = nowStamp()
	}
	if doc.UploadedBy == "" {
		doc.UploadedBy = g.actor(r, "")
	}
	c.Documents = append(c.Documents, doc)
	updated, err := g.gov.UpdateContract(r.Context(), *c)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusCreated, updated)
}

// PresignDownloadDoc returns a short-lived presigned GET URL for an object key.
func (g *GovernanceController) PresignDownloadDoc(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.read") {
		return
	}
	if !g.storageReady(w) {
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		views.Error(w, http.StatusBadRequest, "key query parameter is required")
		return
	}
	views.JSON(w, http.StatusOK, map[string]string{
		"url": g.docs.PresignGet(key, docPresignExpiry),
	})
}

// DeleteContractDoc removes a document's metadata from the contract. Object
// cleanup in the bucket is left to a lifecycle policy.
func (g *GovernanceController) DeleteContractDoc(w http.ResponseWriter, r *http.Request) {
	if !requirePerm(r.Context(), g.model, w, "contracts.update") {
		return
	}
	cid := pathSegmentAfter(r, "contracts")
	docID := pathSegmentAfter(r, "documents")
	c, err := g.gov.GetContract(r.Context(), cid)
	if g.handleErr(w, err) {
		return
	}
	kept := make([]models.GovDoc, 0, len(c.Documents))
	for _, d := range c.Documents {
		if d.ID != docID {
			kept = append(kept, d)
		}
	}
	c.Documents = kept
	updated, err := g.gov.UpdateContract(r.Context(), *c)
	if err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, updated)
}

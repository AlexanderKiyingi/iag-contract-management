package controllers

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/views"
)

const maxUploadBytes = 2 * 1024 * 1024

type UploadsController struct {
	model *models.Store
}

func NewUploadsController(model *models.Store) *UploadsController {
	return &UploadsController{model: model}
}

// UploadProfile accepts JSON { email, dataUrl } or multipart form field "file".
func (c *UploadsController) UploadProfile(w http.ResponseWriter, r *http.Request) {
	email := c.model.GetSessionCtx(r.Context()).Email
	var dataURL string

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxUploadBytes + 512); err != nil {
			views.Error(w, http.StatusBadRequest, "file too large (max 2 MB)")
			return
		}
		if q := r.FormValue("email"); q != "" {
			email = q
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			views.Error(w, http.StatusBadRequest, "missing file field")
			return
		}
		defer file.Close()
		if header.Size > maxUploadBytes {
			views.Error(w, http.StatusBadRequest, "file too large (max 2 MB)")
			return
		}
		raw, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
		if err != nil {
			views.Error(w, http.StatusInternalServerError, "read failed")
			return
		}
		if len(raw) > maxUploadBytes {
			views.Error(w, http.StatusBadRequest, "file too large (max 2 MB)")
			return
		}
		mime := header.Header.Get("Content-Type")
		if mime == "" {
			mime = "image/jpeg"
		}
		if !strings.HasPrefix(mime, "image/") {
			views.Error(w, http.StatusBadRequest, "only image uploads allowed")
			return
		}
		dataURL = fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(raw))
	} else {
		var in models.ProfilePhotoInput
		if err := decodeJSON(r, &in); err != nil {
			views.Error(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if in.Email != "" {
			email = in.Email
		}
		dataURL = in.DataURL
	}

	if !canEditProfile(r.Context(), c.model, email) {
		views.WriteError(w, models.ErrForbidden)
		return
	}
	if err := c.model.SetProfilePhoto(email, dataURL); err != nil {
		views.WriteError(w, err)
		return
	}
	views.JSON(w, http.StatusOK, map[string]string{
		"email":   email,
		"dataUrl": c.model.GetProfilePhoto(email),
	})
}

func (c *UploadsController) GetProfile(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		email = c.model.GetSessionCtx(r.Context()).Email
	}
	views.JSON(w, http.StatusOK, map[string]string{
		"email":   email,
		"dataUrl": c.model.GetProfilePhoto(email),
	})
}

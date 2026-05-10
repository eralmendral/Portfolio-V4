package certificates

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	store   Store
	uploads UploadConfig
}

func NewHandler(store Store, uploads UploadConfig) *Handler {
	return &Handler{
		store:   store,
		uploads: uploads.withDefaults(),
	}
}

func (h *Handler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listCertificates(w, r)
	case http.MethodPost:
		h.createCertificate(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/certificates/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "certificate not found", nil)
		return
	}

	idOrSlug := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.getCertificate(w, r, idOrSlug)
		case http.MethodPut, http.MethodPatch:
			h.updateCertificate(w, r, idOrSlug)
		case http.MethodDelete:
			h.deleteCertificate(w, r, idOrSlug)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
		return
	}

	if len(parts) == 2 && parts[1] == "image" && r.Method == http.MethodPost {
		h.uploadImage(w, r, idOrSlug)
		return
	}

	if len(parts) == 2 && parts[1] == "image" && r.Method == http.MethodDelete {
		h.deleteImage(w, r, idOrSlug)
		return
	}

	writeError(w, http.StatusNotFound, "route not found", nil)
}

func (h *Handler) listCertificates(w http.ResponseWriter, r *http.Request) {
	var featured *bool
	if rawFeatured := r.URL.Query().Get("featured"); rawFeatured != "" {
		parsed, err := strconv.ParseBool(rawFeatured)
		if err != nil {
			writeError(w, http.StatusBadRequest, "featured must be true or false", nil)
			return
		}
		featured = &parsed
	}

	certificates, err := h.store.List(r.Context(), ListFilter{
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Featured: featured,
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list certificates", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"certificates": certificates,
	})
}

func (h *Handler) createCertificate(w http.ResponseWriter, r *http.Request) {
	var input CreateCertificateRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid certificate payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	certificate := certificateFromCreate(input)
	created, err := h.store.Create(r.Context(), certificate)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/certificates/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getCertificate(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	certificate, err := h.store.Get(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, certificate)
}

func (h *Handler) updateCertificate(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	var input UpdateCertificateRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid certificate payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), idOrSlug, func(certificate *Certificate) error {
		applyUpdate(certificate, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteCertificate(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	certificate, err := h.store.Get(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	if err := h.store.Delete(r.Context(), idOrSlug); err != nil {
		writeHandledError(w, err)
		return
	}

	h.removeCertificateFiles(certificate)
	w.WriteHeader(http.StatusNoContent)
}

func certificateFromCreate(input CreateCertificateRequest) Certificate {
	status := normalizeStatus(input.Status)
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugify(input.Title)
	}

	certificate := Certificate{
		Slug:          slug,
		Title:         strings.TrimSpace(input.Title),
		Issuer:        strings.TrimSpace(input.Issuer),
		Summary:       strings.TrimSpace(input.Summary),
		Description:   strings.TrimSpace(input.Description),
		CredentialURL: normalizeURL(input.CredentialURL),
		Featured:      input.Featured,
		SortOrder:     input.SortOrder,
		Status:        status,
		IssuedAt:      utcTime(input.IssuedAt),
		ExpiresAt:     utcTime(input.ExpiresAt),
	}

	if input.Image != nil {
		image := imageFromInput(*input.Image)
		certificate.Image = &image
	}

	return certificate
}

func applyUpdate(certificate *Certificate, input UpdateCertificateRequest) {
	if input.Slug != nil {
		certificate.Slug = strings.TrimSpace(*input.Slug)
		if certificate.Slug == "" {
			certificate.Slug = slugify(certificate.Title)
		}
	}
	if input.Title != nil {
		certificate.Title = strings.TrimSpace(*input.Title)
		if certificate.Slug == "" {
			certificate.Slug = slugify(certificate.Title)
		}
	}
	if input.Issuer != nil {
		certificate.Issuer = strings.TrimSpace(*input.Issuer)
	}
	if input.Summary != nil {
		certificate.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.Description != nil {
		certificate.Description = strings.TrimSpace(*input.Description)
	}
	if input.CredentialURL != nil {
		certificate.CredentialURL = normalizeURL(*input.CredentialURL)
	}
	if input.Image != nil {
		image := imageFromInput(*input.Image)
		certificate.Image = &image
	}
	if input.Featured != nil {
		certificate.Featured = *input.Featured
	}
	if input.SortOrder != nil {
		certificate.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		certificate.Status = normalizeStatus(*input.Status)
	}
	if input.IssuedAt != nil {
		certificate.IssuedAt = utcTime(input.IssuedAt)
	}
	if input.ExpiresAt != nil {
		certificate.ExpiresAt = utcTime(input.ExpiresAt)
	}
}

func imageFromInput(input CertificateImageInput) CertificateImage {
	return CertificateImage{
		ID:         newID(),
		URL:        strings.TrimSpace(input.URL),
		AltText:    strings.TrimSpace(input.AltText),
		Caption:    strings.TrimSpace(input.Caption),
		UploadedAt: time.Now().UTC(),
	}
}

func utcTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	converted := value.UTC()
	return &converted
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeHandledError(w http.ResponseWriter, err error) {
	if validation, ok := isValidationError(err); ok {
		writeError(w, http.StatusBadRequest, "validation failed", validation.Fields())
		return
	}
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "certificate not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "certificate slug or id already exists", nil)
		return
	}

	writeError(w, http.StatusInternalServerError, "internal server error", nil)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string, fields map[string]string) {
	payload := map[string]any{
		"error": message,
	}
	if len(fields) > 0 {
		payload["fields"] = fields
	}

	writeJSON(w, status, payload)
}

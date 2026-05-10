package intro

import (
	"encoding/json"
	"errors"
	"net/http"
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

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/intro" {
		writeError(w, http.StatusNotFound, "route not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getIntro(w, r)
	case http.MethodPut, http.MethodPatch:
		h.updateIntro(w, r)
	case http.MethodDelete:
		h.deleteIntro(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleProfilePicture(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/intro/profile-picture" {
		writeError(w, http.StatusNotFound, "route not found", nil)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.uploadProfilePicture(w, r)
	case http.MethodDelete:
		h.deleteProfilePicture(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) getIntro(w http.ResponseWriter, r *http.Request) {
	intro, err := h.store.Get(r.Context())
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, intro)
}

func (h *Handler) updateIntro(w http.ResponseWriter, r *http.Request) {
	var input UpdateIntroRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid intro payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	current, err := h.store.Get(r.Context())
	if errors.Is(err, ErrNotFound) {
		current = Intro{ID: DefaultID}
	} else if err != nil {
		writeHandledError(w, err)
		return
	}

	previous := current.ProfilePicture
	next := current
	applyUpdate(&next, input)
	if err := validateSavedIntro(next); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Save(r.Context(), next)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	if previous != nil && next.ProfilePicture != nil && previous.Path != next.ProfilePicture.Path {
		h.removeImageFile(*previous)
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteIntro(w http.ResponseWriter, r *http.Request) {
	current, err := h.store.Get(r.Context())
	if err != nil {
		writeHandledError(w, err)
		return
	}

	if err := h.store.Delete(r.Context()); err != nil {
		writeHandledError(w, err)
		return
	}

	h.removeIntroFiles(current)
	w.WriteHeader(http.StatusNoContent)
}

func applyUpdate(intro *Intro, input UpdateIntroRequest) {
	if input.Title != nil {
		intro.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		intro.Description = strings.TrimSpace(*input.Description)
	}
	if input.ProfilePicture != nil {
		image := imageFromInput(*input.ProfilePicture)
		intro.ProfilePicture = &image
	}
}

func imageFromInput(input IntroImageInput) IntroImage {
	return IntroImage{
		ID:         newID(),
		URL:        strings.TrimSpace(input.URL),
		AltText:    strings.TrimSpace(input.AltText),
		Caption:    strings.TrimSpace(input.Caption),
		UploadedAt: time.Now().UTC(),
	}
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
		writeError(w, http.StatusNotFound, "intro not found", nil)
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

package contact

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	store Store
}

func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/contact-submissions" {
		writeError(w, http.StatusNotFound, "route not found", nil)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleProfile(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/contact-profile" {
		writeError(w, http.StatusNotFound, "route not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getProfile(w, r)
	case http.MethodPatch, http.MethodPut:
		h.updateProfile(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input CreateSubmissionRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid contact payload", nil)
		return
	}

	submission, err := validateAndNormalize(input)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	created, err := h.store.Create(r.Context(), submission)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.store.GetProfile(r.Context())
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	var input UpdateProfileRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid contact profile payload", nil)
		return
	}

	current, err := h.store.GetProfile(r.Context())
	if errors.Is(err, ErrNotFound) {
		current = Profile{ID: DefaultProfileID}
	} else if err != nil {
		writeHandledError(w, err)
		return
	}

	applyProfileUpdate(&current, input)
	updated, err := h.store.SaveProfile(r.Context(), current)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
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
	if errors.Is(err, ErrInvalid) {
		writeError(w, http.StatusBadRequest, "invalid contact submission", nil)
		return
	}
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "contact profile not found", nil)
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

package links

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	store Store
}

func NewHandler(store Store) *Handler {
	return &Handler{
		store: store,
	}
}

func (h *Handler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listLinks(w, r)
	case http.MethodPost:
		h.createLink(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/links/"), "/"), "/")
	if len(parts) != 1 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "link not found", nil)
		return
	}

	id := parts[0]
	switch r.Method {
	case http.MethodGet:
		h.getLink(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateLink(w, r, id)
	case http.MethodDelete:
		h.deleteLink(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listLinks(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = StatusPublished
	} else {
		status = normalizeStatus(status)
	}

	var star *bool
	if rawStar := r.URL.Query().Get("star"); rawStar != "" {
		parsed, err := strconv.ParseBool(rawStar)
		if err != nil {
			writeError(w, http.StatusBadRequest, "star must be true or false", nil)
			return
		}
		star = &parsed
	}

	filter := ListFilter{
		Status: status,
		Star:   star,
		Query:  strings.TrimSpace(r.URL.Query().Get("q")),
	}
	if err := validateListFilter(filter); err != nil {
		writeHandledError(w, err)
		return
	}

	links, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list links", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"links": links,
	})
}

func (h *Handler) createLink(w http.ResponseWriter, r *http.Request) {
	var input CreateLinkRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid link payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	link := linkFromCreate(input)
	created, err := h.store.Create(r.Context(), link)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/links/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getLink(w http.ResponseWriter, r *http.Request, id string) {
	link, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, link)
}

func (h *Handler) updateLink(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateLinkRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid link payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), id, func(link *Link) error {
		applyUpdate(link, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteLink(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func linkFromCreate(input CreateLinkRequest) Link {
	return Link{
		Label:     strings.TrimSpace(input.Label),
		URL:       normalizeURL(input.URL),
		IconClass: strings.TrimSpace(input.IconClass),
		SortOrder: input.SortOrder,
		Star:      input.Star,
		Status:    normalizeStatus(input.Status),
	}
}

func applyUpdate(link *Link, input UpdateLinkRequest) {
	if input.Label != nil {
		link.Label = strings.TrimSpace(*input.Label)
	}
	if input.URL != nil {
		link.URL = normalizeURL(*input.URL)
	}
	if input.IconClass != nil {
		link.IconClass = strings.TrimSpace(*input.IconClass)
	}
	if input.SortOrder != nil {
		link.SortOrder = *input.SortOrder
	}
	if input.Star != nil {
		link.Star = *input.Star
	}
	if input.Status != nil {
		link.Status = normalizeStatus(*input.Status)
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
		writeError(w, http.StatusNotFound, "link not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "link already exists", nil)
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

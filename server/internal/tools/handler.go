package tools

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
	return &Handler{store: store}
}

func (h *Handler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTools(w, r)
	case http.MethodPost:
		h.createTool(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/tools/"), "/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "tool not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getTool(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateTool(w, r, id)
	case http.MethodDelete:
		h.deleteTool(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listTools(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = StatusPublished
	} else {
		status = normalizeStatus(status)
	}

	var featured *bool
	if rawFeatured := r.URL.Query().Get("featured"); rawFeatured != "" {
		parsed, err := strconv.ParseBool(rawFeatured)
		if err != nil {
			writeError(w, http.StatusBadRequest, "featured must be true or false", nil)
			return
		}
		featured = &parsed
	}

	filter := ListFilter{
		Status:   status,
		Featured: featured,
		Category: strings.TrimSpace(r.URL.Query().Get("category")),
		Tag:      strings.TrimSpace(r.URL.Query().Get("tag")),
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
	}
	if err := validateListFilter(filter); err != nil {
		writeHandledError(w, err)
		return
	}

	tools, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list tools", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tools": tools,
	})
}

func (h *Handler) createTool(w http.ResponseWriter, r *http.Request) {
	var input CreateToolRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid tool payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	tool := toolFromCreate(input)
	created, err := h.store.Create(r.Context(), tool)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/tools/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getTool(w http.ResponseWriter, r *http.Request, id string) {
	tool, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tool)
}

func (h *Handler) updateTool(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateToolRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid tool payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), id, func(tool *Tool) error {
		applyUpdate(tool, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteTool(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toolFromCreate(input CreateToolRequest) Tool {
	return Tool{
		Name:      strings.TrimSpace(input.Name),
		Category:  strings.TrimSpace(input.Category),
		Summary:   strings.TrimSpace(input.Summary),
		IconClass: strings.TrimSpace(input.IconClass),
		Tags:      normalizeStringSlice(input.Tags),
		SortOrder: input.SortOrder,
		Featured:  input.Featured,
		Status:    normalizeStatus(input.Status),
	}
}

func applyUpdate(tool *Tool, input UpdateToolRequest) {
	if input.Name != nil {
		tool.Name = strings.TrimSpace(*input.Name)
	}
	if input.Category != nil {
		tool.Category = strings.TrimSpace(*input.Category)
	}
	if input.Summary != nil {
		tool.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.IconClass != nil {
		tool.IconClass = strings.TrimSpace(*input.IconClass)
	}
	if input.Tags != nil {
		tool.Tags = normalizeStringSlice(*input.Tags)
	}
	if input.SortOrder != nil {
		tool.SortOrder = *input.SortOrder
	}
	if input.Featured != nil {
		tool.Featured = *input.Featured
	}
	if input.Status != nil {
		tool.Status = normalizeStatus(*input.Status)
	}
}

func normalizeStringSlice(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
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
		writeError(w, http.StatusNotFound, "tool not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "tool already exists", nil)
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

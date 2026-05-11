package series

import (
	"encoding/json"
	"errors"
	"net/http"
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
		h.listSeries(w, r)
	case http.MethodPost:
		h.createSeries(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/series/"), "/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "series entry not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getSeries(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateSeries(w, r, id)
	case http.MethodDelete:
		h.deleteSeries(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listSeries(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = StatusPublished
	} else {
		status = normalizeStatus(status)
	}

	filter := ListFilter{
		Status:   status,
		Category: normalizeCategory(r.URL.Query().Get("category")),
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
		From:     strings.TrimSpace(r.URL.Query().Get("from")),
		To:       strings.TrimSpace(r.URL.Query().Get("to")),
	}
	if err := validateListFilter(filter); err != nil {
		writeHandledError(w, err)
		return
	}

	series, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list series entries", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"series": series,
	})
}

func (h *Handler) createSeries(w http.ResponseWriter, r *http.Request) {
	var input CreateSeriesRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid series payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	series := seriesFromCreate(input)
	created, err := h.store.Create(r.Context(), series)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/series/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getSeries(w http.ResponseWriter, r *http.Request, id string) {
	series, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, series)
}

func (h *Handler) updateSeries(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateSeriesRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid series payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), id, func(series *Series) error {
		applyUpdate(series, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteSeries(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func seriesFromCreate(input CreateSeriesRequest) Series {
	return Series{
		Title:           strings.TrimSpace(input.Title),
		Category:        normalizeCategory(input.Category),
		Creator:         strings.TrimSpace(input.Creator),
		Platform:        strings.TrimSpace(input.Platform),
		WatchURL:        normalizeURL(input.WatchURL),
		MostlyWatchedOn: normalizeDate(input.MostlyWatchedOn),
		Notes:           strings.TrimSpace(input.Notes),
		SortOrder:       input.SortOrder,
		Status:          normalizeStatus(input.Status),
	}
}

func applyUpdate(series *Series, input UpdateSeriesRequest) {
	if input.Title != nil {
		series.Title = strings.TrimSpace(*input.Title)
	}
	if input.Category != nil {
		series.Category = normalizeCategory(*input.Category)
	}
	if input.Creator != nil {
		series.Creator = strings.TrimSpace(*input.Creator)
	}
	if input.Platform != nil {
		series.Platform = strings.TrimSpace(*input.Platform)
	}
	if input.WatchURL != nil {
		series.WatchURL = normalizeURL(*input.WatchURL)
	}
	if input.MostlyWatchedOn != nil {
		series.MostlyWatchedOn = normalizeDate(*input.MostlyWatchedOn)
	}
	if input.Notes != nil {
		series.Notes = strings.TrimSpace(*input.Notes)
	}
	if input.SortOrder != nil {
		series.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		series.Status = normalizeStatus(*input.Status)
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
		writeError(w, http.StatusNotFound, "series entry not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "series entry already exists", nil)
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

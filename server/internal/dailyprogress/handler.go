package dailyprogress

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
		h.listDailyProgress(w, r)
	case http.MethodPost:
		h.createDailyProgress(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	idOrDate := strings.Trim(strings.TrimPrefix(r.URL.Path, "/daily-progress/"), "/")
	if idOrDate == "" || strings.Contains(idOrDate, "/") {
		writeError(w, http.StatusNotFound, "daily progress entry not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getDailyProgress(w, r, idOrDate)
	case http.MethodPut, http.MethodPatch:
		h.updateDailyProgress(w, r, idOrDate)
	case http.MethodDelete:
		h.deleteDailyProgress(w, r, idOrDate)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listDailyProgress(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = StatusPublished
	} else {
		status = normalizeStatus(status)
	}

	filter := ListFilter{
		Status: status,
		Tag:    strings.TrimSpace(r.URL.Query().Get("tag")),
		Query:  strings.TrimSpace(r.URL.Query().Get("q")),
		From:   strings.TrimSpace(r.URL.Query().Get("from")),
		To:     strings.TrimSpace(r.URL.Query().Get("to")),
	}
	if err := validateListFilter(filter); err != nil {
		writeHandledError(w, err)
		return
	}

	entries, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list daily progress entries", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"daily_progress": entries,
	})
}

func (h *Handler) createDailyProgress(w http.ResponseWriter, r *http.Request) {
	var input CreateDailyProgressRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid daily progress payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	entry := dailyProgressFromCreate(input)
	created, err := h.store.Create(r.Context(), entry)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/daily-progress/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getDailyProgress(w http.ResponseWriter, r *http.Request, idOrDate string) {
	entry, err := h.store.Get(r.Context(), idOrDate)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) updateDailyProgress(w http.ResponseWriter, r *http.Request, idOrDate string) {
	var input UpdateDailyProgressRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid daily progress payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), idOrDate, func(entry *DailyProgress) error {
		applyUpdate(entry, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteDailyProgress(w http.ResponseWriter, r *http.Request, idOrDate string) {
	if err := h.store.Delete(r.Context(), idOrDate); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func dailyProgressFromCreate(input CreateDailyProgressRequest) DailyProgress {
	return DailyProgress{
		EntryDate:     normalizeDate(input.EntryDate),
		Title:         strings.TrimSpace(input.Title),
		Summary:       strings.TrimSpace(input.Summary),
		Content:       strings.TrimSpace(input.Content),
		Mood:          strings.TrimSpace(input.Mood),
		ProgressScore: input.ProgressScore,
		Wins:          normalizeStringSlice(input.Wins),
		Blockers:      normalizeStringSlice(input.Blockers),
		Learnings:     normalizeStringSlice(input.Learnings),
		NextSteps:     normalizeStringSlice(input.NextSteps),
		Tags:          normalizeStringSlice(input.Tags),
		Status:        normalizeStatus(input.Status),
	}
}

func applyUpdate(entry *DailyProgress, input UpdateDailyProgressRequest) {
	if input.EntryDate != nil {
		entry.EntryDate = normalizeDate(*input.EntryDate)
	}
	if input.Title != nil {
		entry.Title = strings.TrimSpace(*input.Title)
	}
	if input.Summary != nil {
		entry.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.Content != nil {
		entry.Content = strings.TrimSpace(*input.Content)
	}
	if input.Mood != nil {
		entry.Mood = strings.TrimSpace(*input.Mood)
	}
	if input.ProgressScore != nil {
		entry.ProgressScore = *input.ProgressScore
	}
	if input.Wins != nil {
		entry.Wins = normalizeStringSlice(*input.Wins)
	}
	if input.Blockers != nil {
		entry.Blockers = normalizeStringSlice(*input.Blockers)
	}
	if input.Learnings != nil {
		entry.Learnings = normalizeStringSlice(*input.Learnings)
	}
	if input.NextSteps != nil {
		entry.NextSteps = normalizeStringSlice(*input.NextSteps)
	}
	if input.Tags != nil {
		entry.Tags = normalizeStringSlice(*input.Tags)
	}
	if input.Status != nil {
		entry.Status = normalizeStatus(*input.Status)
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
		writeError(w, http.StatusNotFound, "daily progress entry not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "daily progress entry already exists", nil)
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

package music

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
		h.listMusic(w, r)
	case http.MethodPost:
		h.createMusic(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/music/"), "/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "music entry not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getMusic(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateMusic(w, r, id)
	case http.MethodDelete:
		h.deleteMusic(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listMusic(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = StatusPublished
	} else {
		status = normalizeStatus(status)
	}

	filter := ListFilter{
		Status: status,
		Query:  strings.TrimSpace(r.URL.Query().Get("q")),
		From:   strings.TrimSpace(r.URL.Query().Get("from")),
		To:     strings.TrimSpace(r.URL.Query().Get("to")),
	}
	if err := validateListFilter(filter); err != nil {
		writeHandledError(w, err)
		return
	}

	music, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list music entries", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"music": music,
	})
}

func (h *Handler) createMusic(w http.ResponseWriter, r *http.Request) {
	var input CreateMusicRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid music payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	music := musicFromCreate(input)
	created, err := h.store.Create(r.Context(), music)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/music/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getMusic(w http.ResponseWriter, r *http.Request, id string) {
	music, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, music)
}

func (h *Handler) updateMusic(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateMusicRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid music payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), id, func(music *Music) error {
		applyUpdate(music, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteMusic(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func musicFromCreate(input CreateMusicRequest) Music {
	return Music{
		Title:            strings.TrimSpace(input.Title),
		Artist:           strings.TrimSpace(input.Artist),
		Album:            strings.TrimSpace(input.Album),
		SpotifyURL:       normalizeURL(input.SpotifyURL),
		YouTubeURL:       normalizeURL(input.YouTubeURL),
		MostlyListenedOn: normalizeDate(input.MostlyListenedOn),
		Notes:            strings.TrimSpace(input.Notes),
		SortOrder:        input.SortOrder,
		Status:           normalizeStatus(input.Status),
	}
}

func applyUpdate(music *Music, input UpdateMusicRequest) {
	if input.Title != nil {
		music.Title = strings.TrimSpace(*input.Title)
	}
	if input.Artist != nil {
		music.Artist = strings.TrimSpace(*input.Artist)
	}
	if input.Album != nil {
		music.Album = strings.TrimSpace(*input.Album)
	}
	if input.SpotifyURL != nil {
		music.SpotifyURL = normalizeURL(*input.SpotifyURL)
	}
	if input.YouTubeURL != nil {
		music.YouTubeURL = normalizeURL(*input.YouTubeURL)
	}
	if input.MostlyListenedOn != nil {
		music.MostlyListenedOn = normalizeDate(*input.MostlyListenedOn)
	}
	if input.Notes != nil {
		music.Notes = strings.TrimSpace(*input.Notes)
	}
	if input.SortOrder != nil {
		music.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		music.Status = normalizeStatus(*input.Status)
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
		writeError(w, http.StatusNotFound, "music entry not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "music entry already exists", nil)
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

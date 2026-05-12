package games

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
		h.listGames(w, r)
	case http.MethodPost:
		h.createGame(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/games/"), "/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "game entry not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getGame(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateGame(w, r, id)
	case http.MethodDelete:
		h.deleteGame(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listGames(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = StatusPublished
	} else {
		status = normalizeStatus(status)
	}

	filter := ListFilter{
		Status:   status,
		Platform: strings.TrimSpace(r.URL.Query().Get("platform")),
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
		From:     strings.TrimSpace(r.URL.Query().Get("from")),
		To:       strings.TrimSpace(r.URL.Query().Get("to")),
	}
	if err := validateListFilter(filter); err != nil {
		writeHandledError(w, err)
		return
	}

	games, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list game entries", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"games": games,
	})
}

func (h *Handler) createGame(w http.ResponseWriter, r *http.Request) {
	var input CreateGameRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid game payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	game := gameFromCreate(input)
	created, err := h.store.Create(r.Context(), game)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/games/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getGame(w http.ResponseWriter, r *http.Request, id string) {
	game, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, game)
}

func (h *Handler) updateGame(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateGameRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid game payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), id, func(game *Game) error {
		applyUpdate(game, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteGame(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func gameFromCreate(input CreateGameRequest) Game {
	return Game{
		Title:          strings.TrimSpace(input.Title),
		Studio:         strings.TrimSpace(input.Studio),
		Platform:       strings.TrimSpace(input.Platform),
		Genre:          strings.TrimSpace(input.Genre),
		StoreURL:       normalizeURL(input.StoreURL),
		MostlyPlayedOn: normalizeDate(input.MostlyPlayedOn),
		Notes:          strings.TrimSpace(input.Notes),
		SortOrder:      input.SortOrder,
		Status:         normalizeStatus(input.Status),
	}
}

func applyUpdate(game *Game, input UpdateGameRequest) {
	if input.Title != nil {
		game.Title = strings.TrimSpace(*input.Title)
	}
	if input.Studio != nil {
		game.Studio = strings.TrimSpace(*input.Studio)
	}
	if input.Platform != nil {
		game.Platform = strings.TrimSpace(*input.Platform)
	}
	if input.Genre != nil {
		game.Genre = strings.TrimSpace(*input.Genre)
	}
	if input.StoreURL != nil {
		game.StoreURL = normalizeURL(*input.StoreURL)
	}
	if input.MostlyPlayedOn != nil {
		game.MostlyPlayedOn = normalizeDate(*input.MostlyPlayedOn)
	}
	if input.Notes != nil {
		game.Notes = strings.TrimSpace(*input.Notes)
	}
	if input.SortOrder != nil {
		game.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		game.Status = normalizeStatus(*input.Status)
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
		writeError(w, http.StatusNotFound, "game entry not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "game entry already exists", nil)
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

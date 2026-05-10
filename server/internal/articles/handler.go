package articles

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
		h.listArticles(w, r)
	case http.MethodPost:
		h.createArticle(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/articles/"), "/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "article not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getArticle(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateArticle(w, r, id)
	case http.MethodDelete:
		h.deleteArticle(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listArticles(w http.ResponseWriter, r *http.Request) {
	var featured *bool
	if rawFeatured := r.URL.Query().Get("featured"); rawFeatured != "" {
		parsed, err := strconv.ParseBool(rawFeatured)
		if err != nil {
			writeError(w, http.StatusBadRequest, "featured must be true or false", nil)
			return
		}
		featured = &parsed
	}

	articles, err := h.store.List(r.Context(), ListFilter{
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Featured: featured,
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list articles", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"articles": articles,
	})
}

func (h *Handler) createArticle(w http.ResponseWriter, r *http.Request) {
	var input CreateArticleRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid article payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	article := articleFromCreate(input)
	created, err := h.store.Create(r.Context(), article)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/articles/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getArticle(w http.ResponseWriter, r *http.Request, id string) {
	article, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, article)
}

func (h *Handler) updateArticle(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateArticleRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid article payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), id, func(article *Article) error {
		applyUpdate(article, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteArticle(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func articleFromCreate(input CreateArticleRequest) Article {
	return Article{
		Title:         strings.TrimSpace(input.Title),
		URL:           normalizeURL(input.URL),
		Source:        strings.TrimSpace(input.Source),
		Summary:       strings.TrimSpace(input.Summary),
		CoverImageURL: normalizeURL(input.CoverImageURL),
		Featured:      input.Featured,
		SortOrder:     input.SortOrder,
		Status:        normalizeStatus(input.Status),
		PublishedAt:   utcTime(input.PublishedAt),
	}
}

func applyUpdate(article *Article, input UpdateArticleRequest) {
	if input.Title != nil {
		article.Title = strings.TrimSpace(*input.Title)
	}
	if input.URL != nil {
		article.URL = normalizeURL(*input.URL)
	}
	if input.Source != nil {
		article.Source = strings.TrimSpace(*input.Source)
	}
	if input.Summary != nil {
		article.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.CoverImageURL != nil {
		article.CoverImageURL = normalizeURL(*input.CoverImageURL)
	}
	if input.Featured != nil {
		article.Featured = *input.Featured
	}
	if input.SortOrder != nil {
		article.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		article.Status = normalizeStatus(*input.Status)
	}
	if input.PublishedAt != nil {
		article.PublishedAt = utcTime(input.PublishedAt)
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
		writeError(w, http.StatusNotFound, "article not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "article url or id already exists", nil)
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

package products

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/eralme/server/internal/auth"
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
		h.listProducts(w, r)
	case http.MethodPost:
		h.createProduct(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	idOrSlug := strings.Trim(strings.TrimPrefix(r.URL.Path, "/products/"), "/")
	if idOrSlug == "" || strings.Contains(idOrSlug, "/") {
		writeError(w, http.StatusNotFound, "product not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getProduct(w, r, idOrSlug)
	case http.MethodPut, http.MethodPatch:
		h.updateProduct(w, r, idOrSlug)
	case http.MethodDelete:
		h.deleteProduct(w, r, idOrSlug)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleSection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getSection(w, r)
	case http.MethodPatch:
		h.updateSection(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	status := productStatusFilter(r)

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

	products, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list products", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"products": products,
	})
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	var input CreateProductRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid product payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	product := productFromCreate(input)
	created, err := h.store.Create(r.Context(), product)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/products/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	product, err := h.store.Get(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}
	if !isAdminRequest(r) && product.Status != StatusPublished {
		writeHandledError(w, ErrNotFound)
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	var input UpdateProductRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid product payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), idOrSlug, func(product *Product) error {
		applyUpdate(product, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	if err := h.store.Delete(r.Context(), idOrSlug); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getSection(w http.ResponseWriter, r *http.Request) {
	section, err := h.store.GetSection(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load products section", nil)
		return
	}

	writeJSON(w, http.StatusOK, section)
}

func (h *Handler) updateSection(w http.ResponseWriter, r *http.Request) {
	var input UpdateProductSectionRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid products section payload", nil)
		return
	}
	if err := validateSectionUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	section, err := h.store.UpdateSection(r.Context(), func(settings *ProductSectionSettings) error {
		if input.Enabled != nil {
			settings.Enabled = *input.Enabled
		}
		if input.Title != nil {
			settings.Title = strings.TrimSpace(*input.Title)
		}
		if input.Description != nil {
			settings.Description = strings.TrimSpace(*input.Description)
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update products section", nil)
		return
	}

	writeJSON(w, http.StatusOK, section)
}

func productFromCreate(input CreateProductRequest) Product {
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugify(input.Title)
	}
	status := normalizeStatus(input.Status)

	return Product{
		Slug:          slug,
		Title:         strings.TrimSpace(input.Title),
		Summary:       strings.TrimSpace(input.Summary),
		Description:   strings.TrimSpace(input.Description),
		CoverImageURL: normalizeURL(input.CoverImageURL),
		PriceLabel:    strings.TrimSpace(input.PriceLabel),
		CTALabel:      strings.TrimSpace(input.CTALabel),
		CTAURL:        normalizeURL(input.CTAURL),
		Category:      strings.TrimSpace(input.Category),
		Tags:          normalizeStringSlice(input.Tags),
		Featured:      input.Featured,
		SortOrder:     input.SortOrder,
		Status:        status,
		PublishedAt:   publishTime(status, input.PublishedAt),
	}
}

func applyUpdate(product *Product, input UpdateProductRequest) {
	if input.Slug != nil {
		product.Slug = strings.TrimSpace(*input.Slug)
		if product.Slug == "" {
			product.Slug = slugify(product.Title)
		}
	}
	if input.Title != nil {
		product.Title = strings.TrimSpace(*input.Title)
		if product.Slug == "" {
			product.Slug = slugify(product.Title)
		}
	}
	if input.Summary != nil {
		product.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.Description != nil {
		product.Description = strings.TrimSpace(*input.Description)
	}
	if input.CoverImageURL != nil {
		product.CoverImageURL = normalizeURL(*input.CoverImageURL)
	}
	if input.PriceLabel != nil {
		product.PriceLabel = strings.TrimSpace(*input.PriceLabel)
	}
	if input.CTALabel != nil {
		product.CTALabel = strings.TrimSpace(*input.CTALabel)
	}
	if input.CTAURL != nil {
		product.CTAURL = normalizeURL(*input.CTAURL)
	}
	if input.Category != nil {
		product.Category = strings.TrimSpace(*input.Category)
	}
	if input.Tags != nil {
		product.Tags = normalizeStringSlice(*input.Tags)
	}
	if input.Featured != nil {
		product.Featured = *input.Featured
	}
	if input.SortOrder != nil {
		product.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		product.Status = normalizeStatus(*input.Status)
		if product.Status == StatusPublished && product.PublishedAt == nil {
			product.PublishedAt = publishTime(product.Status, nil)
		}
	}
	if input.PublishedAt != nil {
		product.PublishedAt = publishTime(product.Status, input.PublishedAt)
	}
}

func productStatusFilter(r *http.Request) string {
	if !isAdminRequest(r) {
		return StatusPublished
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if strings.EqualFold(status, "all") {
		return ""
	}
	if status == "" {
		return ""
	}
	return normalizeStatus(status)
}

func isAdminRequest(r *http.Request) bool {
	_, ok := auth.ClaimsFromContext(r.Context())
	return ok
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
		writeError(w, http.StatusNotFound, "product not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "product slug or id already exists", nil)
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

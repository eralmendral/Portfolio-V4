package skills

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

func (h *Handler) HandleCategoryCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listCategories(w, r)
	case http.MethodPost:
		h.createCategory(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleCategoryItem(w http.ResponseWriter, r *http.Request) {
	idOrSlug := strings.Trim(strings.TrimPrefix(r.URL.Path, "/skill-categories/"), "/")
	if idOrSlug == "" || strings.Contains(idOrSlug, "/") {
		writeError(w, http.StatusNotFound, "skill category not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getCategory(w, r, idOrSlug)
	case http.MethodPut, http.MethodPatch:
		h.updateCategory(w, r, idOrSlug)
	case http.MethodDelete:
		h.deleteCategory(w, r, idOrSlug)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleSkillCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listSkills(w, r)
	case http.MethodPost:
		h.createSkill(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleSkillItem(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/skills/"), "/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "skill not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getSkill(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateSkill(w, r, id)
	case http.MethodDelete:
		h.deleteSkill(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = StatusPublished
	} else {
		status = normalizeStatus(status)
	}

	filter := CategoryListFilter{
		Status: status,
		Query:  strings.TrimSpace(r.URL.Query().Get("q")),
	}
	if err := validateCategoryListFilter(filter); err != nil {
		writeHandledError(w, err)
		return
	}

	categories, err := h.store.ListCategories(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list skill categories", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"skill_categories": categories,
	})
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var input CreateSkillCategoryRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid skill category payload", nil)
		return
	}
	if err := validateCreateCategory(input); err != nil {
		writeHandledError(w, err)
		return
	}

	category := categoryFromCreate(input)
	created, err := h.store.CreateCategory(r.Context(), category)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/skill-categories/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	category, err := h.store.GetCategory(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, category)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	var input UpdateSkillCategoryRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid skill category payload", nil)
		return
	}
	if err := validateUpdateCategory(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.UpdateCategory(r.Context(), idOrSlug, func(category *SkillCategory) error {
		applyCategoryUpdate(category, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	if err := h.store.DeleteCategory(r.Context(), idOrSlug); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listSkills(w http.ResponseWriter, r *http.Request) {
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

	filter := SkillListFilter{
		Status:     status,
		Featured:   featured,
		CategoryID: strings.TrimSpace(r.URL.Query().Get("category_id")),
		Category:   strings.TrimSpace(r.URL.Query().Get("category")),
		Query:      strings.TrimSpace(r.URL.Query().Get("q")),
	}
	if err := validateSkillListFilter(filter); err != nil {
		writeHandledError(w, err)
		return
	}

	skills, err := h.store.ListSkills(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list skills", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"skills": skills,
	})
}

func (h *Handler) createSkill(w http.ResponseWriter, r *http.Request) {
	var input CreateSkillRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid skill payload", nil)
		return
	}
	if err := validateCreateSkill(input); err != nil {
		writeHandledError(w, err)
		return
	}

	skill := skillFromCreate(input)
	created, err := h.store.CreateSkill(r.Context(), skill)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/skills/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getSkill(w http.ResponseWriter, r *http.Request, id string) {
	skill, err := h.store.GetSkill(r.Context(), id)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, skill)
}

func (h *Handler) updateSkill(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateSkillRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid skill payload", nil)
		return
	}
	if err := validateUpdateSkill(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.UpdateSkill(r.Context(), id, func(skill *Skill) error {
		applySkillUpdate(skill, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteSkill(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteSkill(r.Context(), id); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func categoryFromCreate(input CreateSkillCategoryRequest) SkillCategory {
	status := normalizeStatus(input.Status)
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugify(input.Name)
	}

	return SkillCategory{
		Slug:        slug,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		IconClass:   strings.TrimSpace(input.IconClass),
		SortOrder:   input.SortOrder,
		Status:      status,
	}
}

func applyCategoryUpdate(category *SkillCategory, input UpdateSkillCategoryRequest) {
	if input.Slug != nil {
		category.Slug = strings.TrimSpace(*input.Slug)
		if category.Slug == "" {
			category.Slug = slugify(category.Name)
		}
	}
	if input.Name != nil {
		category.Name = strings.TrimSpace(*input.Name)
		if category.Slug == "" {
			category.Slug = slugify(category.Name)
		}
	}
	if input.Description != nil {
		category.Description = strings.TrimSpace(*input.Description)
	}
	if input.IconClass != nil {
		category.IconClass = strings.TrimSpace(*input.IconClass)
	}
	if input.SortOrder != nil {
		category.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		category.Status = normalizeStatus(*input.Status)
	}
}

func skillFromCreate(input CreateSkillRequest) Skill {
	return Skill{
		CategoryID: strings.TrimSpace(input.CategoryID),
		Name:       strings.TrimSpace(input.Name),
		Summary:    strings.TrimSpace(input.Summary),
		IconClass:  strings.TrimSpace(input.IconClass),
		SortOrder:  input.SortOrder,
		Featured:   input.Featured,
		Status:     normalizeStatus(input.Status),
	}
}

func applySkillUpdate(skill *Skill, input UpdateSkillRequest) {
	if input.CategoryID != nil {
		skill.CategoryID = strings.TrimSpace(*input.CategoryID)
	}
	if input.Name != nil {
		skill.Name = strings.TrimSpace(*input.Name)
	}
	if input.Summary != nil {
		skill.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.IconClass != nil {
		skill.IconClass = strings.TrimSpace(*input.IconClass)
	}
	if input.SortOrder != nil {
		skill.SortOrder = *input.SortOrder
	}
	if input.Featured != nil {
		skill.Featured = *input.Featured
	}
	if input.Status != nil {
		skill.Status = normalizeStatus(*input.Status)
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
		writeError(w, http.StatusNotFound, "skill resource not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "skill resource already exists", nil)
		return
	}
	if errors.Is(err, ErrInvalidCategory) {
		writeError(w, http.StatusBadRequest, "invalid skill category", map[string]string{
			"category_id": "category_id must reference an existing skill category",
		})
		return
	}
	if errors.Is(err, ErrCategoryInUse) {
		writeError(w, http.StatusConflict, "skill category has skills", nil)
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

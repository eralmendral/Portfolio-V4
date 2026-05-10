package workexperience

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
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
		h.listWorkExperiences(w, r)
	case http.MethodPost:
		h.createWorkExperience(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	idOrSlug := strings.Trim(strings.TrimPrefix(r.URL.Path, "/work-experiences/"), "/")
	if idOrSlug == "" || strings.Contains(idOrSlug, "/") {
		writeError(w, http.StatusNotFound, "work experience not found", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getWorkExperience(w, r, idOrSlug)
	case http.MethodPut, http.MethodPatch:
		h.updateWorkExperience(w, r, idOrSlug)
	case http.MethodDelete:
		h.deleteWorkExperience(w, r, idOrSlug)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) listWorkExperiences(w http.ResponseWriter, r *http.Request) {
	var featured *bool
	if rawFeatured := r.URL.Query().Get("featured"); rawFeatured != "" {
		parsed, err := strconv.ParseBool(rawFeatured)
		if err != nil {
			writeError(w, http.StatusBadRequest, "featured must be true or false", nil)
			return
		}
		featured = &parsed
	}

	var current *bool
	if rawCurrent := r.URL.Query().Get("current"); rawCurrent != "" {
		parsed, err := strconv.ParseBool(rawCurrent)
		if err != nil {
			writeError(w, http.StatusBadRequest, "current must be true or false", nil)
			return
		}
		current = &parsed
	}

	workExperiences, err := h.store.List(r.Context(), ListFilter{
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Featured: featured,
		Current:  current,
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list work experiences", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"work_experiences": workExperiences,
	})
}

func (h *Handler) createWorkExperience(w http.ResponseWriter, r *http.Request) {
	var input CreateWorkExperienceRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid work experience payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	workExperience := workExperienceFromCreate(input)
	created, err := h.store.Create(r.Context(), workExperience)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/work-experiences/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getWorkExperience(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	workExperience, err := h.store.Get(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, workExperience)
}

func (h *Handler) updateWorkExperience(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	var input UpdateWorkExperienceRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid work experience payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), idOrSlug, func(workExperience *WorkExperience) error {
		applyUpdate(workExperience, input)
		return validateRecord(*workExperience)
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteWorkExperience(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	if err := h.store.Delete(r.Context(), idOrSlug); err != nil {
		writeHandledError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func workExperienceFromCreate(input CreateWorkExperienceRequest) WorkExperience {
	status := normalizeStatus(input.Status)
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugify(strings.TrimSpace(input.Company) + " " + strings.TrimSpace(input.Title))
	}

	var startedAt time.Time
	if input.StartedAt != nil {
		startedAt = input.StartedAt.UTC()
	}
	endedAt := utcTime(input.EndedAt)
	current := input.Current
	if current {
		endedAt = nil
	}
	if endedAt != nil {
		current = false
	}

	return WorkExperience{
		Slug:             slug,
		Title:            strings.TrimSpace(input.Title),
		Company:          strings.TrimSpace(input.Company),
		CompanyURL:       normalizeURL(input.CompanyURL),
		CompanyLogoURL:   normalizeURL(input.CompanyLogoURL),
		EmploymentType:   strings.TrimSpace(input.EmploymentType),
		Location:         strings.TrimSpace(input.Location),
		LocationType:     strings.TrimSpace(input.LocationType),
		Summary:          strings.TrimSpace(input.Summary),
		Description:      strings.TrimSpace(input.Description),
		Highlights:       normalizeStringSlice(input.Highlights),
		Responsibilities: normalizeStringSlice(input.Responsibilities),
		TechStack:        normalizeStringSlice(input.TechStack),
		Skills:           normalizeStringSlice(input.Skills),
		StartedAt:        startedAt,
		EndedAt:          endedAt,
		Current:          current,
		Featured:         input.Featured,
		SortOrder:        input.SortOrder,
		Status:           status,
		PublishedAt:      publishTime(status, input.PublishedAt),
	}
}

func applyUpdate(workExperience *WorkExperience, input UpdateWorkExperienceRequest) {
	if input.Slug != nil {
		workExperience.Slug = strings.TrimSpace(*input.Slug)
		if workExperience.Slug == "" {
			workExperience.Slug = slugify(workExperience.Company + " " + workExperience.Title)
		}
	}
	if input.Title != nil {
		workExperience.Title = strings.TrimSpace(*input.Title)
		if workExperience.Slug == "" {
			workExperience.Slug = slugify(workExperience.Company + " " + workExperience.Title)
		}
	}
	if input.Company != nil {
		workExperience.Company = strings.TrimSpace(*input.Company)
		if workExperience.Slug == "" {
			workExperience.Slug = slugify(workExperience.Company + " " + workExperience.Title)
		}
	}
	if input.CompanyURL != nil {
		workExperience.CompanyURL = normalizeURL(*input.CompanyURL)
	}
	if input.CompanyLogoURL != nil {
		workExperience.CompanyLogoURL = normalizeURL(*input.CompanyLogoURL)
	}
	if input.EmploymentType != nil {
		workExperience.EmploymentType = strings.TrimSpace(*input.EmploymentType)
	}
	if input.Location != nil {
		workExperience.Location = strings.TrimSpace(*input.Location)
	}
	if input.LocationType != nil {
		workExperience.LocationType = strings.TrimSpace(*input.LocationType)
	}
	if input.Summary != nil {
		workExperience.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.Description != nil {
		workExperience.Description = strings.TrimSpace(*input.Description)
	}
	if input.Highlights != nil {
		workExperience.Highlights = normalizeStringSlice(*input.Highlights)
	}
	if input.Responsibilities != nil {
		workExperience.Responsibilities = normalizeStringSlice(*input.Responsibilities)
	}
	if input.TechStack != nil {
		workExperience.TechStack = normalizeStringSlice(*input.TechStack)
	}
	if input.Skills != nil {
		workExperience.Skills = normalizeStringSlice(*input.Skills)
	}
	if input.StartedAt != nil {
		workExperience.StartedAt = input.StartedAt.UTC()
	}
	if input.EndedAt != nil {
		endedAt := input.EndedAt.UTC()
		workExperience.EndedAt = &endedAt
		workExperience.Current = false
	}
	if input.Current != nil {
		workExperience.Current = *input.Current
		if *input.Current {
			workExperience.EndedAt = nil
		}
	}
	if input.Featured != nil {
		workExperience.Featured = *input.Featured
	}
	if input.SortOrder != nil {
		workExperience.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		workExperience.Status = normalizeStatus(*input.Status)
	}
	if input.PublishedAt != nil {
		published := input.PublishedAt.UTC()
		workExperience.PublishedAt = &published
	} else if workExperience.Status == StatusPublished && workExperience.PublishedAt == nil {
		now := time.Now().UTC()
		workExperience.PublishedAt = &now
	} else if workExperience.Status != StatusPublished {
		workExperience.PublishedAt = nil
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
		writeError(w, http.StatusNotFound, "work experience not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "work experience slug or id already exists", nil)
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

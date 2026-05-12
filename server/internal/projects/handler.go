package projects

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	store   Store
	uploads UploadConfig
}

func NewHandler(store Store, uploads UploadConfig) *Handler {
	return &Handler{
		store:   store,
		uploads: uploads.withDefaults(),
	}
}

func (h *Handler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listProjects(w, r)
	case http.MethodPost:
		h.createProject(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
	}
}

func (h *Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/projects/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "project not found", nil)
		return
	}

	idOrSlug := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.getProject(w, r, idOrSlug)
		case http.MethodPut, http.MethodPatch:
			h.updateProject(w, r, idOrSlug)
		case http.MethodDelete:
			h.deleteProject(w, r, idOrSlug)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
		}
		return
	}

	if len(parts) == 2 && parts[1] == "images" && r.Method == http.MethodPost {
		h.uploadGalleryImages(w, r, idOrSlug)
		return
	}

	if len(parts) == 3 && parts[1] == "images" && parts[2] == "main" && r.Method == http.MethodPost {
		h.uploadMainImage(w, r, idOrSlug)
		return
	}

	if len(parts) == 3 && parts[1] == "images" && r.Method == http.MethodDelete {
		h.deleteProjectImage(w, r, idOrSlug, parts[2])
		return
	}

	writeError(w, http.StatusNotFound, "route not found", nil)
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	var featured *bool
	if rawFeatured := r.URL.Query().Get("featured"); rawFeatured != "" {
		parsed, err := strconv.ParseBool(rawFeatured)
		if err != nil {
			writeError(w, http.StatusBadRequest, "featured must be true or false", nil)
			return
		}
		featured = &parsed
	}

	var archived *bool
	if rawArchived := r.URL.Query().Get("archived"); rawArchived != "" {
		parsed, err := strconv.ParseBool(rawArchived)
		if err != nil {
			writeError(w, http.StatusBadRequest, "archived must be true or false", nil)
			return
		}
		archived = &parsed
	}

	projects, err := h.store.List(r.Context(), ListFilter{
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Featured: featured,
		Archived: archived,
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list projects", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projects": projects,
	})
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var input CreateProjectRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid project payload", nil)
		return
	}
	if err := validateCreate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	project := projectFromCreate(input)
	created, err := h.store.Create(r.Context(), project)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	w.Header().Set("Location", "/projects/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getProject(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	project, err := h.store.Get(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, project)
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	var input UpdateProjectRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid project payload", nil)
		return
	}
	if err := validateUpdate(input); err != nil {
		writeHandledError(w, err)
		return
	}

	updated, err := h.store.Update(r.Context(), idOrSlug, func(project *Project) error {
		applyUpdate(project, input)
		return nil
	})
	if err != nil {
		writeHandledError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request, idOrSlug string) {
	project, err := h.store.Get(r.Context(), idOrSlug)
	if err != nil {
		writeHandledError(w, err)
		return
	}

	if err := h.store.Delete(r.Context(), idOrSlug); err != nil {
		writeHandledError(w, err)
		return
	}

	h.removeProjectFiles(project)
	w.WriteHeader(http.StatusNoContent)
}

func projectFromCreate(input CreateProjectRequest) Project {
	status := normalizeStatus(input.Status)
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugify(input.Title)
	}

	project := Project{
		Slug:        slug,
		Title:       strings.TrimSpace(input.Title),
		Summary:     strings.TrimSpace(input.Summary),
		Description: strings.TrimSpace(input.Description),
		Body:        strings.TrimSpace(input.Body),
		TechStack:   normalizeStringSlice(input.TechStack),
		Tags:        normalizeStringSlice(input.Tags),
		GitHubURL:   normalizeURL(input.GitHubURL),
		DemoURL:     normalizeURL(input.DemoURL),
		Featured:    input.Featured,
		Archived:    input.Archived,
		SortOrder:   input.SortOrder,
		Status:      status,
		PublishedAt: publishTime(status, input.PublishedAt),
	}

	if input.MainImage != nil {
		mainImage := imageFromInput(*input.MainImage)
		project.MainImage = &mainImage
	}

	for _, inputImage := range input.Images {
		project.Images = append(project.Images, imageFromInput(inputImage))
	}

	return project
}

func applyUpdate(project *Project, input UpdateProjectRequest) {
	if input.Slug != nil {
		project.Slug = strings.TrimSpace(*input.Slug)
		if project.Slug == "" {
			project.Slug = slugify(project.Title)
		}
	}
	if input.Title != nil {
		project.Title = strings.TrimSpace(*input.Title)
		if project.Slug == "" {
			project.Slug = slugify(project.Title)
		}
	}
	if input.Summary != nil {
		project.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.Description != nil {
		project.Description = strings.TrimSpace(*input.Description)
	}
	if input.Body != nil {
		project.Body = strings.TrimSpace(*input.Body)
	}
	if input.TechStack != nil {
		project.TechStack = normalizeStringSlice(*input.TechStack)
	}
	if input.Tags != nil {
		project.Tags = normalizeStringSlice(*input.Tags)
	}
	if input.MainImage != nil {
		mainImage := imageFromInput(*input.MainImage)
		project.MainImage = &mainImage
	}
	if input.Images != nil {
		project.Images = make([]ProjectImage, 0, len(*input.Images))
		for _, inputImage := range *input.Images {
			project.Images = append(project.Images, imageFromInput(inputImage))
		}
	}
	if input.GitHubURL != nil {
		project.GitHubURL = normalizeURL(*input.GitHubURL)
	}
	if input.DemoURL != nil {
		project.DemoURL = normalizeURL(*input.DemoURL)
	}
	if input.Featured != nil {
		project.Featured = *input.Featured
	}
	if input.Archived != nil {
		project.Archived = *input.Archived
	}
	if input.SortOrder != nil {
		project.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		project.Status = normalizeStatus(*input.Status)
	}
	if input.PublishedAt != nil {
		published := input.PublishedAt.UTC()
		project.PublishedAt = &published
	} else if project.Status == StatusPublished && project.PublishedAt == nil {
		now := time.Now().UTC()
		project.PublishedAt = &now
	} else if project.Status != StatusPublished {
		project.PublishedAt = nil
	}
}

func imageFromInput(input ProjectImageInput) ProjectImage {
	return ProjectImage{
		ID:         newID(),
		URL:        strings.TrimSpace(input.URL),
		AltText:    strings.TrimSpace(input.AltText),
		Caption:    strings.TrimSpace(input.Caption),
		SortOrder:  input.SortOrder,
		UploadedAt: time.Now().UTC(),
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
		writeError(w, http.StatusNotFound, "project not found", nil)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, http.StatusConflict, "project slug or id already exists", nil)
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

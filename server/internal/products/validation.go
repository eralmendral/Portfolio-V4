package products

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type validationError struct {
	fields map[string]string
}

func (e validationError) Error() string {
	return "validation failed"
}

func (e validationError) Fields() map[string]string {
	return e.fields
}

func validateCreate(input CreateProductRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.Title) == "" {
		fields["title"] = "title is required"
	}
	validateSlug(input.Slug, fields)
	if strings.TrimSpace(input.CTAURL) == "" {
		fields["cta_url"] = "cta url is required"
	} else {
		validateURL("cta_url", input.CTAURL, fields)
	}
	validateURL("cover_image_url", input.CoverImageURL, fields)
	validateStatus(input.Status, fields)
	validateSortOrder("sort_order", input.SortOrder, fields)

	return fieldsError(fields)
}

func validateUpdate(input UpdateProductRequest) error {
	fields := make(map[string]string)

	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		fields["title"] = "title cannot be blank"
	}
	if input.Slug != nil {
		validateSlug(*input.Slug, fields)
	}
	if input.CTAURL != nil {
		if strings.TrimSpace(*input.CTAURL) == "" {
			fields["cta_url"] = "cta url cannot be blank"
		} else {
			validateURL("cta_url", *input.CTAURL, fields)
		}
	}
	if input.CoverImageURL != nil {
		validateURL("cover_image_url", *input.CoverImageURL, fields)
	}
	if input.Status != nil {
		validateStatus(*input.Status, fields)
	}
	if input.SortOrder != nil {
		validateSortOrder("sort_order", *input.SortOrder, fields)
	}

	return fieldsError(fields)
}

func validateListFilter(filter ListFilter) error {
	fields := make(map[string]string)
	validateStatus(filter.Status, fields)
	return fieldsError(fields)
}

func validateSectionUpdate(input UpdateProductSectionRequest) error {
	fields := make(map[string]string)
	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		fields["title"] = "title cannot be blank"
	}
	return fieldsError(fields)
}

func validateSortOrder(field string, sortOrder int, fields map[string]string) {
	if sortOrder < 0 {
		fields[field] = "sort order must be zero or greater"
	}
}

func validateSlug(slug string, fields map[string]string) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return
	}
	if slug == "section" {
		fields["slug"] = "slug is reserved"
		return
	}
	if !slugPattern.MatchString(slug) {
		fields["slug"] = "slug must use lowercase letters, numbers, and hyphens"
	}
}

func validateStatus(status string, fields map[string]string) {
	if strings.TrimSpace(status) == "" {
		return
	}

	switch normalizeStatus(status) {
	case StatusDraft, StatusPublished, StatusArchived:
	default:
		fields["status"] = "status must be draft, published, or archived"
	}
}

func validateURL(field string, value string, fields map[string]string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		fields[field] = "must be a valid http or https URL"
	}
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return StatusDraft
	}
	return status
}

func normalizeURL(value string) string {
	return strings.TrimSpace(value)
}

func publishTime(status string, provided *time.Time) *time.Time {
	if provided != nil {
		published := provided.UTC()
		return &published
	}
	if status == StatusPublished {
		now := time.Now().UTC()
		return &now
	}
	return nil
}

func fieldsError(fields map[string]string) error {
	if len(fields) == 0 {
		return nil
	}
	return validationError{fields: fields}
}

func isValidationError(err error) (validationError, bool) {
	var validation validationError
	if errors.As(err, &validation) {
		return validation, true
	}
	return validationError{}, false
}

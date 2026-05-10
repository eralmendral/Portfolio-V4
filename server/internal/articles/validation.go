package articles

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

type validationError struct {
	fields map[string]string
}

func (e validationError) Error() string {
	return "validation failed"
}

func (e validationError) Fields() map[string]string {
	return e.fields
}

func validateCreate(input CreateArticleRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.Title) == "" {
		fields["title"] = "title is required"
	}
	if strings.TrimSpace(input.URL) == "" {
		fields["url"] = "url is required"
	} else {
		validateURL("url", input.URL, fields)
	}
	if strings.TrimSpace(input.Source) == "" {
		fields["source"] = "source is required"
	}
	validateURL("cover_image_url", input.CoverImageURL, fields)
	validateStatus(input.Status, fields)
	validateSortOrder("sort_order", input.SortOrder, fields)

	return fieldsError(fields)
}

func validateUpdate(input UpdateArticleRequest) error {
	fields := make(map[string]string)

	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		fields["title"] = "title cannot be blank"
	}
	if input.URL != nil {
		if strings.TrimSpace(*input.URL) == "" {
			fields["url"] = "url cannot be blank"
		} else {
			validateURL("url", *input.URL, fields)
		}
	}
	if input.Source != nil && strings.TrimSpace(*input.Source) == "" {
		fields["source"] = "source cannot be blank"
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

func validateStatus(status string, fields map[string]string) {
	if status == "" {
		return
	}

	switch status {
	case StatusDraft, StatusPublished, StatusArchived:
	default:
		fields["status"] = "status must be draft, published, or archived"
	}
}

func validateSortOrder(field string, sortOrder int, fields map[string]string) {
	if sortOrder < 0 {
		fields[field] = "sort order must be zero or greater"
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

func utcTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	converted := value.UTC()
	return &converted
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

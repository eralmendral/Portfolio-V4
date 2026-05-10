package links

import (
	"errors"
	"net/url"
	"strings"
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

func validateCreate(input CreateLinkRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.Label) == "" {
		fields["label"] = "label is required"
	}
	if strings.TrimSpace(input.URL) == "" {
		fields["url"] = "url is required"
	} else {
		validateURL("url", input.URL, fields)
	}
	validateSortOrder("sort_order", input.SortOrder, fields)
	validateStatus(input.Status, fields)

	return fieldsError(fields)
}

func validateUpdate(input UpdateLinkRequest) error {
	fields := make(map[string]string)

	if input.Label != nil && strings.TrimSpace(*input.Label) == "" {
		fields["label"] = "label cannot be blank"
	}
	if input.URL != nil {
		if strings.TrimSpace(*input.URL) == "" {
			fields["url"] = "url cannot be blank"
		} else {
			validateURL("url", *input.URL, fields)
		}
	}
	if input.SortOrder != nil {
		validateSortOrder("sort_order", *input.SortOrder, fields)
	}
	if input.Status != nil {
		validateStatus(*input.Status, fields)
	}

	return fieldsError(fields)
}

func validateListFilter(filter ListFilter) error {
	fields := make(map[string]string)
	validateStatus(filter.Status, fields)
	return fieldsError(fields)
}

func validateSortOrder(field string, sortOrder int, fields map[string]string) {
	if sortOrder < 0 {
		fields[field] = "sort order must be zero or greater"
	}
}

func validateStatus(status string, fields map[string]string) {
	if status == "" {
		return
	}

	switch normalizeStatus(status) {
	case StatusDraft, StatusPublished, StatusArchived:
	default:
		fields["status"] = "status must be draft, published, or archived"
	}
}

func validateURL(field string, value string, fields map[string]string) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
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

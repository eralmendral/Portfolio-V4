package tools

import (
	"errors"
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

func validateCreate(input CreateToolRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.Name) == "" {
		fields["name"] = "name is required"
	}
	if strings.TrimSpace(input.Category) == "" {
		fields["category"] = "category is required"
	}
	validateSortOrder("sort_order", input.SortOrder, fields)
	validateStatus(input.Status, fields)

	return fieldsError(fields)
}

func validateUpdate(input UpdateToolRequest) error {
	fields := make(map[string]string)

	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		fields["name"] = "name cannot be blank"
	}
	if input.Category != nil && strings.TrimSpace(*input.Category) == "" {
		fields["category"] = "category cannot be blank"
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

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return StatusDraft
	}
	return status
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

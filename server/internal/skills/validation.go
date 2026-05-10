package skills

import (
	"errors"
	"regexp"
	"strings"
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

func validateCreateCategory(input CreateSkillCategoryRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.Name) == "" {
		fields["name"] = "name is required"
	}
	validateSlug(input.Slug, fields)
	validateStatus(input.Status, fields)
	validateSortOrder("sort_order", input.SortOrder, fields)

	return fieldsError(fields)
}

func validateUpdateCategory(input UpdateSkillCategoryRequest) error {
	fields := make(map[string]string)

	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		fields["name"] = "name cannot be blank"
	}
	if input.Slug != nil {
		validateSlug(*input.Slug, fields)
	}
	if input.Status != nil {
		validateStatus(*input.Status, fields)
	}
	if input.SortOrder != nil {
		validateSortOrder("sort_order", *input.SortOrder, fields)
	}

	return fieldsError(fields)
}

func validateCreateSkill(input CreateSkillRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.CategoryID) == "" {
		fields["category_id"] = "category_id is required"
	}
	if strings.TrimSpace(input.Name) == "" {
		fields["name"] = "name is required"
	}
	validateStatus(input.Status, fields)
	validateSortOrder("sort_order", input.SortOrder, fields)

	return fieldsError(fields)
}

func validateUpdateSkill(input UpdateSkillRequest) error {
	fields := make(map[string]string)

	if input.CategoryID != nil && strings.TrimSpace(*input.CategoryID) == "" {
		fields["category_id"] = "category_id cannot be blank"
	}
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		fields["name"] = "name cannot be blank"
	}
	if input.Status != nil {
		validateStatus(*input.Status, fields)
	}
	if input.SortOrder != nil {
		validateSortOrder("sort_order", *input.SortOrder, fields)
	}

	return fieldsError(fields)
}

func validateCategoryListFilter(filter CategoryListFilter) error {
	fields := make(map[string]string)
	validateStatus(filter.Status, fields)
	return fieldsError(fields)
}

func validateSkillListFilter(filter SkillListFilter) error {
	fields := make(map[string]string)
	validateStatus(filter.Status, fields)
	return fieldsError(fields)
}

func validateSlug(slug string, fields map[string]string) {
	if slug == "" {
		return
	}
	if !slugPattern.MatchString(slug) {
		fields["slug"] = "slug must use lowercase letters, numbers, and hyphens"
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

func validateSortOrder(field string, sortOrder int, fields map[string]string) {
	if sortOrder < 0 {
		fields[field] = "sort order must be zero or greater"
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

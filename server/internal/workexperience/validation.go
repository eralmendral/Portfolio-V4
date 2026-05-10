package workexperience

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

func validateCreate(input CreateWorkExperienceRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.Title) == "" {
		fields["title"] = "title is required"
	}
	if strings.TrimSpace(input.Company) == "" {
		fields["company"] = "company is required"
	}
	if input.StartedAt == nil {
		fields["started_at"] = "started_at is required"
	} else if input.StartedAt.IsZero() {
		fields["started_at"] = "started_at is required"
	}
	validateSlug(input.Slug, fields)
	validateURL("company_url", input.CompanyURL, fields)
	validateURL("company_logo_url", input.CompanyLogoURL, fields)
	validateStatus(input.Status, fields)
	validateSortOrder("sort_order", input.SortOrder, fields)
	validateEmploymentRange(input.StartedAt, input.EndedAt, input.Current, fields)

	return fieldsError(fields)
}

func validateUpdate(input UpdateWorkExperienceRequest) error {
	fields := make(map[string]string)

	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		fields["title"] = "title cannot be blank"
	}
	if input.Company != nil && strings.TrimSpace(*input.Company) == "" {
		fields["company"] = "company cannot be blank"
	}
	if input.Slug != nil {
		validateSlug(*input.Slug, fields)
	}
	if input.CompanyURL != nil {
		validateURL("company_url", *input.CompanyURL, fields)
	}
	if input.CompanyLogoURL != nil {
		validateURL("company_logo_url", *input.CompanyLogoURL, fields)
	}
	if input.Status != nil {
		validateStatus(*input.Status, fields)
	}
	if input.SortOrder != nil {
		validateSortOrder("sort_order", *input.SortOrder, fields)
	}
	if input.StartedAt != nil && input.StartedAt.IsZero() {
		fields["started_at"] = "started_at is required"
	}
	if input.Current != nil && *input.Current && input.EndedAt != nil {
		fields["ended_at"] = "ended_at must be empty for a current experience"
	}
	if input.StartedAt != nil && input.EndedAt != nil && input.EndedAt.Before(*input.StartedAt) {
		fields["ended_at"] = "ended_at cannot be before started_at"
	}

	return fieldsError(fields)
}

func validateRecord(experience WorkExperience) error {
	fields := make(map[string]string)

	if experience.StartedAt.IsZero() {
		fields["started_at"] = "started_at is required"
	}
	if experience.Current && experience.EndedAt != nil {
		fields["ended_at"] = "ended_at must be empty for a current experience"
	}
	if experience.EndedAt != nil && experience.EndedAt.Before(experience.StartedAt) {
		fields["ended_at"] = "ended_at cannot be before started_at"
	}

	return fieldsError(fields)
}

func validateEmploymentRange(startedAt *time.Time, endedAt *time.Time, current bool, fields map[string]string) {
	if current && endedAt != nil {
		fields["ended_at"] = "ended_at must be empty for a current experience"
	}
	if startedAt != nil && endedAt != nil && endedAt.Before(*startedAt) {
		fields["ended_at"] = "ended_at cannot be before started_at"
	}
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

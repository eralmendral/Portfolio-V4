package certificates

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

func validateCreate(input CreateCertificateRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.Title) == "" {
		fields["title"] = "title is required"
	}
	if strings.TrimSpace(input.Issuer) == "" {
		fields["issuer"] = "issuer is required"
	}
	validateSlug(input.Slug, fields)
	validateStatus(input.Status, fields)
	validateSortOrder("sort_order", input.SortOrder, fields)
	validateURL("credential_url", input.CredentialURL, fields)
	validateDates(input.IssuedAt, input.ExpiresAt, fields)

	if input.Image != nil {
		validateImageInput("image", *input.Image, fields)
	}

	return fieldsError(fields)
}

func validateUpdate(input UpdateCertificateRequest) error {
	fields := make(map[string]string)

	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		fields["title"] = "title cannot be blank"
	}
	if input.Issuer != nil && strings.TrimSpace(*input.Issuer) == "" {
		fields["issuer"] = "issuer cannot be blank"
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
	if input.CredentialURL != nil {
		validateURL("credential_url", *input.CredentialURL, fields)
	}
	validateDates(input.IssuedAt, input.ExpiresAt, fields)
	if input.Image != nil {
		validateImageInput("image", *input.Image, fields)
	}

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

func validateImageInput(field string, input CertificateImageInput, fields map[string]string) {
	if strings.TrimSpace(input.URL) == "" {
		fields[field+".url"] = "image URL is required"
		return
	}
	validateURL(field+".url", input.URL, fields)
}

func validateDates(issuedAt *time.Time, expiresAt *time.Time, fields map[string]string) {
	if issuedAt != nil && expiresAt != nil && expiresAt.Before(*issuedAt) {
		fields["expires_at"] = "expires_at cannot be before issued_at"
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

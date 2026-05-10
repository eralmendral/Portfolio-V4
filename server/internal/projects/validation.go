package projects

import (
	"errors"
	"fmt"
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

func validateCreate(input CreateProjectRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.Title) == "" {
		fields["title"] = "title is required"
	}
	validateSlug(input.Slug, fields)
	validateStatus(input.Status, fields)
	validateURL("github_url", input.GitHubURL, fields)
	validateURL("demo_url", input.DemoURL, fields)

	if input.MainImage != nil {
		validateImageInput("main_image", *input.MainImage, fields)
	}
	for i, image := range input.Images {
		validateImageInput(fmt.Sprintf("images[%d]", i), image, fields)
	}

	return fieldsError(fields)
}

func validateUpdate(input UpdateProjectRequest) error {
	fields := make(map[string]string)

	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		fields["title"] = "title cannot be blank"
	}
	if input.Slug != nil {
		validateSlug(*input.Slug, fields)
	}
	if input.Status != nil {
		validateStatus(*input.Status, fields)
	}
	if input.GitHubURL != nil {
		validateURL("github_url", *input.GitHubURL, fields)
	}
	if input.DemoURL != nil {
		validateURL("demo_url", *input.DemoURL, fields)
	}
	if input.MainImage != nil {
		validateImageInput("main_image", *input.MainImage, fields)
	}
	if input.Images != nil {
		for i, image := range *input.Images {
			validateImageInput(fmt.Sprintf("images[%d]", i), image, fields)
		}
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

func validateURL(field string, value string, fields map[string]string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		fields[field] = "must be a valid http or https URL"
	}
}

func validateImageInput(field string, input ProjectImageInput, fields map[string]string) {
	if strings.TrimSpace(input.URL) == "" {
		fields[field+".url"] = "image URL is required"
		return
	}
	validateURL(field+".url", input.URL, fields)
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

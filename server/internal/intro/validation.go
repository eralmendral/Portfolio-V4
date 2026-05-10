package intro

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

func validateUpdate(input UpdateIntroRequest) error {
	fields := make(map[string]string)

	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		fields["title"] = "title cannot be blank"
	}
	if input.Description != nil && strings.TrimSpace(*input.Description) == "" {
		fields["description"] = "description cannot be blank"
	}
	if input.ProfilePicture != nil {
		validateImageInput("profile_picture", *input.ProfilePicture, fields)
	}

	return fieldsError(fields)
}

func validateSavedIntro(value Intro) error {
	fields := make(map[string]string)

	if strings.TrimSpace(value.Title) == "" {
		fields["title"] = "title is required"
	}
	if strings.TrimSpace(value.Description) == "" {
		fields["description"] = "description is required"
	}

	return fieldsError(fields)
}

func validateImageInput(field string, input IntroImageInput, fields map[string]string) {
	if strings.TrimSpace(input.URL) == "" {
		fields[field+".url"] = "image URL is required"
		return
	}
	validateURL(field+".url", input.URL, fields)
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

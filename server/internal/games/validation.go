package games

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

type validationError struct {
	fields map[string]string
}

func (e validationError) Error() string {
	return "validation failed"
}

func (e validationError) Fields() map[string]string {
	return e.fields
}

func validateCreate(input CreateGameRequest) error {
	fields := make(map[string]string)

	if strings.TrimSpace(input.Title) == "" {
		fields["title"] = "title is required"
	}
	if strings.TrimSpace(input.Platform) == "" {
		fields["platform"] = "platform is required"
	}
	validateDate("mostly_played_on", input.MostlyPlayedOn, true, fields)
	validateURL("store_url", input.StoreURL, fields)
	validateSortOrder("sort_order", input.SortOrder, fields)
	validateStatus(input.Status, fields)

	return fieldsError(fields)
}

func validateUpdate(input UpdateGameRequest) error {
	fields := make(map[string]string)

	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		fields["title"] = "title cannot be blank"
	}
	if input.Platform != nil && strings.TrimSpace(*input.Platform) == "" {
		fields["platform"] = "platform cannot be blank"
	}
	if input.MostlyPlayedOn != nil {
		validateDate("mostly_played_on", *input.MostlyPlayedOn, true, fields)
	}
	if input.StoreURL != nil {
		validateURL("store_url", *input.StoreURL, fields)
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
	validateDate("from", filter.From, false, fields)
	validateDate("to", filter.To, false, fields)

	if filter.From != "" && filter.To != "" {
		from, _ := parseDate(filter.From)
		to, _ := parseDate(filter.To)
		if from.After(to) {
			fields["to"] = "to must be on or after from"
		}
	}

	return fieldsError(fields)
}

func validateDate(field string, value string, required bool, fields map[string]string) {
	if strings.TrimSpace(value) == "" {
		if required {
			fields[field] = "date is required"
		}
		return
	}
	if _, err := parseDate(value); err != nil {
		fields[field] = "date must use YYYY-MM-DD"
	}
}

func parseDate(value string) (time.Time, error) {
	return time.Parse(dateLayout, strings.TrimSpace(value))
}

func normalizeDate(value string) string {
	date, err := parseDate(value)
	if err != nil {
		return strings.TrimSpace(value)
	}
	return date.Format(dateLayout)
}

func validateURL(field string, value string, fields map[string]string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		fields[field] = "must be a valid http or https URL"
	}
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

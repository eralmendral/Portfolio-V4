package dailyprogress

import (
	"errors"
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

func validateCreate(input CreateDailyProgressRequest) error {
	fields := make(map[string]string)

	validateDate("entry_date", input.EntryDate, true, fields)
	if strings.TrimSpace(input.Title) == "" {
		fields["title"] = "title is required"
	}
	validateProgressScore("progress_score", input.ProgressScore, fields)
	validateStatus(input.Status, fields)

	return fieldsError(fields)
}

func validateUpdate(input UpdateDailyProgressRequest) error {
	fields := make(map[string]string)

	if input.EntryDate != nil {
		validateDate("entry_date", *input.EntryDate, true, fields)
	}
	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		fields["title"] = "title cannot be blank"
	}
	if input.ProgressScore != nil {
		validateProgressScore("progress_score", *input.ProgressScore, fields)
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

func validateProgressScore(field string, score int, fields map[string]string) {
	if score < 0 || score > 100 {
		fields[field] = "progress score must be between 0 and 100"
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

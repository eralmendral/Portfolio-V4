package contact

import (
	"net/mail"
	"strings"
)

type validationError map[string]string

func (e validationError) Error() string {
	return "validation failed"
}

func (e validationError) Fields() map[string]string {
	return map[string]string(e)
}

func isValidationError(err error) (validationError, bool) {
	validation, ok := err.(validationError)
	return validation, ok
}

func validateAndNormalize(input CreateSubmissionRequest) (Submission, error) {
	fields := validationError{}
	submission := Submission{
		ID:      newID(),
		Name:    strings.TrimSpace(input.Name),
		Email:   strings.TrimSpace(input.Email),
		Subject: strings.TrimSpace(input.Subject),
		Message: strings.TrimSpace(input.Message),
	}

	if submission.ID == "" {
		fields["id"] = "could not generate submission id"
	}
	if submission.Name == "" {
		fields["name"] = "name is required"
	} else if len(submission.Name) > 120 {
		fields["name"] = "name must be 120 characters or fewer"
	}
	if submission.Email == "" {
		fields["email"] = "email is required"
	} else if len(submission.Email) > 254 {
		fields["email"] = "email must be 254 characters or fewer"
	} else if _, err := mail.ParseAddress(submission.Email); err != nil {
		fields["email"] = "email must be valid"
	}
	if len(submission.Subject) > 160 {
		fields["subject"] = "subject must be 160 characters or fewer"
	}
	if submission.Message == "" {
		fields["message"] = "message is required"
	} else if len(submission.Message) > 4000 {
		fields["message"] = "message must be 4000 characters or fewer"
	}

	if len(fields) > 0 {
		return Submission{}, fields
	}

	return submission, nil
}

func applyProfileUpdate(profile *Profile, input UpdateProfileRequest) {
	if input.WorkEmail != nil {
		profile.WorkEmail = strings.TrimSpace(*input.WorkEmail)
	}
	if input.PhoneNumber != nil {
		profile.PhoneNumber = strings.TrimSpace(*input.PhoneNumber)
	}
}

func validateSavedProfile(profile Profile) error {
	fields := validationError{}
	if profile.ID == "" {
		fields["id"] = "id is required"
	} else if profile.ID != DefaultProfileID {
		fields["id"] = "id must be default"
	}
	if profile.WorkEmail == "" {
		fields["work_email"] = "work email is required"
	} else if len(profile.WorkEmail) > 254 {
		fields["work_email"] = "work email must be 254 characters or fewer"
	} else if _, err := mail.ParseAddress(profile.WorkEmail); err != nil {
		fields["work_email"] = "work email must be valid"
	}
	if profile.PhoneNumber == "" {
		fields["phone_number"] = "phone number is required"
	} else if len(profile.PhoneNumber) > 40 {
		fields["phone_number"] = "phone number must be 40 characters or fewer"
	}
	if len(fields) > 0 {
		return fields
	}
	return nil
}

func validateSaved(submission Submission) error {
	fields := validationError{}
	if submission.ID == "" {
		fields["id"] = "id is required"
	}
	if submission.Name == "" {
		fields["name"] = "name is required"
	}
	if submission.Email == "" {
		fields["email"] = "email is required"
	}
	if submission.Message == "" {
		fields["message"] = "message is required"
	}
	if len(fields) > 0 {
		return fields
	}
	return nil
}

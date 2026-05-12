package contact

import "time"

const DefaultProfileID = "default"

type Submission struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Subject   string    `json:"subject,omitempty"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateSubmissionRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type Profile struct {
	ID          string    `json:"id"`
	WorkEmail   string    `json:"work_email"`
	PhoneNumber string    `json:"phone_number"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	WorkEmail   *string `json:"work_email"`
	PhoneNumber *string `json:"phone_number"`
}

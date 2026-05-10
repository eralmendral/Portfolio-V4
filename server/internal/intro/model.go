package intro

import "time"

const DefaultID = "default"

type Intro struct {
	ID             string      `json:"id"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	ProfilePicture *IntroImage `json:"profile_picture,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type IntroImage struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Path        string    `json:"path,omitempty"`
	AltText     string    `json:"alt_text,omitempty"`
	Caption     string    `json:"caption,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	SizeBytes   int64     `json:"size_bytes,omitempty"`
	Width       int       `json:"width,omitempty"`
	Height      int       `json:"height,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type IntroImageInput struct {
	URL     string `json:"url"`
	AltText string `json:"alt_text"`
	Caption string `json:"caption"`
}

type UpdateIntroRequest struct {
	Title          *string          `json:"title"`
	Description    *string          `json:"description"`
	ProfilePicture *IntroImageInput `json:"profile_picture"`
}

package certificates

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type Certificate struct {
	ID            string            `json:"id"`
	Slug          string            `json:"slug"`
	Title         string            `json:"title"`
	Issuer        string            `json:"issuer"`
	Summary       string            `json:"summary,omitempty"`
	Description   string            `json:"description,omitempty"`
	CredentialURL string            `json:"credential_url,omitempty"`
	Image         *CertificateImage `json:"image,omitempty"`
	Featured      bool              `json:"featured"`
	SortOrder     int               `json:"sort_order"`
	Status        string            `json:"status"`
	IssuedAt      *time.Time        `json:"issued_at,omitempty"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type CertificateImage struct {
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

type CertificateImageInput struct {
	URL     string `json:"url"`
	AltText string `json:"alt_text"`
	Caption string `json:"caption"`
}

type CreateCertificateRequest struct {
	Slug          string                 `json:"slug"`
	Title         string                 `json:"title"`
	Issuer        string                 `json:"issuer"`
	Summary       string                 `json:"summary"`
	Description   string                 `json:"description"`
	CredentialURL string                 `json:"credential_url"`
	Image         *CertificateImageInput `json:"image"`
	Featured      bool                   `json:"featured"`
	SortOrder     int                    `json:"sort_order"`
	Status        string                 `json:"status"`
	IssuedAt      *time.Time             `json:"issued_at"`
	ExpiresAt     *time.Time             `json:"expires_at"`
}

type UpdateCertificateRequest struct {
	Slug          *string                `json:"slug"`
	Title         *string                `json:"title"`
	Issuer        *string                `json:"issuer"`
	Summary       *string                `json:"summary"`
	Description   *string                `json:"description"`
	CredentialURL *string                `json:"credential_url"`
	Image         *CertificateImageInput `json:"image"`
	Featured      *bool                  `json:"featured"`
	SortOrder     *int                   `json:"sort_order"`
	Status        *string                `json:"status"`
	IssuedAt      *time.Time             `json:"issued_at"`
	ExpiresAt     *time.Time             `json:"expires_at"`
}

type ListFilter struct {
	Status   string
	Featured *bool
	Query    string
}

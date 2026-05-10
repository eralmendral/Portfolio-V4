package products

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type Product struct {
	ID            string     `json:"id"`
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary,omitempty"`
	Description   string     `json:"description,omitempty"`
	CoverImageURL string     `json:"cover_image_url,omitempty"`
	PriceLabel    string     `json:"price_label,omitempty"`
	CTALabel      string     `json:"cta_label,omitempty"`
	CTAURL        string     `json:"cta_url"`
	Category      string     `json:"category,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	Featured      bool       `json:"featured"`
	SortOrder     int        `json:"sort_order"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CreateProductRequest struct {
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	Description   string     `json:"description"`
	CoverImageURL string     `json:"cover_image_url"`
	PriceLabel    string     `json:"price_label"`
	CTALabel      string     `json:"cta_label"`
	CTAURL        string     `json:"cta_url"`
	Category      string     `json:"category"`
	Tags          []string   `json:"tags"`
	Featured      bool       `json:"featured"`
	SortOrder     int        `json:"sort_order"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"published_at"`
}

type UpdateProductRequest struct {
	Slug          *string    `json:"slug"`
	Title         *string    `json:"title"`
	Summary       *string    `json:"summary"`
	Description   *string    `json:"description"`
	CoverImageURL *string    `json:"cover_image_url"`
	PriceLabel    *string    `json:"price_label"`
	CTALabel      *string    `json:"cta_label"`
	CTAURL        *string    `json:"cta_url"`
	Category      *string    `json:"category"`
	Tags          *[]string  `json:"tags"`
	Featured      *bool      `json:"featured"`
	SortOrder     *int       `json:"sort_order"`
	Status        *string    `json:"status"`
	PublishedAt   *time.Time `json:"published_at"`
}

type ProductSectionSettings struct {
	Enabled     bool      `json:"enabled"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateProductSectionRequest struct {
	Enabled     *bool   `json:"enabled"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type ListFilter struct {
	Status   string
	Featured *bool
	Category string
	Tag      string
	Query    string
}

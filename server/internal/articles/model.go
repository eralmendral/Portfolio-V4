package articles

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type Article struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	URL           string     `json:"url"`
	Source        string     `json:"source"`
	Summary       string     `json:"summary,omitempty"`
	CoverImageURL string     `json:"cover_image_url,omitempty"`
	Featured      bool       `json:"featured"`
	SortOrder     int        `json:"sort_order"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CreateArticleRequest struct {
	Title         string     `json:"title"`
	URL           string     `json:"url"`
	Source        string     `json:"source"`
	Summary       string     `json:"summary"`
	CoverImageURL string     `json:"cover_image_url"`
	Featured      bool       `json:"featured"`
	SortOrder     int        `json:"sort_order"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"published_at"`
}

type UpdateArticleRequest struct {
	Title         *string    `json:"title"`
	URL           *string    `json:"url"`
	Source        *string    `json:"source"`
	Summary       *string    `json:"summary"`
	CoverImageURL *string    `json:"cover_image_url"`
	Featured      *bool      `json:"featured"`
	SortOrder     *int       `json:"sort_order"`
	Status        *string    `json:"status"`
	PublishedAt   *time.Time `json:"published_at"`
}

type ListFilter struct {
	Status   string
	Featured *bool
	Query    string
}

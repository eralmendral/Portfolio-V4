package links

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type Link struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	URL       string    `json:"url"`
	IconClass string    `json:"icon_class,omitempty"`
	SortOrder int       `json:"sort_order"`
	Star      bool      `json:"star"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateLinkRequest struct {
	Label     string `json:"label"`
	URL       string `json:"url"`
	IconClass string `json:"icon_class"`
	SortOrder int    `json:"sort_order"`
	Star      bool   `json:"star"`
	Status    string `json:"status"`
}

type UpdateLinkRequest struct {
	Label     *string `json:"label"`
	URL       *string `json:"url"`
	IconClass *string `json:"icon_class"`
	SortOrder *int    `json:"sort_order"`
	Star      *bool   `json:"star"`
	Status    *string `json:"status"`
}

type ListFilter struct {
	Status string
	Star   *bool
	Query  string
}

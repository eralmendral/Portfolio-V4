package tools

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type Tool struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Summary   string    `json:"summary,omitempty"`
	IconClass string    `json:"icon_class,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	SortOrder int       `json:"sort_order"`
	Featured  bool      `json:"featured"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateToolRequest struct {
	Name      string   `json:"name"`
	Category  string   `json:"category"`
	Summary   string   `json:"summary"`
	IconClass string   `json:"icon_class"`
	Tags      []string `json:"tags"`
	SortOrder int      `json:"sort_order"`
	Featured  bool     `json:"featured"`
	Status    string   `json:"status"`
}

type UpdateToolRequest struct {
	Name      *string   `json:"name"`
	Category  *string   `json:"category"`
	Summary   *string   `json:"summary"`
	IconClass *string   `json:"icon_class"`
	Tags      *[]string `json:"tags"`
	SortOrder *int      `json:"sort_order"`
	Featured  *bool     `json:"featured"`
	Status    *string   `json:"status"`
}

type ListFilter struct {
	Status   string
	Featured *bool
	Category string
	Tag      string
	Query    string
}

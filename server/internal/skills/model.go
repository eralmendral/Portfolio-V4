package skills

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type SkillCategory struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IconClass   string    `json:"icon_class,omitempty"`
	SortOrder   int       `json:"sort_order"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Skill struct {
	ID         string    `json:"id"`
	CategoryID string    `json:"category_id"`
	Name       string    `json:"name"`
	Summary    string    `json:"summary,omitempty"`
	IconClass  string    `json:"icon_class,omitempty"`
	SortOrder  int       `json:"sort_order"`
	Featured   bool      `json:"featured"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateSkillCategoryRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconClass   string `json:"icon_class"`
	SortOrder   int    `json:"sort_order"`
	Status      string `json:"status"`
}

type UpdateSkillCategoryRequest struct {
	Slug        *string `json:"slug"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IconClass   *string `json:"icon_class"`
	SortOrder   *int    `json:"sort_order"`
	Status      *string `json:"status"`
}

type CreateSkillRequest struct {
	CategoryID string `json:"category_id"`
	Name       string `json:"name"`
	Summary    string `json:"summary"`
	IconClass  string `json:"icon_class"`
	SortOrder  int    `json:"sort_order"`
	Featured   bool   `json:"featured"`
	Status     string `json:"status"`
}

type UpdateSkillRequest struct {
	CategoryID *string `json:"category_id"`
	Name       *string `json:"name"`
	Summary    *string `json:"summary"`
	IconClass  *string `json:"icon_class"`
	SortOrder  *int    `json:"sort_order"`
	Featured   *bool   `json:"featured"`
	Status     *string `json:"status"`
}

type CategoryListFilter struct {
	Status string
	Query  string
}

type SkillListFilter struct {
	Status     string
	Featured   *bool
	CategoryID string
	Category   string
	Query      string
}

package projects

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type Project struct {
	ID          string         `json:"id"`
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	Body        string         `json:"body,omitempty"`
	TechStack   []string       `json:"tech_stack,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	MainImage   *ProjectImage  `json:"main_image,omitempty"`
	Images      []ProjectImage `json:"images,omitempty"`
	GitHubURL   string         `json:"github_url,omitempty"`
	DemoURL     string         `json:"demo_url,omitempty"`
	Featured    bool           `json:"featured"`
	SortOrder   int            `json:"sort_order"`
	Status      string         `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	PublishedAt *time.Time     `json:"published_at,omitempty"`
}

type ProjectImage struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Path        string    `json:"path,omitempty"`
	AltText     string    `json:"alt_text,omitempty"`
	Caption     string    `json:"caption,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	SizeBytes   int64     `json:"size_bytes,omitempty"`
	Width       int       `json:"width,omitempty"`
	Height      int       `json:"height,omitempty"`
	SortOrder   int       `json:"sort_order"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type ProjectImageInput struct {
	URL       string `json:"url"`
	AltText   string `json:"alt_text"`
	Caption   string `json:"caption"`
	SortOrder int    `json:"sort_order"`
}

type CreateProjectRequest struct {
	Slug        string              `json:"slug"`
	Title       string              `json:"title"`
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	Body        string              `json:"body"`
	TechStack   []string            `json:"tech_stack"`
	Tags        []string            `json:"tags"`
	MainImage   *ProjectImageInput  `json:"main_image"`
	Images      []ProjectImageInput `json:"images"`
	GitHubURL   string              `json:"github_url"`
	DemoURL     string              `json:"demo_url"`
	Featured    bool                `json:"featured"`
	SortOrder   int                 `json:"sort_order"`
	Status      string              `json:"status"`
	PublishedAt *time.Time          `json:"published_at"`
}

type UpdateProjectRequest struct {
	Slug        *string              `json:"slug"`
	Title       *string              `json:"title"`
	Summary     *string              `json:"summary"`
	Description *string              `json:"description"`
	Body        *string              `json:"body"`
	TechStack   *[]string            `json:"tech_stack"`
	Tags        *[]string            `json:"tags"`
	MainImage   *ProjectImageInput   `json:"main_image"`
	Images      *[]ProjectImageInput `json:"images"`
	GitHubURL   *string              `json:"github_url"`
	DemoURL     *string              `json:"demo_url"`
	Featured    *bool                `json:"featured"`
	SortOrder   *int                 `json:"sort_order"`
	Status      *string              `json:"status"`
	PublishedAt *time.Time           `json:"published_at"`
}

type ListFilter struct {
	Status   string
	Featured *bool
	Query    string
}

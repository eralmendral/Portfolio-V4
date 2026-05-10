package workexperience

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type WorkExperience struct {
	ID               string     `json:"id"`
	Slug             string     `json:"slug"`
	Title            string     `json:"title"`
	Company          string     `json:"company"`
	CompanyURL       string     `json:"company_url,omitempty"`
	CompanyLogoURL   string     `json:"company_logo_url,omitempty"`
	EmploymentType   string     `json:"employment_type,omitempty"`
	Location         string     `json:"location,omitempty"`
	LocationType     string     `json:"location_type,omitempty"`
	Summary          string     `json:"summary,omitempty"`
	Description      string     `json:"description,omitempty"`
	Highlights       []string   `json:"highlights,omitempty"`
	Responsibilities []string   `json:"responsibilities,omitempty"`
	TechStack        []string   `json:"tech_stack,omitempty"`
	Skills           []string   `json:"skills,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
	Current          bool       `json:"current"`
	Featured         bool       `json:"featured"`
	SortOrder        int        `json:"sort_order"`
	Status           string     `json:"status"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type CreateWorkExperienceRequest struct {
	Slug             string     `json:"slug"`
	Title            string     `json:"title"`
	Company          string     `json:"company"`
	CompanyURL       string     `json:"company_url"`
	CompanyLogoURL   string     `json:"company_logo_url"`
	EmploymentType   string     `json:"employment_type"`
	Location         string     `json:"location"`
	LocationType     string     `json:"location_type"`
	Summary          string     `json:"summary"`
	Description      string     `json:"description"`
	Highlights       []string   `json:"highlights"`
	Responsibilities []string   `json:"responsibilities"`
	TechStack        []string   `json:"tech_stack"`
	Skills           []string   `json:"skills"`
	StartedAt        *time.Time `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
	Current          bool       `json:"current"`
	Featured         bool       `json:"featured"`
	SortOrder        int        `json:"sort_order"`
	Status           string     `json:"status"`
	PublishedAt      *time.Time `json:"published_at"`
}

type UpdateWorkExperienceRequest struct {
	Slug             *string    `json:"slug"`
	Title            *string    `json:"title"`
	Company          *string    `json:"company"`
	CompanyURL       *string    `json:"company_url"`
	CompanyLogoURL   *string    `json:"company_logo_url"`
	EmploymentType   *string    `json:"employment_type"`
	Location         *string    `json:"location"`
	LocationType     *string    `json:"location_type"`
	Summary          *string    `json:"summary"`
	Description      *string    `json:"description"`
	Highlights       *[]string  `json:"highlights"`
	Responsibilities *[]string  `json:"responsibilities"`
	TechStack        *[]string  `json:"tech_stack"`
	Skills           *[]string  `json:"skills"`
	StartedAt        *time.Time `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
	Current          *bool      `json:"current"`
	Featured         *bool      `json:"featured"`
	SortOrder        *int       `json:"sort_order"`
	Status           *string    `json:"status"`
	PublishedAt      *time.Time `json:"published_at"`
}

type ListFilter struct {
	Status   string
	Featured *bool
	Current  *bool
	Query    string
}

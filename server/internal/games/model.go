package games

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type Game struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Studio         string    `json:"studio,omitempty"`
	Platform       string    `json:"platform"`
	Genre          string    `json:"genre,omitempty"`
	StoreURL       string    `json:"store_url,omitempty"`
	MostlyPlayedOn string    `json:"mostly_played_on"`
	Notes          string    `json:"notes,omitempty"`
	SortOrder      int       `json:"sort_order"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateGameRequest struct {
	Title          string `json:"title"`
	Studio         string `json:"studio"`
	Platform       string `json:"platform"`
	Genre          string `json:"genre"`
	StoreURL       string `json:"store_url"`
	MostlyPlayedOn string `json:"mostly_played_on"`
	Notes          string `json:"notes"`
	SortOrder      int    `json:"sort_order"`
	Status         string `json:"status"`
}

type UpdateGameRequest struct {
	Title          *string `json:"title"`
	Studio         *string `json:"studio"`
	Platform       *string `json:"platform"`
	Genre          *string `json:"genre"`
	StoreURL       *string `json:"store_url"`
	MostlyPlayedOn *string `json:"mostly_played_on"`
	Notes          *string `json:"notes"`
	SortOrder      *int    `json:"sort_order"`
	Status         *string `json:"status"`
}

type ListFilter struct {
	Status   string
	Platform string
	Query    string
	From     string
	To       string
}

package series

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"

	CategoryTVSeries = "tv_series"
	CategoryAnime    = "anime"
)

type Series struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Category        string    `json:"category"`
	Creator         string    `json:"creator,omitempty"`
	Platform        string    `json:"platform,omitempty"`
	WatchURL        string    `json:"watch_url,omitempty"`
	MostlyWatchedOn string    `json:"mostly_watched_on"`
	Notes           string    `json:"notes,omitempty"`
	SortOrder       int       `json:"sort_order"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateSeriesRequest struct {
	Title           string `json:"title"`
	Category        string `json:"category"`
	Creator         string `json:"creator"`
	Platform        string `json:"platform"`
	WatchURL        string `json:"watch_url"`
	MostlyWatchedOn string `json:"mostly_watched_on"`
	Notes           string `json:"notes"`
	SortOrder       int    `json:"sort_order"`
	Status          string `json:"status"`
}

type UpdateSeriesRequest struct {
	Title           *string `json:"title"`
	Category        *string `json:"category"`
	Creator         *string `json:"creator"`
	Platform        *string `json:"platform"`
	WatchURL        *string `json:"watch_url"`
	MostlyWatchedOn *string `json:"mostly_watched_on"`
	Notes           *string `json:"notes"`
	SortOrder       *int    `json:"sort_order"`
	Status          *string `json:"status"`
}

type ListFilter struct {
	Status   string
	Category string
	Query    string
	From     string
	To       string
}

package music

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type Music struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Artist           string    `json:"artist"`
	Album            string    `json:"album,omitempty"`
	SpotifyURL       string    `json:"spotify_url,omitempty"`
	YouTubeURL       string    `json:"youtube_url,omitempty"`
	MostlyListenedOn string    `json:"mostly_listened_on"`
	Notes            string    `json:"notes,omitempty"`
	SortOrder        int       `json:"sort_order"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateMusicRequest struct {
	Title            string `json:"title"`
	Artist           string `json:"artist"`
	Album            string `json:"album"`
	SpotifyURL       string `json:"spotify_url"`
	YouTubeURL       string `json:"youtube_url"`
	MostlyListenedOn string `json:"mostly_listened_on"`
	Notes            string `json:"notes"`
	SortOrder        int    `json:"sort_order"`
	Status           string `json:"status"`
}

type UpdateMusicRequest struct {
	Title            *string `json:"title"`
	Artist           *string `json:"artist"`
	Album            *string `json:"album"`
	SpotifyURL       *string `json:"spotify_url"`
	YouTubeURL       *string `json:"youtube_url"`
	MostlyListenedOn *string `json:"mostly_listened_on"`
	Notes            *string `json:"notes"`
	SortOrder        *int    `json:"sort_order"`
	Status           *string `json:"status"`
}

type ListFilter struct {
	Status string
	Query  string
	From   string
	To     string
}

package dailyprogress

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type DailyProgress struct {
	ID            string    `json:"id"`
	EntryDate     string    `json:"entry_date"`
	Title         string    `json:"title"`
	Summary       string    `json:"summary,omitempty"`
	Content       string    `json:"content,omitempty"`
	Mood          string    `json:"mood,omitempty"`
	ProgressScore int       `json:"progress_score"`
	Wins          []string  `json:"wins,omitempty"`
	Blockers      []string  `json:"blockers,omitempty"`
	Learnings     []string  `json:"learnings,omitempty"`
	NextSteps     []string  `json:"next_steps,omitempty"`
	Tags          []string  `json:"tags,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateDailyProgressRequest struct {
	EntryDate     string   `json:"entry_date"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Content       string   `json:"content"`
	Mood          string   `json:"mood"`
	ProgressScore int      `json:"progress_score"`
	Wins          []string `json:"wins"`
	Blockers      []string `json:"blockers"`
	Learnings     []string `json:"learnings"`
	NextSteps     []string `json:"next_steps"`
	Tags          []string `json:"tags"`
	Status        string   `json:"status"`
}

type UpdateDailyProgressRequest struct {
	EntryDate     *string   `json:"entry_date"`
	Title         *string   `json:"title"`
	Summary       *string   `json:"summary"`
	Content       *string   `json:"content"`
	Mood          *string   `json:"mood"`
	ProgressScore *int      `json:"progress_score"`
	Wins          *[]string `json:"wins"`
	Blockers      *[]string `json:"blockers"`
	Learnings     *[]string `json:"learnings"`
	NextSteps     *[]string `json:"next_steps"`
	Tags          *[]string `json:"tags"`
	Status        *string   `json:"status"`
}

type ListFilter struct {
	Status string
	Tag    string
	Query  string
	From   string
	To     string
}

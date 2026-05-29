package publish

import "time"

// Published represents a published note visible to the public.
type Published struct {
	Slug        string    `json:"slug"`
	NotePath    string    `json:"note_path"`
	Username    string    `json:"username"`
	Nickname    string    `json:"nickname"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Tags        []string  `json:"tags"`
	PublishedAt time.Time `json:"published_at"`
}

// PublishRequest is the body for publishing a note.
type PublishRequest struct {
	Slug string `json:"slug" binding:"required"`
}

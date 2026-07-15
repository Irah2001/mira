package notes

import (
	"time"
)

type Note struct {
	ID               int64     `json:"id"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	Tags             []string  `json:"tags"`
	EnrichmentStatus string    `json:"enrichment_status"` // "pending", "done", "failed"
	Summary          *string   `json:"summary,omitempty"`
	Score            int       `json:"score"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func NewNote(title, content string, tags []string) *Note {
	now := time.Now()

	if tags == nil {
		tags = []string{}
	}

	return &Note{
		ID:               now.UnixNano(),
		Title:            title,
		Content:          content,
		Tags:             tags,
		EnrichmentStatus: "pending",
		Score:            0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

type NotePayload struct {
	Title   *string  `json:"title"`
	Content *string  `json:"content"`
	Tags    []string `json:"tags"`
}

func (p NotePayload) Validate() map[string]string {
	errors := make(map[string]string)
	if p.Title == nil || *p.Title == "" {
		errors["title"] = "le titre est obligatoire et ne peut pas être vide"
	}
	if p.Content == nil || *p.Content == "" {
		errors["content"] = "le contenu est obligatoire et ne peut pas être vide"
	}
	return errors
}

package notes

import (
	"time"
)

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewNote(title, content string) Note {
	now := time.Now()
	return Note{
		ID:        now.UnixNano(),
		Title:     title,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type NotePayload struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
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

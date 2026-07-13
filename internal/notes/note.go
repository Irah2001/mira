package notes

import (
	"time"
)

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func NewNote(title, content string) Note {
	now := time.Now()
	return Note{
		ID:        now.UnixNano(),
		Title:     title,
		Content:   content,
		CreatedAt: now,
	}
}

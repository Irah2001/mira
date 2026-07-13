package notes

import "errors"

var ErrNotFound = errors.New("note non trouvée")

// NoteStore définit les opérations requises pour gérer le stockage des notes.
type NoteStore interface {
	Add(note Note) error
	List(limit int) ([]Note, error)
	GetAll() ([]Note, error)
	GetByID(id int64) (Note, error)
	Update(id int64, payload NotePayload) (Note, error)
	Delete(id int64) error
	ListPaginated(limit, offset int) ([]Note, int, error)
}

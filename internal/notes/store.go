package notes

// NoteStore définit les opérations requises pour gérer le stockage des notes.
type NoteStore interface {
	Add(note Note) error
	List(limit int) ([]Note, error)
	GetAll() ([]Note, error)
}

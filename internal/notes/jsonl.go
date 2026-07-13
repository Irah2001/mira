package notes

import (
	"bufio"
	"encoding/json"
	"os"
)

// JSONLStore implémente NoteStore avec un fichier JSON Lines.
type JSONLStore struct {
	FilePath string
}

// NewJSONLStore initialise le store.
func NewJSONLStore(path string) *JSONLStore {
	return &JSONLStore{FilePath: path}
}

// Add ajoute une note à la fin du fichier.
func (s *JSONLStore) Add(note Note) error {
	f, err := os.OpenFile(s.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(note)
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(data) + "\n")
	return err
}

// GetAll lit toutes les notes du fichier.
func (s *JSONLStore) GetAll() ([]Note, error) {
	f, err := os.Open(s.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Note{}, nil // Fichier vide/inexistant = aucune note
		}
		return nil, err
	}
	defer f.Close()

	var notes []Note
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var n Note
		if err := json.Unmarshal(scanner.Bytes(), &n); err != nil {
			continue // On ignore les lignes corrompues
		}
		notes = append(notes, n)
	}

	return notes, scanner.Err()
}

// List retourne les 'limit' dernières notes créées (les plus récentes).
func (s *JSONLStore) List(limit int) ([]Note, error) {
	allNotes, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	// Inverser pour avoir les plus récentes en premier
	for i, j := 0, len(allNotes)-1; i < j; i, j = i+1, j-1 {
		allNotes[i], allNotes[j] = allNotes[j], allNotes[i]
	}

	if len(allNotes) > limit {
		return allNotes[:limit], nil
	}
	return allNotes, nil
}

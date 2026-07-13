package notes

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// JSONLStore implémente NoteStore avec un fichier JSON Lines.
type JSONLStore struct {
	mu       sync.RWMutex // Protège le fichier contre les accès HTTP concurrents
	FilePath string
}

// NewJSONLStore initialise le store.
func NewJSONLStore(path string) *JSONLStore {
	return &JSONLStore{FilePath: path}
}

// Add ajoute une note à la fin du fichier.
func (s *JSONLStore) Add(note Note) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := os.Open(s.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Note{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var notes []Note
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var n Note
		if err := json.Unmarshal(scanner.Bytes(), &n); err != nil {
			continue
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

// rewriteAll est une méthode utilitaire interne pour réécrire le fichier complet
func (s *JSONLStore) rewriteAll(notes []Note) error {
	f, err := os.OpenFile(s.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, n := range notes {
		data, err := json.Marshal(n)
		if err != nil {
			return err
		}
		if _, err := f.WriteString(string(data) + "\n"); err != nil {
			return err
		}
	}
	return nil
}

// GetByID cherche une note spécifique par son ID int64
func (s *JSONLStore) GetByID(id int64) (Note, error) {
	allNotes, err := s.GetAll()
	if err != nil {
		return Note{}, err
	}

	for _, n := range allNotes {
		if n.ID == id {
			return n, nil
		}
	}
	return Note{}, ErrNotFound
}

// Update modifie partiellement (PATCH) une note en réécrivant le fichier
func (s *JSONLStore) Update(id int64, payload NotePayload) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Chargement direct via os.Open pour éviter un deadlock de verrou
	f, err := os.Open(s.FilePath)
	if err != nil {
		return Note{}, err
	}

	var allNotes []Note
	var updatedNote Note
	found := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var n Note
		if err := json.Unmarshal(scanner.Bytes(), &n); err == nil {
			if n.ID == id {
				if payload.Title != nil {
					n.Title = *payload.Title
				}
				if payload.Content != nil {
					n.Content = *payload.Content
				}
				n.UpdatedAt = time.Now()
				updatedNote = n
				found = true
			}
			allNotes = append(allNotes, n)
		}
	}
	f.Close()

	if !found {
		return Note{}, ErrNotFound
	}

	// Réécriture du fichier mis à jour
	if err := s.rewriteAll(allNotes); err != nil {
		return Note{}, err
	}

	return updatedNote, nil
}

// Delete supprime une note de la liste et réécrit le fichier
func (s *JSONLStore) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.FilePath)
	if err != nil {
		return err
	}

	var remainingNotes []Note
	found := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var n Note
		if err := json.Unmarshal(scanner.Bytes(), &n); err == nil {
			if n.ID == id {
				found = true
				continue // Exclut la note à supprimer
			}
			remainingNotes = append(remainingNotes, n)
		}
	}
	f.Close()

	if !found {
		return ErrNotFound
	}

	return s.rewriteAll(remainingNotes)
}

// ListPaginated implémente le Bonus de pagination (tri décroissant)
func (s *JSONLStore) ListPaginated(limit, offset int) ([]Note, int, error) {
	allNotes, err := s.GetAll()
	if err != nil {
		return nil, 0, err
	}

	total := len(allNotes)

	// Inverser pour avoir les plus récentes en premier
	for i, j := 0, len(allNotes)-1; i < j; i, j = i+1, j-1 {
		allNotes[i], allNotes[j] = allNotes[j], allNotes[i]
	}

	if offset >= total {
		return []Note{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return allNotes[offset:end], total, nil
}

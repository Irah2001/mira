package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"mira/internal/notes"
)

type MockStore struct {
	data map[int64]*notes.Note
}

func NewMockStore() *MockStore {
	return &MockStore{data: make(map[int64]*notes.Note)}
}

func (m *MockStore) Add(n *notes.Note) error {
	n.ID = 1
	m.data[n.ID] = n
	return nil
}

func (m *MockStore) GetByID(id int64) (notes.Note, error) {
	if note, exists := m.data[id]; exists {
		return *note, nil
	}

	return notes.Note{}, notes.ErrNotFound
}

func (m *MockStore) List(limit int) ([]notes.Note, error)                       { return nil, nil }
func (m *MockStore) ListPaginated(limit, offset int) ([]notes.Note, int, error) { return nil, 0, nil }
func (m *MockStore) GetAll() ([]notes.Note, error)                              { return nil, nil }
func (m *MockStore) Update(id int64, payload notes.NotePayload) (notes.Note, error) {
	return notes.Note{}, nil
}
func (m *MockStore) Delete(id int64) error                     { return nil }
func (m *MockStore) Search(query string) ([]notes.Note, error) { return nil, nil }
func (m *MockStore) UpdateEnrichmentStatus(id int64, status string, summary *string, score int, extraTags []string, embedding []float32) error {
	return nil
}

func TestCreateNote_SuccessAnd400(t *testing.T) {
	store := NewMockStore()

	dummyEnricher := notes.NewEnrichmentService(nil, 10, 1)

	handler := NewNoteHandler(store, dummyEnricher)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/notes", handler.Create)

	// --- TEST 1 : Création réussie (201 Created) ---
	payload := []byte(`{"title":"Mon super Titre","content":"Mon contenu"}`)
	req := httptest.NewRequest("POST", "/api/v1/notes", bytes.NewBuffer(payload))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Attendu statut %d (Created), obtenu %d", http.StatusCreated, rr.Code)
	}

	// --- TEST 2 : Erreur de validation (400 Bad Request) ---
	badPayload := []byte(`{"content":"Pas de titre"}`)
	reqBad := httptest.NewRequest("POST", "/api/v1/notes", bytes.NewBuffer(badPayload))
	rrBad := httptest.NewRecorder()

	mux.ServeHTTP(rrBad, reqBad)

	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("Attendu statut %d (BadRequest) pour validation erronée, obtenu %d", http.StatusBadRequest, rrBad.Code)
	}
}

func TestGetNote_NotFound(t *testing.T) {
	store := NewMockStore()
	dummyEnricher := notes.NewEnrichmentService(nil, 10, 1)
	handler := NewNoteHandler(store, dummyEnricher)

	mux := http.NewServeMux()
	// Route avec récupération de la variable {id} (Go 1.22+)
	mux.HandleFunc("GET /api/v1/notes/{id}", handler.Get)

	// --- TEST 3 : Note introuvable (404 Not Found) ---
	// L'ID 1234567890 n'existe pas dans notre MockStore
	req := httptest.NewRequest("GET", "/api/v1/notes/1234567890", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Attendu statut %d (NotFound) pour ID absent, obtenu %d", http.StatusNotFound, rr.Code)
	}
}

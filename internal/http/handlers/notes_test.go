package handlers

import (
	"bytes"
	"mira/internal/notes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupTestStore(t *testing.T) (*notes.JSONLStore, func()) {
	tmpFile, err := os.CreateTemp("", "mira_test_*.jsonl")
	if err != nil {
		t.Fatalf("Impossible de créer le fichier temporaire de test : %v", err)
	}

	store := notes.NewJSONLStore(tmpFile.Name())

	cleanUp := func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}

	return store, cleanUp
}

func TestCreateNote_SuccessAnd400(t *testing.T) {
	store, clean := setupTestStore(t)
	defer clean()

	dummyEnricher := notes.NewEnrichmentService(nil, 10, 1)

	handler := NewNoteHandler(store, dummyEnricher)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/notes", handler.Create)

	payload := []byte(`{"title":"Mon super Titre","content":"Mon contenu"}`)
	req := httptest.NewRequest("POST", "/api/v1/notes", bytes.NewBuffer(payload))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Attendu statut %d, obtenu %d", http.StatusCreated, rr.Code)
	}

	badPayload := []byte(`{"content":"Pas de titre"}`)
	reqBad := httptest.NewRequest("POST", "/api/v1/notes", bytes.NewBuffer(badPayload))
	rrBad := httptest.NewRecorder()

	mux.ServeHTTP(rrBad, reqBad)

	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("Attendu statut %d pour validation erronée, obtenu %d", http.StatusBadRequest, rrBad.Code)
	}
}

func TestGetNote_NotFound(t *testing.T) {
	store, clean := setupTestStore(t)
	defer clean()

	dummyEnricher := notes.NewEnrichmentService(nil, 10, 1)
	handler := NewNoteHandler(store, dummyEnricher)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/notes/{id}", handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/notes/1234567890", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Attendu statut %d (NotFound) pour ID absent, obtenu %d", http.StatusNotFound, rr.Code)
	}
}

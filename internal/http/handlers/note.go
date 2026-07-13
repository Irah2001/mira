package handlers

import (
	"encoding/json"
	"errors"
	"mira/internal/notes"
	"mira/internal/search"
	"net/http"
	"strconv"
)

type NoteHandler struct {
	store notes.NoteStore
}

func NewNoteHandler(s notes.NoteStore) *NoteHandler {
	return &NoteHandler{store: s}
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var payload notes.NotePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		JSONError(w, http.StatusBadRequest, "Corps JSON invalide")
		return
	}

	if errs := payload.Validate(); len(errs) > 0 {
		JSONError(w, http.StatusBadRequest, errs)
		return
	}

	note := notes.NewNote(*payload.Title, *payload.Content)
	if err := h.store.Add(note); err != nil {
		JSONError(w, http.StatusInternalServerError, "Erreur lors de l'écriture disque")
		return
	}

	JSON(w, http.StatusCreated, note, nil)
}

func (h *NoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id") // Go 1.22
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "Identifiant ID invalide (doit être un entier)")
		return
	}

	note, err := h.store.GetByID(id)
	if err != nil {
		if errors.Is(err, notes.ErrNotFound) {
			JSONError(w, http.StatusNotFound, "Note introuvable")
			return
		}
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, note, nil)
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	paginatedNotes, total, err := h.store.ListPaginated(limit, offset)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "Erreur lors de la lecture des données")
		return
	}

	meta := map[string]int{"total": total, "limit": limit, "offset": offset}
	JSON(w, http.StatusOK, paginatedNotes, meta)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "ID invalide")
		return
	}

	var payload notes.NotePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		JSONError(w, http.StatusBadRequest, "JSON malformé")
		return
	}

	updatedNote, err := h.store.Update(id, payload)
	if err != nil {
		if errors.Is(err, notes.ErrNotFound) {
			JSONError(w, http.StatusNotFound, "Note impossible à modifier car introuvable")
			return
		}
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, updatedNote, nil)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "ID invalide")
		return
	}

	if err := h.store.Delete(id); err != nil {
		if errors.Is(err, notes.ErrNotFound) {
			JSONError(w, http.StatusNotFound, "Note impossible à supprimer car introuvable")
			return
		}
		JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Note supprimée"}, nil)
}

func (h *NoteHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		JSONError(w, http.StatusBadRequest, "Le paramètre de requête '?q=' est obligatoire")
		return
	}

	allNotes, err := h.store.GetAll()
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "Erreur de traitement")
		return
	}

	results := search.Search(allNotes, query)
	JSON(w, http.StatusOK, results, map[string]int{"count": len(results)})
}

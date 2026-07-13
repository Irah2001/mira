package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"mira/internal/http/handlers"
	"mira/internal/notes"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Impossible de localiser la home directory de l'utilisateur", "err", err)
		os.Exit(1)
	}
	miraDir := filepath.Join(homeDir, ".mira")
	_ = os.MkdirAll(miraDir, 0755)
	storePath := filepath.Join(miraDir, "notes.jsonl")

	store := notes.NewJSONLStore(storePath)
	noteHandler := handlers.NewNoteHandler(store)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/notes", noteHandler.Create)
	mux.HandleFunc("GET /api/v1/notes", noteHandler.List)
	mux.HandleFunc("GET /api/v1/notes/{id}", noteHandler.Get)
	mux.HandleFunc("PATCH /api/v1/notes/{id}", noteHandler.Update)
	mux.HandleFunc("DELETE /api/v1/notes/{id}", noteHandler.Delete)
	mux.HandleFunc("GET /api/v1/search", noteHandler.Search)

	fs := http.FileServer(http.Dir("./docs"))
	mux.Handle("/docs/", http.StripPrefix("/docs/", fs))

	handlerPipeline := handlers.Chain(mux,
		handlers.RecoveryMiddleware,
		handlers.RequestIDMiddleware,
		handlers.LoggingMiddleware(logger),
	)

	timeoutServer := http.TimeoutHandler(handlerPipeline, 5*time.Second, `{"success":false,"error":"Le serveur a mis trop de temps à répondre (Timeout)"}`)

	slog.Info("Démarrage du Serveur HTTP Mira v1", "port", "8080", "storage_file", storePath)
	slog.Info("Swagger UI disponible sur", "url", "http://localhost:8080/docs/")
	if err := http.ListenAndServe(":8080", timeoutServer); err != nil {
		slog.Error("Crash du serveur", "err", err)
	}
}

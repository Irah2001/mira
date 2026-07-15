package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"mira/internal/http/handlers"
	"mira/internal/notes"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Warn("Aucun fichier .env trouvé, utilisation des variables d'environnement système")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("La variable DATABASE_URL n'est pas définie")
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("Impossible de créer le pool de connexion", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("La base de données PostgreSQL ne répond pas", "err", err)
		os.Exit(1)
	}
	slog.Info("✅ Connecté à PostgreSQL avec succès")

	store := notes.NewPostgresStore(pool)
	enrichService := notes.NewEnrichmentService(store, 100, 3)
	enrichService.Start(context.Background())
	noteHandler := handlers.NewNoteHandler(store, enrichService)

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

	slog.Info("🚀 Démarrage du Serveur HTTP Mira v1", "port", port)

	if err := http.ListenAndServe(":"+port, timeoutServer); err != nil {
		slog.Error("Crash du serveur", "err", err)
	}
}

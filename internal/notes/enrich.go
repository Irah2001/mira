package notes

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

type EnrichmentJob struct {
	NoteID int64
}

type EnrichmentService struct {
	store      *PostgresStore
	jobQueue   chan EnrichmentJob
	numWorkers int
}

func NewEnrichmentService(store *PostgresStore, queueSize int, numWorkers int) *EnrichmentService {
	return &EnrichmentService{
		store:      store,
		jobQueue:   make(chan EnrichmentJob, queueSize),
		numWorkers: numWorkers,
	}
}

func (s *EnrichmentService) Start(ctx context.Context) {
	slog.Info("Initialisation du pool d'enrichissement", "workers", s.numWorkers, "queue_size", cap(s.jobQueue))
	for i := 1; i <= s.numWorkers; i++ {
		go s.worker(ctx, i)
	}
}

func (s *EnrichmentService) Submit(noteID int64) {
	select {
	case s.jobQueue <- EnrichmentJob{NoteID: noteID}:
		slog.Debug("Tâche d'enrichissement planifiée", "note_id", noteID)
	default:
		slog.Warn("⚠️ File d'attente saturée, job ignoré", "note_id", noteID)
	}
}

func (s *EnrichmentService) worker(ctx context.Context, workerID int) {
	slog.Debug("Worker d'enrichissement démarré", "worker_id", workerID)

	for {
		select {
		case <-ctx.Done():
			slog.Debug("Arrêt du worker", "worker_id", workerID)
			return
		case job, ok := <-s.jobQueue:
			if !ok {
				return
			}
			s.processJobWithTimeout(job.NoteID, workerID)
		}
	}
}

func (s *EnrichmentService) processJobWithTimeout(noteID int64, workerID int) {
	taskCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	slog.Info("Début enrichissement", "note_id", noteID, "worker_id", workerID)

	err := s.enrich(taskCtx, noteID)
	if err != nil {
		slog.Error("❌ Échec d'enrichissement", "note_id", noteID, "err", err, "worker_id", workerID)
		// Écriture du statut 'failed' en BDD
		_ = s.store.UpdateEnrichmentStatus(noteID, "failed", nil, 0, nil)
		return
	}

	slog.Info("✅ Enrichissement réussi", "note_id", noteID, "worker_id", workerID)
}

func (s *EnrichmentService) enrich(ctx context.Context, noteID int64) error {
	note, err := s.store.GetByID(noteID)
	if err != nil {
		return fmt.Errorf("impossible de charger la note : %w", err)
	}

	select {
	case <-time.After(time.Duration(1500+rand.Intn(1500)) * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	summary := fmt.Sprintf("Résumé automatique : La note traite de %q et contient %d caractères.", note.Title, len(note.Content))
	score := len(note.Content) / 10
	if score > 100 {
		score = 100
	}

	discoveredTags := []string{"enrichi"}
	if len(note.Content) > 50 {
		discoveredTags = append(discoveredTags, "long-read")
	}

	err = s.store.UpdateEnrichmentStatus(noteID, "done", &summary, score, discoveredTags)
	if err != nil {
		return fmt.Errorf("impossible de sauvegarder l'enrichissement : %w", err)
	}

	return nil
}

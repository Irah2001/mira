package notes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Add(note *Note) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO notes (title, content) 
		 VALUES ($1, $2) 
		 RETURNING id, created_at, updated_at`,
		note.Title, note.Content,
	).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)

	if err != nil {
		return err
	}

	for _, tagName := range note.Tags {
		var tagID int
		err = tx.QueryRow(ctx,
			`INSERT INTO tags (name) VALUES ($1)
			 ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name
			 RETURNING id`,
			tagName,
		).Scan(&tagID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO note_tags (note_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			note.ID, tagID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresStore) GetByID(id int64) (Note, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var note Note
	query := `
		SELECT n.id, n.title, n.content, n.enrichment_status, n.summary, n.score, n.created_at, n.updated_at, 
		       COALESCE(array_remove(array_agg(t.name), NULL), '{}') as tags
		FROM notes n
		LEFT JOIN note_tags nt ON n.id = nt.note_id
		LEFT JOIN tags t ON nt.tag_id = t.id
		WHERE n.id = $1
		GROUP BY n.id
	`

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&note.ID, &note.Title, &note.Content, &note.EnrichmentStatus, &note.Summary, &note.Score, &note.CreatedAt, &note.UpdatedAt, &note.Tags,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return Note{}, ErrNotFound
		}
		return Note{}, err
	}
	return note, nil
}

func (s *PostgresStore) GetAll() ([]Note, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT n.id, n.title, n.content, n.created_at, n.updated_at, 
		       COALESCE(array_remove(array_agg(t.name), NULL), '{}') as tags
		FROM notes n
		LEFT JOIN note_tags nt ON n.id = nt.note_id
		LEFT JOIN tags t ON nt.tag_id = t.id
		GROUP BY n.id
		ORDER BY n.created_at DESC
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt, &note.Tags); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	return notes, nil
}

func (s *PostgresStore) List(limit int) ([]Note, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT n.id, n.title, n.content, n.created_at, n.updated_at, 
		       COALESCE(array_remove(array_agg(t.name), NULL), '{}') as tags
		FROM notes n
		LEFT JOIN note_tags nt ON n.id = nt.note_id
		LEFT JOIN tags t ON nt.tag_id = t.id
		GROUP BY n.id
		ORDER BY n.created_at DESC
		LIMIT $1
	`

	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt, &note.Tags); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	return notes, nil
}

func (s *PostgresStore) ListPaginated(limit, offset int) ([]Note, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var total int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM notes").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT n.id, n.title, n.content, n.created_at, n.updated_at, 
		       COALESCE(array_remove(array_agg(t.name), NULL), '{}') as tags
		FROM notes n
		LEFT JOIN note_tags nt ON n.id = nt.note_id
		LEFT JOIN tags t ON nt.tag_id = t.id
		GROUP BY n.id
		ORDER BY n.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt, &note.Tags); err != nil {
			return nil, 0, err
		}
		notes = append(notes, note)
	}

	return notes, total, nil
}

func (s *PostgresStore) Update(id int64, payload NotePayload) (Note, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		UPDATE notes SET
			title = COALESCE($1, title),
			content = COALESCE($2, content),
			updated_at = NOW()
		WHERE id = $3
	`

	res, err := s.pool.Exec(ctx, query, payload.Title, payload.Content, id)
	if err != nil {
		return Note{}, err
	}

	if res.RowsAffected() == 0 {
		return Note{}, ErrNotFound
	}

	return s.GetByID(id)
}

func (s *PostgresStore) Delete(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := s.pool.Exec(ctx, "DELETE FROM notes WHERE id = $1", id)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgresStore) UpdateEnrichmentStatus(id int64, status string, summary *string, score int, extraTags []string, embedding []float32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE notes 
		 SET enrichment_status = $1, summary = $2, score = $3, embedding = $4, updated_at = NOW() 
		 WHERE id = $5`,
		status, summary, score, pgvector.NewVector(embedding), id,
	)
	if err != nil {
		return err
	}

	for _, tagName := range extraTags {
		var tagID int
		err = tx.QueryRow(ctx,
			`INSERT INTO tags (name) VALUES ($1)
			 ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name
			 RETURNING id`,
			tagName,
		).Scan(&tagID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO note_tags (note_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			id, tagID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresStore) Search(query string) ([]Note, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	queryVector := []float32{0.5, 0.5, 0.5}

	sqlQuery := `
		SELECT n.id, n.title, n.content, n.enrichment_status, n.summary, n.score, n.created_at, n.updated_at,
		       COALESCE(array_remove(array_agg(t.name), NULL), '{}') as tags
		FROM notes n
		LEFT JOIN note_tags nt ON n.id = nt.note_id
		LEFT JOIN tags t ON nt.tag_id = t.id
		
		-- On filtre grossièrement par texte OU on garde tout pour trier par vecteur
		WHERE n.search_vector @@ websearch_to_tsquery('french', $1) 
		   OR n.embedding IS NOT NULL
		
		GROUP BY n.id, n.search_vector, n.embedding
		
		-- Le TRI HYBRIDE : 70% Textuel + 30% Sémantique
		ORDER BY (
			0.7 * ts_rank(n.search_vector, websearch_to_tsquery('french', $1)) +
			0.3 * (1 - (n.embedding <=> $2::vector))
		) DESC
		LIMIT 20
	`

	rows, err := s.pool.Query(ctx, sqlQuery, query, pgvector.NewVector(queryVector))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Note
	for rows.Next() {
		var note Note
		err := rows.Scan(
			&note.ID, &note.Title, &note.Content, &note.EnrichmentStatus, &note.Summary,
			&note.Score, &note.CreatedAt, &note.UpdatedAt, &note.Tags,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, note)
	}

	return results, nil
}

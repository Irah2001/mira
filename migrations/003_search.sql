-- Activer l'extension pgvector
CREATE EXTENSION IF NOT EXISTS vector;

-- 1. Index Full-Text (Recherche par mots-clés)
ALTER TABLE notes ADD COLUMN search_vector tsvector 
GENERATED ALWAYS AS (to_tsvector('french', coalesce(title, '') || ' ' || coalesce(content, ''))) STORED;

CREATE INDEX IF NOT EXISTS idx_notes_fulltext ON notes USING GIN (search_vector);

-- 2. Index Vectoriel (Similarité sémantique)
-- On utilise des vecteurs de dimension 3 pour notre simulation (les vrais LLM utilisent souvent 1536)
ALTER TABLE notes ADD COLUMN embedding vector(3);

CREATE INDEX IF NOT EXISTS idx_notes_embedding ON notes USING hnsw (embedding vector_cosine_ops);

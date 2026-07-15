-- migrations/002_enrichment.sql

-- Ajout des statuts et données d'enrichissement
ALTER TABLE notes ADD COLUMN enrichment_status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE notes ADD COLUMN summary TEXT;
ALTER TABLE notes ADD COLUMN score INT DEFAULT 0;

-- Index pour accélérer les futures requêtes de filtrage par statut
CREATE INDEX IF NOT EXISTS idx_notes_enrichment_status ON notes(enrichment_status);

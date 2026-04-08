package storage

import (
	"time"

	pgvector "github.com/pgvector/pgvector-go"
)

// ChunkRecord is the DB row representation.
type ChunkRecord struct {
	ID            int64
	DocName       string
	ChunkText     string
	TokenCount    int
	ChunkStrategy string
	Embedding     pgvector.Vector
	CreatedAt     time.Time
}

package storage

import (
	"context"
	"fmt"

	"rag-qa/internal/ingestion"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
)

// ChunkRepository is the interface the pipeline and retriever depend on.
type ChunkRepository interface {
	SaveMany(ctx context.Context, docName string, chunks []ingestion.Chunk, embeddings [][]float32, strategy string) error
	FindSimilar(ctx context.Context, embedding []float32, topK int, strategy string) ([]ChunkRecord, error)
	DeleteByDoc(ctx context.Context, docName string) (int64, error)
}

// PgVectorRepository implements ChunkRepository using pgx + pgvector.
type PgVectorRepository struct {
	pool *pgxpool.Pool
}

func NewPgVectorRepository(pool *pgxpool.Pool) *PgVectorRepository {
	return &PgVectorRepository{pool: pool}
}

func (r *PgVectorRepository) SaveMany(ctx context.Context, docName string, chunks []ingestion.Chunk, embeddings [][]float32, strategy string) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("repository: save many: embeddings/chunks length mismatch: %d/%d", len(chunks), len(embeddings))
	}
	if len(chunks) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for i, chunk := range chunks {
		batch.Queue(
			`INSERT INTO chunks (doc_name, chunk_text, token_count, chunk_strategy, embedding)
             VALUES ($1, $2, $3, $4, $5)`,
			docName,
			chunk.Text,
			chunk.TokenCount,
			strategy,
			pgvector.NewVector(embeddings[i]),
		)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range chunks {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("repository: save chunk: %w", err)
		}
	}

	return nil
}

func (r *PgVectorRepository) FindSimilar(ctx context.Context, embedding []float32, topK int, strategy string) ([]ChunkRecord, error) {
	query := `
        SELECT id, doc_name, chunk_text, token_count, chunk_strategy, embedding, created_at
        FROM chunks
        WHERE ($3 = '' OR chunk_strategy = $3)
        ORDER BY embedding <=> $1
        LIMIT $2`

	rows, err := r.pool.Query(ctx, query, pgvector.NewVector(embedding), topK, strategy)
	if err != nil {
		return nil, fmt.Errorf("repository: find similar: %w", err)
	}
	defer rows.Close()

	var records []ChunkRecord
	for rows.Next() {
		var rec ChunkRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.DocName,
			&rec.ChunkText,
			&rec.TokenCount,
			&rec.ChunkStrategy,
			&rec.Embedding,
			&rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: scan row: %w", err)
		}
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: rows err: %w", err)
	}
	return records, nil
}

func (r *PgVectorRepository) DeleteByDoc(ctx context.Context, docName string) (int64, error) {
	res, err := r.pool.Exec(ctx, "DELETE FROM chunks WHERE doc_name = $1", docName)
	if err != nil {
		return 0, fmt.Errorf("repository: delete by doc: %w", err)
	}
	return res.RowsAffected(), nil
}

var _ ChunkRepository = (*PgVectorRepository)(nil)

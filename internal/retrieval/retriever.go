package retrieval

import (
	"context"
	"fmt"

	"rag-qa/internal/storage"
)

// RetrievedChunk is what the generator receives — decoupled from the DB model.
type RetrievedChunk struct {
	Text       string
	DocName    string
	Similarity float32
	Strategy   string
}

// Retriever is the interface the query handler depends on.
type Retriever interface {
	Retrieve(ctx context.Context, queryEmbedding []float32, topK int, strategy string) ([]RetrievedChunk, error)
}

// PgVectorRetriever implements Retriever.
type PgVectorRetriever struct {
	repo storage.ChunkRepository
}

func NewPgVectorRetriever(repo storage.ChunkRepository) *PgVectorRetriever {
	return &PgVectorRetriever{repo: repo}
}

func (r *PgVectorRetriever) Retrieve(ctx context.Context, queryEmbedding []float32, topK int, strategy string) ([]RetrievedChunk, error) {
	records, err := r.repo.FindSimilar(ctx, queryEmbedding, topK, strategy)
	if err != nil {
		return nil, fmt.Errorf("retriever: %w", err)
	}

	chunks := make([]RetrievedChunk, len(records))
	for i, rec := range records {
		chunks[i] = RetrievedChunk{
			Text:     rec.ChunkText,
			DocName:  rec.DocName,
			Strategy: rec.ChunkStrategy,
		}
	}
	return chunks, nil
}

var _ Retriever = (*PgVectorRetriever)(nil)

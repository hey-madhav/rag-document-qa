package ingestion

import (
	"context"
	"fmt"
)

// IngestionPipeline orchestrates chunking, embedding, and persistence.
type IngestionPipeline struct {
	chunker  Chunker
	embedder Embedder
	repo     ChunkSaver
}

func NewIngestionPipeline(chunker Chunker, embedder Embedder, repo ChunkSaver) *IngestionPipeline {
	return &IngestionPipeline{
		chunker:  chunker,
		embedder: embedder,
		repo:     repo,
	}
}

func (p *IngestionPipeline) StrategyName() string {
	return p.chunker.StrategyName()
}

func (p *IngestionPipeline) Ingest(ctx context.Context, docName, text string) (int, error) {
	chunks, err := p.chunker.Chunk(text)
	if err != nil {
		return 0, fmt.Errorf("pipeline: chunk: %w", err)
	}
	if len(chunks) == 0 {
		return 0, nil
	}

	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}

	embeddings, err := p.embedder.Embed(ctx, texts)
	if err != nil {
		return 0, fmt.Errorf("pipeline: embed: %w", err)
	}

	strategy := p.chunker.StrategyName()
	if err := p.repo.SaveMany(ctx, docName, chunks, embeddings, strategy); err != nil {
		return 0, fmt.Errorf("pipeline: save: %w", err)
	}

	return len(chunks), nil
}

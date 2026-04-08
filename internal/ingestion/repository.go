package ingestion

import "context"

// ChunkSaver is the minimal persistence capability required by the ingestion pipeline.
// Keeping this interface small avoids Go import cycles between ingestion <-> storage.
type ChunkSaver interface {
	SaveMany(ctx context.Context, docName string, chunks []Chunk, embeddings [][]float32, strategy string) error
}

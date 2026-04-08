package retrieval

import "context"

// Reranker is a stub interface — intentional extension point.
// A cross-encoder (e.g. ms-marco-MiniLM) would implement this.
type Reranker interface {
	Rerank(ctx context.Context, query string, chunks []RetrievedChunk) ([]RetrievedChunk, error)
}

package unit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rag-qa/internal/ingestion"
	"rag-qa/internal/retrieval"
	"rag-qa/internal/storage"
)

type mockChunkRepository struct {
	lastTopK      int
	lastStrategy  string
	lastEmbedding []float32
	returnRows    []storage.ChunkRecord
}

func (m *mockChunkRepository) SaveMany(_ context.Context, _ string, _ []ingestion.Chunk, _ [][]float32, _ string) error {
	return nil
}

func (m *mockChunkRepository) FindSimilar(ctx context.Context, embedding []float32, topK int, strategy string) ([]storage.ChunkRecord, error) {
	_ = ctx
	m.lastTopK = topK
	m.lastStrategy = strategy
	m.lastEmbedding = embedding
	return m.returnRows, nil
}

func (m *mockChunkRepository) DeleteByDoc(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func TestPgVectorRetriever_MapsRecords(t *testing.T) {
	mock := &mockChunkRepository{
		returnRows: []storage.ChunkRecord{
			{
				DocName:       "policy.pdf",
				ChunkText:     "Refunds within 30 days.",
				ChunkStrategy: "token_256",
			},
		},
	}

	r := retrieval.NewPgVectorRetriever(mock)

	out, err := r.Retrieve(context.Background(), []float32{0.1, 0.2}, 3, "token_256")
	require.NoError(t, err)
	require.Len(t, out, 1)

	assert.Equal(t, 3, mock.lastTopK)
	assert.Equal(t, "token_256", mock.lastStrategy)
	assert.Equal(t, "policy.pdf", out[0].DocName)
	assert.Equal(t, "Refunds within 30 days.", out[0].Text)
	assert.Equal(t, "token_256", out[0].Strategy)
}

// Interface compliance — compile-time check.
var _ storage.ChunkRepository = (*mockChunkRepository)(nil)
var _ retrieval.Retriever = (*retrieval.PgVectorRetriever)(nil)

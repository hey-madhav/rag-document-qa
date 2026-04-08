package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rag-qa/internal/api/handler"
	"rag-qa/internal/generation"
	"rag-qa/internal/ingestion"
	"rag-qa/internal/retrieval"
)

type mockEmbedder struct {
	embeddings [][]float32
}

func (m *mockEmbedder) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return m.embeddings, nil
}

func (m *mockEmbedder) Dimensions() int { return 1536 }

type mockRetriever struct {
	chunks []retrieval.RetrievedChunk
}

func (m *mockRetriever) Retrieve(_ context.Context, _ []float32, _ int, _ string) ([]retrieval.RetrievedChunk, error) {
	return m.chunks, nil
}

type mockGenerator struct {
	result *generation.GenerationResult
}

func (m *mockGenerator) Generate(_ context.Context, _ string, _ []retrieval.RetrievedChunk) (*generation.GenerationResult, error) {
	return m.result, nil
}

func TestQueryHandler_ReturnsSourcesAndUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewQueryHandler(
		&mockEmbedder{embeddings: [][]float32{{0.1, 0.2}}},
		&mockRetriever{chunks: []retrieval.RetrievedChunk{{Text: "Refunds", DocName: "policy.pdf", Strategy: "token_256"}}},
		&mockGenerator{result: &generation.GenerationResult{
			Answer:           "Refunds within 30 days.",
			Sources:          []string{"policy.pdf"},
			Model:            "gpt-test",
			PromptTokens:     12,
			CompletionTokens: 34,
		}},
	)

	r := gin.New()
	r.POST("/ask", h.Handle)

	body := map[string]any{
		"query":          "What is the return policy?",
		"top_k":          4,
		"chunk_strategy": "token_256",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var parsed map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Refunds within 30 days.", parsed["answer"])
	assert.Equal(t, []any{"policy.pdf"}, parsed["sources"])
	assert.Equal(t, "gpt-test", parsed["model"])

	usage := parsed["usage"].(map[string]any)
	assert.Equal(t, float64(12), usage["prompt_tokens"])
	assert.Equal(t, float64(34), usage["completion_tokens"])
}

// Interface compliance — compile-time checks.
var _ ingestion.Embedder = (*mockEmbedder)(nil)
var _ retrieval.Retriever = (*mockRetriever)(nil)
var _ generation.Generator = (*mockGenerator)(nil)

var _ generation.Generator = (*generation.GeminiGenerator)(nil)

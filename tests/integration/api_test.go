package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"rag-qa/internal/generation"
	"rag-qa/internal/ingestion"
	"rag-qa/internal/retrieval"
	"rag-qa/internal/storage"
)

func TestIngestAndAsk(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("set INTEGRATION_TESTS=1 to run")
	}
	baseURL := os.Getenv("INTEGRATION_BASE_URL")
	if baseURL == "" {
		t.Skip("set INTEGRATION_BASE_URL, e.g. http://localhost:8080")
	}

	client := &http.Client{Timeout: 60 * time.Second}

	docName := "test-doc.txt"
	content := "This is a test document. Refunds are available within 30 days. Contact support by email."

	// Ingest
	ingestReq := map[string]any{
		"doc_name":       docName,
		"content":        content,
		"chunk_strategy": "token_256",
	}
	b, err := json.Marshal(ingestReq)
	require.NoError(t, err)

	ingestHTTPReq, err := http.NewRequest(http.MethodPost, baseURL+"/ingest", bytes.NewReader(b))
	require.NoError(t, err)
	ingestHTTPReq.Header.Set("Content-Type", "application/json")

	ingestResp, err := client.Do(ingestHTTPReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, ingestResp.StatusCode)
	_ = ingestResp.Body.Close()

	// Ask
	askReq := map[string]any{
		"query":          "What is the return policy?",
		"top_k":          4,
		"chunk_strategy": "token_256",
	}
	b2, err := json.Marshal(askReq)
	require.NoError(t, err)

	askHTTPReq, err := http.NewRequest(http.MethodPost, baseURL+"/ask", bytes.NewReader(b2))
	require.NoError(t, err)
	askHTTPReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	askResp, err := client.Do(askHTTPReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, askResp.StatusCode)
	_ = askResp.Body.Close()
	require.Less(t, time.Since(start), 30*time.Second)
}

// Interface compliance — compile-time checks.
var _ ingestion.Chunker = (*ingestion.TokenChunker)(nil)
var _ retrieval.Retriever = (*retrieval.PgVectorRetriever)(nil)
var _ generation.Generator = (*generation.GeminiGenerator)(nil)
var _ storage.ChunkRepository = (*storage.PgVectorRepository)(nil)

package ingestion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Embedder converts text slices into float32 embedding vectors.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
}

// GeminiEmbedder implements Embedder using Gemini embeddings API.
type GeminiEmbedder struct {
	httpClient *http.Client
	apiKey     string
	model      string
}

func NewGeminiEmbedder(apiKey, model string) *GeminiEmbedder {
	return &GeminiEmbedder{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		model:      model,
	}
}

func (e *GeminiEmbedder) Dimensions() int {
	// Keep 1536 to match current pgvector schema.
	return 1536
}

func (e *GeminiEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	const batchSize = 100
	if len(texts) == 0 {
		return nil, nil
	}

	batches := make([][]string, 0, (len(texts)+batchSize-1)/batchSize)
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batches = append(batches, texts[i:end])
	}

	type result struct {
		index      int
		embeddings [][]float32
		err        error
	}

	results := make([]result, len(batches))
	var wg sync.WaitGroup
	wg.Add(len(batches))

	for batchIdx, batch := range batches {
		batchIdx := batchIdx
		batch := batch
		go func() {
			defer wg.Done()
			embs := make([][]float32, 0, len(batch))
			for _, text := range batch {
				embedding, err := e.embedOne(ctx, text)
				if err != nil {
					results[batchIdx] = result{
						index: batchIdx,
						err:   fmt.Errorf("embedder: batch %d: %w", batchIdx, err),
					}
					return
				}
				embs = append(embs, embedding)
			}

			results[batchIdx] = result{
				index:      batchIdx,
				embeddings: embs,
			}
		}()
	}

	wg.Wait()

	all := make([][]float32, 0, len(texts))
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		all = append(all, r.embeddings...)
	}

	return all, nil
}

type geminiEmbedRequest struct {
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
	TaskType             string `json:"taskType"`
	OutputDimensionality int    `json:"outputDimensionality"`
}

type geminiEmbedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

func (e *GeminiEmbedder) embedOne(ctx context.Context, text string) ([]float32, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("gemini embedder: missing API key")
	}

	u := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s",
		url.PathEscape(e.model),
		url.QueryEscape(e.apiKey),
	)

	var reqBody geminiEmbedRequest
	reqBody.Content.Parts = []struct {
		Text string `json:"text"`
	}{{Text: text}}
	reqBody.TaskType = "RETRIEVAL_DOCUMENT"
	reqBody.OutputDimensionality = e.Dimensions()

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("gemini embedder: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("gemini embedder: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini embedder: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini embedder: read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gemini embedder: status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed geminiEmbedResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("gemini embedder: parse response: %w", err)
	}
	if len(parsed.Embedding.Values) == 0 {
		return nil, fmt.Errorf("gemini embedder: empty embedding returned")
	}
	return parsed.Embedding.Values, nil
}

var _ Embedder = (*GeminiEmbedder)(nil)

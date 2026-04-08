package evaluation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TestCase struct {
	Query            string
	ExpectedKeywords []string
}

type StrategyResult struct {
	Strategy   string
	AvgHitRate float64
	AvgLatency time.Duration
	N          int
}

type askRequest struct {
	Query         string `json:"query"`
	TopK          int    `json:"top_k,omitempty"`
	ChunkStrategy string `json:"chunk_strategy,omitempty"`
}

type askResponse struct {
	Answer string `json:"answer"`
}

// RunChunkingExperiment runs both token strategies against POST /ask and computes metrics.
func RunChunkingExperiment(ctx context.Context, client *http.Client, baseURL string, testCases []TestCase, strategies []string, topK int) (map[string]*StrategyResult, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	results := make(map[string]*StrategyResult, len(strategies))

	for _, strategy := range strategies {
		var totalHit float64
		var totalLatency time.Duration

		for _, tc := range testCases {
			start := time.Now()

			answer, err := queryAPI(ctx, client, baseURL, tc.Query, topK, strategy)
			if err != nil {
				return nil, fmt.Errorf("experiment: query %q (%s): %w", tc.Query, strategy, err)
			}

			elapsed := time.Since(start)
			totalHit += KeywordHitRate(answer, tc.ExpectedKeywords)
			totalLatency += elapsed
		}

		n := len(testCases)
		results[strategy] = &StrategyResult{
			Strategy:   strategy,
			AvgHitRate: totalHit / float64(n),
			AvgLatency: totalLatency / time.Duration(n),
			N:          n,
		}
	}

	return results, nil
}

func queryAPI(ctx context.Context, client *http.Client, baseURL, query string, topK int, strategy string) (string, error) {
	reqBody := askRequest{
		Query:         query,
		TopK:          topK,
		ChunkStrategy: strategy,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("experiment: marshal ask request: %w", err)
	}

	url := baseURL + "/ask"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("experiment: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("experiment: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("experiment: ask failed: status=%d", resp.StatusCode)
	}

	var parsed askResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("experiment: decode response: %w", err)
	}
	return parsed.Answer, nil
}

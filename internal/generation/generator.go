package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"rag-qa/internal/retrieval"
)

const systemPrompt = `You are a precise document assistant.
Answer using ONLY the provided context.
If the context does not contain enough information, say so — do not hallucinate.
Cite your sources at the end of your answer.`

// GenerationResult holds the LLM response and token usage.
type GenerationResult struct {
	Answer           string
	Sources          []string
	Model            string
	PromptTokens     int
	CompletionTokens int
}

// Generator is the interface the query handler depends on.
type Generator interface {
	Generate(ctx context.Context, query string, chunks []retrieval.RetrievedChunk) (*GenerationResult, error)
}

// GeminiGenerator implements Generator using Gemini generateContent API.
type GeminiGenerator struct {
	httpClient *http.Client
	apiKey     string
	model      string
}

func NewGeminiGenerator(apiKey, model string) *GeminiGenerator {
	return &GeminiGenerator{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		apiKey:     apiKey,
		model:      model,
	}
}

func (g *GeminiGenerator) Generate(ctx context.Context, query string, chunks []retrieval.RetrievedChunk) (*GenerationResult, error) {
	var sb strings.Builder

	seenSources := make(map[string]struct{}, len(chunks))
	sources := make([]string, 0, len(chunks))

	for _, c := range chunks {
		fmt.Fprintf(&sb, "[Source: %s]\n%s\n\n---\n\n", c.DocName, c.Text)
		if _, seen := seenSources[c.DocName]; !seen {
			seenSources[c.DocName] = struct{}{}
			sources = append(sources, c.DocName)
		}
	}

	answer, promptTokens, completionTokens, err := g.generateContent(
		ctx,
		fmt.Sprintf("Context:\n%s\nQuestion: %s", sb.String(), query),
	)
	if err != nil {
		return nil, fmt.Errorf("generator: generate content: %w", err)
	}

	return &GenerationResult{
		Answer:           answer,
		Sources:          sources,
		Model:            g.model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	}, nil
}

type geminiGenerateRequest struct {
	SystemInstruction struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"system_instruction"`
	Contents []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	GenerationConfig struct {
		Temperature float32 `json:"temperature"`
	} `json:"generationConfig"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func (g *GeminiGenerator) generateContent(ctx context.Context, prompt string) (string, int, int, error) {
	if g.apiKey == "" {
		return "", 0, 0, fmt.Errorf("gemini generator: missing API key")
	}
	u := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		url.PathEscape(g.model),
		url.QueryEscape(g.apiKey),
	)

	var reqBody geminiGenerateRequest
	reqBody.SystemInstruction.Parts = []struct {
		Text string `json:"text"`
	}{{Text: systemPrompt}}
	reqBody.Contents = []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{
		{
			Role: "user",
			Parts: []struct {
				Text string `json:"text"`
			}{{Text: prompt}},
		},
	}
	reqBody.GenerationConfig.Temperature = 0.1

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, 0, fmt.Errorf("gemini generator: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, 0, fmt.Errorf("gemini generator: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", 0, 0, fmt.Errorf("gemini generator: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, fmt.Errorf("gemini generator: read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return "", 0, 0, fmt.Errorf("gemini generator: status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", 0, 0, fmt.Errorf("gemini generator: parse response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", 0, 0, fmt.Errorf("gemini generator: no candidates returned")
	}

	return parsed.Candidates[0].Content.Parts[0].Text,
		parsed.UsageMetadata.PromptTokenCount,
		parsed.UsageMetadata.CandidatesTokenCount,
		nil
}

var _ Generator = (*GeminiGenerator)(nil)

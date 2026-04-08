package ingestion

import (
	"errors"
	"fmt"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// Chunk is a text segment produced by a chunking strategy.
type Chunk struct {
	Text       string
	TokenCount int
	StartIndex int
}

// Chunker is the abstraction for all text segmentation strategies.
type Chunker interface {
	Chunk(text string) ([]Chunk, error)
	StrategyName() string
}

// TokenChunker implements Chunker with a fixed-size token window + overlap.
type TokenChunker struct {
	maxTokens int
	overlap   int
	encoding  string
}

func NewTokenChunker(maxTokens, overlap int, encoding string) *TokenChunker {
	return &TokenChunker{maxTokens: maxTokens, overlap: overlap, encoding: encoding}
}

func (c *TokenChunker) StrategyName() string {
	return fmt.Sprintf("token_%d", c.maxTokens)
}

func (c *TokenChunker) Chunk(text string) ([]Chunk, error) {
	if c.maxTokens <= 0 {
		return nil, fmt.Errorf("chunker: maxTokens must be > 0")
	}
	if c.overlap < 0 || c.overlap >= c.maxTokens {
		return nil, fmt.Errorf("chunker: overlap must be >= 0 and < maxTokens")
	}

	enc, err := tiktoken.GetEncoding(c.encoding)
	if err != nil {
		return nil, fmt.Errorf("chunker: get encoding: %w", err)
	}

	tokens := enc.Encode(text, nil, nil)
	var chunks []Chunk

	step := c.maxTokens - c.overlap
	start := 0
	for start < len(tokens) {
		end := start + c.maxTokens
		if end > len(tokens) {
			end = len(tokens)
		}
		segment := tokens[start:end]
		chunks = append(chunks, Chunk{
			Text:       enc.Decode(segment),
			TokenCount: len(segment),
			StartIndex: start,
		})
		start += step
	}

	return chunks, nil
}

// SentenceChunker is a stub — intentional extension point.
type SentenceChunker struct{}

func (s *SentenceChunker) StrategyName() string { return "sentence" }

func (s *SentenceChunker) Chunk(_ string) ([]Chunk, error) {
	return nil, errors.New("SentenceChunker: not yet implemented")
}

var _ Chunker = (*TokenChunker)(nil)
var _ Chunker = (*SentenceChunker)(nil)

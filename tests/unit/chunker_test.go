package unit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rag-qa/internal/ingestion"
)

func TestTokenChunker_ChunkCount(t *testing.T) {
	text := strings.Repeat("word ", 600)
	chunker := ingestion.NewTokenChunker(256, 20, "cl100k_base")

	chunks, err := chunker.Chunk(text)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1)

	for _, c := range chunks {
		assert.LessOrEqual(t, c.TokenCount, 256)
	}
}

func TestTokenChunker_Overlap(t *testing.T) {
	text := strings.Repeat("word ", 300)
	chunker := ingestion.NewTokenChunker(100, 20, "cl100k_base")

	chunks, err := chunker.Chunk(text)
	require.NoError(t, err)
	// Second chunk starts at index 80 (100 - 20 overlap)
	if assert.GreaterOrEqual(t, len(chunks), 2) {
		assert.Equal(t, 80, chunks[1].StartIndex)
	}
}

func TestTokenChunker_StrategyName(t *testing.T) {
	assert.Equal(t, "token_256", ingestion.NewTokenChunker(256, 20, "cl100k_base").StrategyName())
	assert.Equal(t, "token_512", ingestion.NewTokenChunker(512, 40, "cl100k_base").StrategyName())
}

// Interface compliance — compile-time check.
var _ ingestion.Chunker = (*ingestion.TokenChunker)(nil)
var _ ingestion.Chunker = (*ingestion.SentenceChunker)(nil)

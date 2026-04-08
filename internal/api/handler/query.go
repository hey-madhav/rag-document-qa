package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"rag-qa/internal/generation"
	"rag-qa/internal/ingestion"
	"rag-qa/internal/retrieval"
)

type QueryRequest struct {
	Query         string `json:"query" validate:"required,min=3,max=1000"`
	TopK          int    `json:"top_k" validate:"omitempty,min=1,max=10"`
	ChunkStrategy string `json:"chunk_strategy" validate:"omitempty,oneof=token_256 token_512"`
}

type QueryResponse struct {
	Answer  string         `json:"answer"`
	Sources []string       `json:"sources"`
	Model   string         `json:"model"`
	Usage   map[string]int `json:"usage"`
}

type QueryHandler struct {
	embedder  ingestion.Embedder
	retriever retrieval.Retriever
	generator generation.Generator
	validate  *validator.Validate
}

func NewQueryHandler(e ingestion.Embedder, r retrieval.Retriever, g generation.Generator) *QueryHandler {
	return &QueryHandler{
		embedder:  e,
		retriever: r,
		generator: g,
		validate:  validator.New(),
	}
}

func (h *QueryHandler) Handle(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("query request bind failed: method=%s path=%s error=%q", c.Request.Method, c.Request.URL.Path, err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		log.Printf("query request validation failed: method=%s path=%s error=%q", c.Request.Method, c.Request.URL.Path, err.Error())
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	if req.TopK == 0 {
		req.TopK = 4
	}

	ctx := c.Request.Context()

	embeddings, err := h.embedder.Embed(ctx, []string{req.Query})
	if err != nil {
		log.Printf(
			"critical query embedding error: method=%s path=%s top_k=%d chunk_strategy=%s query_len=%d error=%q",
			c.Request.Method,
			c.Request.URL.Path,
			req.TopK,
			req.ChunkStrategy,
			len(req.Query),
			err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "embedding failed"})
		return
	}
	if len(embeddings) == 0 {
		log.Printf(
			"critical query embedding empty result: method=%s path=%s top_k=%d chunk_strategy=%s query_len=%d",
			c.Request.Method,
			c.Request.URL.Path,
			req.TopK,
			req.ChunkStrategy,
			len(req.Query),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "embedding failed: empty embeddings"})
		return
	}

	chunks, err := h.retriever.Retrieve(ctx, embeddings[0], req.TopK, req.ChunkStrategy)
	if err != nil {
		log.Printf(
			"critical query retrieval error: method=%s path=%s top_k=%d chunk_strategy=%s error=%q",
			c.Request.Method,
			c.Request.URL.Path,
			req.TopK,
			req.ChunkStrategy,
			err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "retrieval failed"})
		return
	}

	result, err := h.generator.Generate(ctx, req.Query, chunks)
	if err != nil {
		log.Printf(
			"critical query generation error: method=%s path=%s top_k=%d chunk_strategy=%s chunks_count=%d error=%q",
			c.Request.Method,
			c.Request.URL.Path,
			req.TopK,
			req.ChunkStrategy,
			len(chunks),
			err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generation failed"})
		return
	}

	c.JSON(http.StatusOK, QueryResponse{
		Answer:  result.Answer,
		Sources: result.Sources,
		Model:   result.Model,
		Usage: map[string]int{
			"prompt_tokens":     result.PromptTokens,
			"completion_tokens": result.CompletionTokens,
		},
	})
}

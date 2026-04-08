package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"rag-qa/internal/ingestion"
)

type IngestRequest struct {
	DocName       string `json:"doc_name" validate:"required,min=1,max=512"`
	Content       string `json:"content" validate:"required,min=10"`
	ChunkStrategy string `json:"chunk_strategy" validate:"omitempty,oneof=token_256 token_512"`
}

type IngestResponse struct {
	DocName       string `json:"doc_name"`
	ChunksCreated int    `json:"chunks_created"`
	Strategy      string `json:"strategy"`
}

type IngestHandler struct {
	pipeline256 *ingestion.IngestionPipeline
	pipeline512 *ingestion.IngestionPipeline
	validate    *validator.Validate
}

func NewIngestHandler(p256, p512 *ingestion.IngestionPipeline) *IngestHandler {
	return &IngestHandler{
		pipeline256: p256,
		pipeline512: p512,
		validate:    validator.New(),
	}
}

func (h *IngestHandler) Handle(c *gin.Context) {
	var req IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	pipeline := h.pipeline256
	strategy := pipeline.StrategyName()
	if req.ChunkStrategy == "token_512" {
		pipeline = h.pipeline512
		strategy = pipeline.StrategyName()
	} else if req.ChunkStrategy == "token_256" {
		pipeline = h.pipeline256
		strategy = pipeline.StrategyName()
	}

	n, err := pipeline.Ingest(c.Request.Context(), req.DocName, req.Content)
	if err != nil {
		log.Printf(
			"critical ingest error: method=%s path=%s strategy=%s doc_name=%q content_len=%d error=%q",
			c.Request.Method,
			c.Request.URL.Path,
			strategy,
			req.DocName,
			len(req.Content),
			err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, IngestResponse{
		DocName:       req.DocName,
		ChunksCreated: n,
		Strategy:      strategy,
	})
}

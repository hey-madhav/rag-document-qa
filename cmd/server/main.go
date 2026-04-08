package main

import (
	"context"
	"log"

	"rag-qa/internal/api"
	"rag-qa/internal/api/handler"
	"rag-qa/internal/config"
	"rag-qa/internal/generation"
	"rag-qa/internal/ingestion"
	"rag-qa/internal/retrieval"
	"rag-qa/internal/storage"
)

func main() {
	cfg := config.Load()

	pool, err := storage.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("server: %v", err)
	}
	defer pool.Close()

	repo := storage.NewPgVectorRepository(pool)

	embedder := ingestion.NewGeminiEmbedder(cfg.GeminiAPIKey, cfg.EmbeddingModel)

	// API validation and DB strategy names are fixed to token_256 and token_512.
	// Overlap is configurable, while chunk sizes remain 256/512 for the experiment.
	max256 := 256
	overlap256 := cfg.DefaultOverlap
	max512 := 512
	overlap512 := overlap256 * 2

	pipeline256 := ingestion.NewIngestionPipeline(
		ingestion.NewTokenChunker(max256, overlap256, cfg.TokenEncoding),
		embedder,
		repo,
	)
	pipeline512 := ingestion.NewIngestionPipeline(
		ingestion.NewTokenChunker(max512, overlap512, cfg.TokenEncoding),
		embedder,
		repo,
	)

	retriever := retrieval.NewPgVectorRetriever(repo)
	generator := generation.NewGeminiGenerator(cfg.GeminiAPIKey, cfg.LLMModel)

	ingestHandler := handler.NewIngestHandler(pipeline256, pipeline512)
	queryHandler := handler.NewQueryHandler(embedder, retriever, generator)

	r := api.NewRouter(ingestHandler, queryHandler)
	log.Printf("server starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}

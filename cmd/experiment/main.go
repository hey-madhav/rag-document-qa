package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"rag-qa/internal/config"
	"rag-qa/internal/evaluation"
)

func main() {
	cfg := config.Load()

	baseURL := os.Getenv("EXPERIMENT_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + cfg.Port
	}

	testCases := []evaluation.TestCase{
		{Query: "What is the return policy?", ExpectedKeywords: []string{"30 days", "receipt", "refund"}},
		{Query: "How do I contact support?", ExpectedKeywords: []string{"email", "phone", "support"}},
		{Query: "What are the shipping options?", ExpectedKeywords: []string{"standard", "express", "overnight"}},
		// Add 15-20 domain-specific cases for your sample docs.
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	results, err := evaluation.RunChunkingExperiment(
		ctx,
		&http.Client{Timeout: 60 * time.Second},
		baseURL,
		testCases,
		[]string{"token_256", "token_512"},
		cfg.DefaultTopK,
	)
	if err != nil {
		fmt.Printf("experiment failed: %v\n", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf("experiment marshal failed: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("experiment_results.json", out, 0644); err != nil {
		fmt.Printf("experiment write failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Results written to experiment_results.json")
}

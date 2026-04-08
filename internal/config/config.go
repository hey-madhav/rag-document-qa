package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all runtime configuration loaded from a YAML file.
type Config struct {
	GeminiAPIKey     string `yaml:"gemini_api_key"`
	DatabaseURL      string `yaml:"database_url"`
	EmbeddingModel   string `yaml:"embedding_model"`
	LLMModel         string `yaml:"llm_model"`
	TokenEncoding    string `yaml:"token_encoding"`
	DefaultChunkSize int    `yaml:"default_chunk_size"`
	DefaultOverlap   int    `yaml:"default_overlap"`
	DefaultTopK      int    `yaml:"default_top_k"`
	Port             string `yaml:"port"`
}

// Load reads configuration from a YAML file, defaulting to ./config.yml.
// The path can be overridden via the CONFIG_FILE environment variable.
func Load() *Config {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = "config.yml"
	}

	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("config: unable to open %q: %v", path, err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("config: unable to parse %q: %v", path, err)
	}

	return &cfg
}

package embeddings

import "context"

// Embedder generates vector embeddings for text
type Embedder interface {
	// Embed generates an embedding vector for a single text
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts efficiently
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the size of the embedding vectors
	Dimensions() int

	// Name returns the name/model of this embedder
	Name() string
}

// Config holds configuration for creating an embedder
type Config struct {
	Provider string

	// Ollama config
	OllamaURL   string
	OllamaModel string

	// OpenAI config
	OpenAIKey   string
	OpenAIModel string
}

// NewEmbedder creates an embedder based on the config
func NewEmbedder(cfg Config) (Embedder, error) {
	switch cfg.Provider {
	case "ollama":
		return NewOllamaEmbedder(cfg.OllamaURL, cfg.OllamaModel)
	case "openai":
		return NewOpenAIEmbedder(cfg.OpenAIKey, cfg.OpenAIModel)
	default:
		return nil, nil // No embedder for keyword-only
	}
}

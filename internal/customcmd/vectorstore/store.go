package vectorstore

import "context"

// Store manages vector embeddings and similarity search
type Store interface {
	// Add stores a vector with associated metadata
	Add(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error

	// Search finds the top K most similar vectors
	Search(ctx context.Context, query []float32, topK int) ([]SearchResult, error)

	// Delete removes a vector by ID
	Delete(ctx context.Context, id string) error

	// Clear removes all vectors
	Clear(ctx context.Context) error

	// Count returns the number of stored vectors
	Count() int
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	ID       string
	Score    float32 // Cosine similarity (0-1, higher is better)
	Metadata map[string]interface{}
}

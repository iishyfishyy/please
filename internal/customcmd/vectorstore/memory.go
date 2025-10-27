package vectorstore

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
)

// MemoryStore is an in-memory vector store using cosine similarity
type MemoryStore struct {
	vectors  map[string][]float32
	metadata map[string]map[string]interface{}
	mu       sync.RWMutex
}

// NewMemoryStore creates a new in-memory vector store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		vectors:  make(map[string][]float32),
		metadata: make(map[string]map[string]interface{}),
	}
}

// Add stores a vector with metadata
func (m *MemoryStore) Add(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	if len(vector) == 0 {
		return fmt.Errorf("empty vector")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.vectors[id] = vector
	m.metadata[id] = metadata

	return nil
}

// Search finds the top K most similar vectors using cosine similarity
func (m *MemoryStore) Search(ctx context.Context, query []float32, topK int) ([]SearchResult, error) {
	if len(query) == 0 {
		return nil, fmt.Errorf("empty query vector")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.vectors) == 0 {
		return []SearchResult{}, nil
	}

	// Calculate similarity scores
	type scoredResult struct {
		id    string
		score float32
	}

	results := make([]scoredResult, 0, len(m.vectors))

	for id, vector := range m.vectors {
		score := cosineSimilarity(query, vector)
		results = append(results, scoredResult{id: id, score: score})
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Take top K
	k := topK
	if k > len(results) {
		k = len(results)
	}

	searchResults := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		searchResults[i] = SearchResult{
			ID:       results[i].id,
			Score:    results[i].score,
			Metadata: m.metadata[results[i].id],
		}
	}

	return searchResults, nil
}

// Delete removes a vector by ID
func (m *MemoryStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.vectors, id)
	delete(m.metadata, id)

	return nil
}

// Clear removes all vectors
func (m *MemoryStore) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.vectors = make(map[string][]float32)
	m.metadata = make(map[string]map[string]interface{})

	return nil
}

// Count returns the number of stored vectors
func (m *MemoryStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.vectors)
}

// cosineSimilarity calculates cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 means identical direction
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

package customcmd

import (
	"context"
	"fmt"

	"github.com/iishyfishyy/please/internal/customcmd/embeddings"
	"github.com/iishyfishyy/please/internal/customcmd/vectorstore"
)

// SemanticMatcher performs semantic search using embeddings
type SemanticMatcher struct {
	embedder    embeddings.Embedder
	vectorStore vectorstore.Store
	indexed     bool
}

// NewSemanticMatcher creates a new semantic matcher
func NewSemanticMatcher(embedder embeddings.Embedder, store vectorstore.Store) *SemanticMatcher {
	if store == nil {
		store = vectorstore.NewMemoryStore() // Fallback to in-memory
	}

	return &SemanticMatcher{
		embedder:    embedder,
		vectorStore: store,
		indexed:     false,
	}
}

// Index creates embeddings for all command documents
func (s *SemanticMatcher) Index(ctx context.Context, docs []CommandDoc) error {
	if s.embedder == nil {
		return fmt.Errorf("no embedder configured")
	}

	// Clear existing vectors
	s.vectorStore.Clear(ctx)

	// Create embeddings for each document
	for _, doc := range docs {
		// Combine command name, keywords, and examples into searchable text
		searchText := s.buildSearchText(doc)

		// Generate embedding
		embedding, err := s.embedder.Embed(ctx, searchText)
		if err != nil {
			return fmt.Errorf("failed to embed doc %s: %w", doc.Command, err)
		}

		// Store with metadata
		metadata := map[string]interface{}{
			"command":    doc.Command,
			"filename":   doc.Filename,
			"file_mtime": doc.UpdatedAt.Unix(), // For cache validation
		}

		id := fmt.Sprintf("cmd_%s", doc.Command)
		if err := s.vectorStore.Add(ctx, id, embedding, metadata); err != nil {
			return fmt.Errorf("failed to store embedding for %s: %w", doc.Command, err)
		}
	}

	s.indexed = true

	// Update indexed_at timestamp if using SQLiteStore
	if sqlStore, ok := s.vectorStore.(*vectorstore.SQLiteStore); ok {
		sqlStore.UpdateIndexTime()
	}

	return nil
}

// Search finds relevant documents using semantic similarity
func (s *SemanticMatcher) Search(ctx context.Context, query string, topK int) ([]CommandDoc, []float32, error) {
	if !s.indexed {
		return nil, nil, fmt.Errorf("not indexed")
	}

	// Generate embedding for query
	queryEmbed, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search vector store
	results, err := s.vectorStore.Search(ctx, queryEmbed, topK)
	if err != nil {
		return nil, nil, fmt.Errorf("search failed: %w", err)
	}

	// For now, return empty since we need to map back to docs
	// This would require storing doc references in metadata
	scores := make([]float32, len(results))
	for i, result := range results {
		scores[i] = result.Score
	}

	return nil, scores, nil
}

// buildSearchText creates searchable text from a document
func (s *SemanticMatcher) buildSearchText(doc CommandDoc) string {
	text := doc.Command

	// Add aliases
	for _, alias := range doc.Aliases {
		text += " " + alias
	}

	// Add keywords
	for _, keyword := range doc.Keywords {
		text += " " + keyword
	}

	// Add example user requests (very important for matching)
	for _, example := range doc.Examples {
		text += " " + example.UserRequest
	}

	return text
}

// HybridMatcher combines keyword and semantic matching
type HybridMatcher struct {
	keywordMatcher  *Matcher
	semanticMatcher *SemanticMatcher
	strategy        string // "keyword", "semantic", "hybrid"
	threshold       int    // Score threshold for keyword matches
}

// NewHybridMatcher creates a new hybrid matcher
func NewHybridMatcher(embedder embeddings.Embedder, strategy string, threshold int) *HybridMatcher {
	return &HybridMatcher{
		keywordMatcher:  NewMatcher(),
		semanticMatcher: NewSemanticMatcher(embedder, nil), // Use MemoryStore by default
		strategy:        strategy,
		threshold:       threshold,
	}
}

// SetDocs sets the documents for both matchers
func (h *HybridMatcher) SetDocs(docs []CommandDoc) {
	h.keywordMatcher.SetDocs(docs)
}

// IndexSemantic indexes documents for semantic search
func (h *HybridMatcher) IndexSemantic(ctx context.Context, docs []CommandDoc) error {
	if h.semanticMatcher.embedder == nil {
		return nil // No embedder, skip semantic indexing
	}
	return h.semanticMatcher.Index(ctx, docs)
}

// FindRelevantDocs finds relevant docs using hybrid strategy
func (h *HybridMatcher) FindRelevantDocs(ctx context.Context, request string, maxDocs int) ([]CommandDoc, error) {
	switch h.strategy {
	case "keyword":
		// Keyword only
		return h.keywordMatcher.FindRelevantDocs(request, maxDocs), nil

	case "semantic":
		// Semantic only
		docs, _, err := h.semanticMatcher.Search(ctx, request, maxDocs)
		return docs, err

	case "hybrid":
		// Try keyword first
		keywordDocs := h.keywordMatcher.FindRelevantDocs(request, maxDocs)

		// If we got good keyword matches (score > threshold), use them
		if len(keywordDocs) > 0 {
			// For now, just use keyword matches
			// In a full implementation, we'd check the actual scores
			return keywordDocs, nil
		}

		// Otherwise fall back to semantic search
		if h.semanticMatcher.indexed {
			docs, _, err := h.semanticMatcher.Search(ctx, request, maxDocs)
			if err == nil && len(docs) > 0 {
				return docs, nil
			}
		}

		// Fall back to keyword matches even if scores are low
		return keywordDocs, nil

	default:
		return h.keywordMatcher.FindRelevantDocs(request, maxDocs), nil
	}
}

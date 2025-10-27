package customcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/iishyfishyy/please/internal/config"
	"github.com/iishyfishyy/please/internal/customcmd/embeddings"
	"github.com/iishyfishyy/please/internal/customcmd/vectorstore"
	"github.com/iishyfishyy/please/internal/ui"
)

// Manager coordinates loading, matching, and indexing of custom commands
type Manager struct {
	commandsDir      string
	docs             []CommandDoc
	matcher          *Matcher
	semanticMatcher  *SemanticMatcher
	indexed          bool
	indexTime        time.Time
	mu               sync.RWMutex
	// Embedding configuration (optional)
	embeddingEnabled bool
	provider         string
	model            string
	dims             int
	// Debug flag
	debug bool
}

// CommandDoc represents a custom command documentation file
type CommandDoc struct {
	Filename   string    // Original file name
	Command    string    // Primary command name
	Aliases    []string  // Alternative names
	Keywords   []string  // Keywords for matching
	Categories []string  // Categories (e.g., devops, database)
	Priority   string    // Priority (high, medium, low)
	Version    string    // Version of the tool
	Content    string    // Full markdown content
	Examples   []Example // Parsed examples
	UpdatedAt  time.Time // File modification time
}

// Example represents a user request → command example
type Example struct {
	UserRequest string
	Command     string
}

// NewManager creates a new custom command manager
func NewManager() (*Manager, error) {
	return NewManagerWithDebug(false)
}

// NewManagerWithDebug creates a new custom command manager with debug logging
func NewManagerWithDebug(debug bool) (*Manager, error) {
	commandsDir, err := GetCommandsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get commands directory: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: manager created (commands_dir=%s)\n", commandsDir)
	}

	m := &Manager{
		commandsDir: commandsDir,
		docs:        []CommandDoc{},
		matcher:     NewMatcherWithDebug(debug),
		debug:       debug,
	}

	return m, nil
}

// GetCommandsDir returns the path to the custom commands directory
func GetCommandsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".please", "commands"), nil
}

// GetEmbeddingsCachePath returns the path to the embeddings cache database
func GetEmbeddingsCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".please", "embeddings.db"), nil
}

// EnsureCommandsDir creates the commands directory if it doesn't exist
func EnsureCommandsDir() error {
	dir, err := GetCommandsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	return nil
}

// Load reads all command documentation files from the commands directory
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: loading commands from %s\n", m.commandsDir)
	}

	loader := NewLoaderWithDebug(m.debug)
	docs, err := loader.LoadAll(m.commandsDir)
	if err != nil {
		return fmt.Errorf("failed to load commands: %w", err)
	}

	if m.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: loaded %d command docs\n", len(docs))
		for _, doc := range docs {
			fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd:   - %s (%d examples, %d keywords)\n",
				doc.Command, len(doc.Examples), len(doc.Keywords))
		}
	}

	m.docs = docs
	m.matcher.SetDocs(docs)
	m.indexed = true
	m.indexTime = time.Now()

	return nil
}

// GetRelevantDocs finds the most relevant custom command docs for a request
func (m *Manager) GetRelevantDocs(request string, maxDocs int) []CommandDoc {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.indexed || len(m.docs) == 0 {
		if m.debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: no docs available (indexed=%v, count=%d)\n", m.indexed, len(m.docs))
		}
		return []CommandDoc{}
	}

	results := m.matcher.FindRelevantDocs(request, maxDocs)
	if m.debug && len(results) > 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: found %d relevant docs for %q\n", len(results), request)
		for i, doc := range results {
			fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd:   %d. %s\n", i+1, doc.Command)
		}
	}

	return results
}

// IsIndexed returns whether commands have been loaded
func (m *Manager) IsIndexed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.indexed
}

// GetIndexTime returns when commands were last indexed
func (m *Manager) GetIndexTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.indexTime
}

// NeedsReindex checks if any command files have been modified since last index
func (m *Manager) NeedsReindex() bool {
	m.mu.RLock()
	indexTime := m.indexTime
	m.mu.RUnlock()

	if !m.IsIndexed() {
		return true
	}

	// Check if any .md files are newer than index
	pattern := filepath.Join(m.commandsDir, "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if info.ModTime().After(indexTime) {
			return true
		}
	}

	return false
}

// Count returns the number of loaded command docs
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.docs)
}

// GetDocs returns all loaded command docs (for listing)
func (m *Manager) GetDocs() []CommandDoc {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	docs := make([]CommandDoc, len(m.docs))
	copy(docs, m.docs)
	return docs
}

// SetEmbeddingConfig configures the manager to use embeddings for semantic search
func (m *Manager) SetEmbeddingConfig(provider, model string, dims int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.embeddingEnabled = true
	m.provider = provider
	m.model = model
	m.dims = dims
}

// createEmbedder creates an embedder instance based on the configured provider
func (m *Manager) createEmbedder() (embeddings.Embedder, error) {
	switch m.provider {
	case "ollama":
		baseURL := "http://localhost:11434"
		return embeddings.NewOllamaEmbedder(baseURL, m.model)

	case "openai":
		// Get API key from env var first
		apiKey := os.Getenv("OPENAI_API_KEY")

		// If not in env, try to load from config
		if apiKey == "" {
			cfg, err := config.Load()
			if err == nil && cfg != nil && cfg.CustomCommands != nil {
				apiKey = cfg.CustomCommands.OpenAI.APIKey
			}
		}

		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key not found (set OPENAI_API_KEY or store in config)")
		}

		return embeddings.NewOpenAIEmbedder(apiKey, m.model)

	default:
		return nil, fmt.Errorf("unknown embedding provider: %s", m.provider)
	}
}

// Index explicitly loads/reloads all command documentation
// If force is true, bypasses cache and regenerates embeddings
func (m *Manager) Index(ctx context.Context, force bool) error {
	// 1. Load command docs
	if err := m.Load(); err != nil {
		return err
	}

	// 2. If embeddings not enabled, just return
	if !m.embeddingEnabled {
		return nil
	}

	// 3. Get cache path
	cachePath, err := GetEmbeddingsCachePath()
	if err != nil {
		// Non-fatal, continue with in-memory
		m.semanticMatcher = NewSemanticMatcher(nil, nil)
		return nil
	}

	// 4. Try to load existing cache
	if !force {
		sqlStore, err := vectorstore.OpenSQLiteStore(cachePath)
		if err == nil {
			// Convert docs to vectorstore.CommandDoc type
			vstoreDocs := make([]vectorstore.CommandDoc, len(m.docs))
			for i, doc := range m.docs {
				vstoreDocs[i] = vectorstore.CommandDoc{
					Filename:  doc.Filename,
					UpdatedAt: doc.UpdatedAt,
				}
			}

			// Cache exists - validate it
			valid, _ := sqlStore.IsValid(vstoreDocs, m.provider, m.model, m.dims)

			if valid {
				// Cache is valid - use it
				m.semanticMatcher = &SemanticMatcher{
					vectorStore: sqlStore,
					indexed:     true,
				}
				return nil
			}

			// Cache invalid - clear it
			sqlStore.Clear(ctx)
			sqlStore.Close()
		}
	}

	// 5. Cache doesn't exist, is invalid, or force flag set - regenerate embeddings
	ui.ShowInfo("Generating embeddings...")

	// Create embedder
	embedder, err := m.createEmbedder()
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	// Create new SQLite store
	sqlStore, err := vectorstore.NewSQLiteStore(cachePath, m.provider, m.model, m.dims)
	if err != nil {
		// Fallback to in-memory if SQLite fails
		ui.ShowWarning(fmt.Sprintf("Failed to create cache: %v", err))
		ui.ShowWarning("Using in-memory storage (embeddings won't be persisted)")

		m.semanticMatcher = NewSemanticMatcher(embedder, nil)
		if err := m.semanticMatcher.Index(ctx, m.docs); err != nil {
			return fmt.Errorf("failed to index: %w", err)
		}

		return nil
	}

	// Create semantic matcher with SQLite store
	m.semanticMatcher = NewSemanticMatcher(embedder, sqlStore)

	// Generate embeddings and store them
	ui.ShowInfo(fmt.Sprintf("Processing %d commands...", len(m.docs)))
	start := time.Now()

	if err := m.semanticMatcher.Index(ctx, m.docs); err != nil {
		sqlStore.Close()
		return fmt.Errorf("failed to index: %w", err)
	}

	duration := time.Since(start)
	ui.ShowSuccess(fmt.Sprintf("Generated and cached embeddings (%.1fs)", duration.Seconds()))

	return nil
}

// GetRelevantDocsForAgent returns docs in a format suitable for the agent
// This implements part of the CustomCommandManager interface
func (m *Manager) GetRelevantDocsForAgent(request string, maxDocs int) []AgentCommandDoc {
	docs := m.GetRelevantDocs(request, maxDocs)

	agentDocs := make([]AgentCommandDoc, len(docs))
	for i, doc := range docs {
		agentDocs[i] = AgentCommandDoc{
			Command: doc.Command,
			Content: doc.Content,
			Examples: make([]AgentExample, len(doc.Examples)),
		}
		for j, ex := range doc.Examples {
			agentDocs[i].Examples[j] = AgentExample{
				UserRequest: ex.UserRequest,
				Command:     ex.Command,
			}
		}
	}

	return agentDocs
}

// AgentCommandDoc is a simplified command doc for the agent
type AgentCommandDoc struct {
	Command  string
	Content  string
	Examples []AgentExample
}

// AgentExample represents a command example for the agent
type AgentExample struct {
	UserRequest string
	Command     string
}

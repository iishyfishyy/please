package vectorstore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore is a persistent vector store using SQLite
type SQLiteStore struct {
	db       *sql.DB
	dbPath   string
	provider string
	model    string
	dims     int
	mu       sync.RWMutex
}

// NewSQLiteStore creates a new SQLite vector store
func NewSQLiteStore(dbPath, provider, model string, dims int) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{
		db:       db,
		dbPath:   dbPath,
		provider: provider,
		model:    model,
		dims:     dims,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Store metadata
	if err := store.setMetadata("version", "1"); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.setMetadata("provider", provider); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.setMetadata("model", model); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.setMetadata("dimensions", strconv.Itoa(dims)); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.setMetadata("indexed_at", time.Now().Format(time.RFC3339)); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// OpenSQLiteStore opens an existing SQLite vector store
func OpenSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database does not exist: %s", dbPath)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{
		db:     db,
		dbPath: dbPath,
	}

	// Load metadata
	provider, err := store.getMetadata("provider")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to read provider metadata: %w", err)
	}

	model, err := store.getMetadata("model")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to read model metadata: %w", err)
	}

	dimsStr, err := store.getMetadata("dimensions")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to read dimensions metadata: %w", err)
	}

	dims, err := strconv.Atoi(dimsStr)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("invalid dimensions: %w", err)
	}

	store.provider = provider
	store.model = model
	store.dims = dims

	return store, nil
}

// initSchema creates the database schema
func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS embeddings (
		id TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		filename TEXT NOT NULL,
		file_mtime INTEGER NOT NULL,
		vector BLOB NOT NULL,
		metadata_json TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_command ON embeddings(command);
	CREATE INDEX IF NOT EXISTS idx_filename ON embeddings(filename);
	CREATE INDEX IF NOT EXISTS idx_mtime ON embeddings(file_mtime);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Add stores a vector with metadata
func (s *SQLiteStore) Add(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	if len(vector) == 0 {
		return fmt.Errorf("empty vector")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Encode vector to binary
	vectorBlob := encodeVector(vector)

	// Encode metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	// Extract required fields from metadata
	command, _ := metadata["command"].(string)
	filename, _ := metadata["filename"].(string)
	fileMtime, _ := metadata["file_mtime"].(int64)

	// Insert or replace
	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO embeddings (id, command, filename, file_mtime, vector, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, command, filename, fileMtime, vectorBlob, string(metadataJSON))

	return err
}

// Search finds the top K most similar vectors using cosine similarity
func (s *SQLiteStore) Search(ctx context.Context, query []float32, topK int) ([]SearchResult, error) {
	if len(query) == 0 {
		return nil, fmt.Errorf("empty query vector")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Load all vectors from database
	rows, err := s.db.QueryContext(ctx, `SELECT id, vector, metadata_json FROM embeddings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Calculate cosine similarity for each
	type scoredResult struct {
		id       string
		score    float32
		metadata map[string]interface{}
	}

	results := []scoredResult{}

	for rows.Next() {
		var id string
		var vectorBlob []byte
		var metadataJSON string

		if err := rows.Scan(&id, &vectorBlob, &metadataJSON); err != nil {
			continue
		}

		vector := decodeVector(vectorBlob)
		score := cosineSimilarity(query, vector)

		var metadata map[string]interface{}
		json.Unmarshal([]byte(metadataJSON), &metadata)

		results = append(results, scoredResult{id, score, metadata})
	}

	if len(results) == 0 {
		return []SearchResult{}, nil
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Return top K
	k := topK
	if k > len(results) {
		k = len(results)
	}

	searchResults := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		searchResults[i] = SearchResult{
			ID:       results[i].id,
			Score:    results[i].score,
			Metadata: results[i].metadata,
		}
	}

	return searchResults, nil
}

// Delete removes a vector by ID
func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM embeddings WHERE id = ?`, id)
	return err
}

// Clear removes all vectors
func (s *SQLiteStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM embeddings`)
	return err
}

// Count returns the number of stored vectors
func (s *SQLiteStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM embeddings`).Scan(&count)
	if err != nil {
		return 0
	}

	return count
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// IsValid checks if the cache is valid for the given documents and configuration
func (s *SQLiteStore) IsValid(docs []CommandDoc, provider, model string, dims int) (bool, string) {
	// 1. Check metadata matches
	storedProvider, err := s.getMetadata("provider")
	if err != nil || storedProvider != provider {
		return false, fmt.Sprintf("provider changed: %s → %s", storedProvider, provider)
	}

	storedModel, err := s.getMetadata("model")
	if err != nil || storedModel != model {
		return false, fmt.Sprintf("model changed: %s → %s", storedModel, model)
	}

	storedDims, err := s.getMetadata("dimensions")
	if err != nil || storedDims != strconv.Itoa(dims) {
		return false, fmt.Sprintf("dimensions changed: %s → %d", storedDims, dims)
	}

	// 2. Build map of expected filenames
	expectedFiles := make(map[string]time.Time)
	for _, doc := range docs {
		expectedFiles[doc.Filename] = doc.UpdatedAt
	}

	// 3. Check all embeddings have corresponding files with matching mtimes
	rows, err := s.db.Query(`SELECT filename, file_mtime FROM embeddings`)
	if err != nil {
		return false, "failed to query embeddings"
	}
	defer rows.Close()

	cachedFiles := make(map[string]bool)

	for rows.Next() {
		var filename string
		var fileMtime int64

		if err := rows.Scan(&filename, &fileMtime); err != nil {
			continue
		}

		cachedFiles[filename] = true

		// Check if file still exists in docs
		expectedMtime, exists := expectedFiles[filename]
		if !exists {
			return false, fmt.Sprintf("file deleted: %s", filename)
		}

		// Check if mtime matches
		if expectedMtime.Unix() != fileMtime {
			return false, fmt.Sprintf("file modified: %s", filename)
		}
	}

	// 4. Check for new files not in cache
	for filename := range expectedFiles {
		if !cachedFiles[filename] {
			return false, fmt.Sprintf("new file added: %s", filename)
		}
	}

	return true, ""
}

// getMetadata retrieves a metadata value
func (s *SQLiteStore) getMetadata(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("metadata key not found: %s", key)
	}
	return value, err
}

// setMetadata stores a metadata value
func (s *SQLiteStore) setMetadata(key, value string) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO metadata (key, value)
		VALUES (?, ?)
	`, key, value)
	return err
}

// UpdateIndexTime updates the indexed_at timestamp
func (s *SQLiteStore) UpdateIndexTime() error {
	return s.setMetadata("indexed_at", time.Now().Format(time.RFC3339))
}

// encodeVector encodes a float32 slice to binary
func encodeVector(v []float32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, v)
	return buf.Bytes()
}

// decodeVector decodes binary data to a float32 slice
func decodeVector(b []byte) []float32 {
	buf := bytes.NewReader(b)
	v := make([]float32, len(b)/4)
	binary.Read(buf, binary.LittleEndian, &v)
	return v
}

// CommandDoc represents a custom command documentation file
// Duplicated here to avoid circular dependency
type CommandDoc struct {
	Filename  string
	UpdatedAt time.Time
}

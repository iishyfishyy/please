package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	HistoryFileName = "history.json"
)

// Entry represents a single command history entry
type Entry struct {
	Timestamp       time.Time `json:"timestamp"`
	OriginalRequest string    `json:"original_request"`
	FinalCommand    string    `json:"final_command"`
	Executed        bool      `json:"executed"`
	Modifications   []string  `json:"modifications,omitempty"`
}

// History manages command history
type History struct {
	Entries []Entry `json:"entries"`
}

// GetHistoryPath returns the path to the history file
func GetHistoryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".please", HistoryFileName), nil
}

// Load reads the history from disk
func Load() (*History, error) {
	historyPath, err := GetHistoryPath()
	if err != nil {
		return nil, err
	}

	// If history doesn't exist, return empty history
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return &History{Entries: []Entry{}}, nil
	}

	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var hist History
	if err := json.Unmarshal(data, &hist); err != nil {
		return nil, fmt.Errorf("failed to parse history file: %w", err)
	}

	return &hist, nil
}

// Save writes the history to disk
func (h *History) Save() error {
	historyPath, err := GetHistoryPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// AddEntry adds a new entry to the history
func (h *History) AddEntry(entry Entry) {
	h.Entries = append(h.Entries, entry)
}

// NewEntry creates a new history entry
func NewEntry(originalRequest, finalCommand string, executed bool, modifications []string) Entry {
	return Entry{
		Timestamp:       time.Now(),
		OriginalRequest: originalRequest,
		FinalCommand:    finalCommand,
		Executed:        executed,
		Modifications:   modifications,
	}
}

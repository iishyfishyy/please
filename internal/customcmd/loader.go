package customcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Loader handles loading command documentation files
type Loader struct {
	parser *Parser
	debug  bool
}

// NewLoader creates a new loader
func NewLoader() *Loader {
	return NewLoaderWithDebug(false)
}

// NewLoaderWithDebug creates a new loader with debug logging
func NewLoaderWithDebug(debug bool) *Loader {
	return &Loader{
		parser: NewParser(),
		debug:  debug,
	}
}

// LoadAll loads all .md files from the specified directory
func (l *Loader) LoadAll(dir string) ([]CommandDoc, error) {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []CommandDoc{}, nil // Not an error, just no commands yet
	}

	// Find all .md files
	pattern := filepath.Join(dir, "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob files: %w", err)
	}

	if l.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Loader: found %d .md files in %s\n", len(files), dir)
	}

	if len(files) == 0 {
		return []CommandDoc{}, nil
	}

	// Filter out README.md and other meta files
	var commandFiles []string
	for _, file := range files {
		basename := filepath.Base(file)
		// Skip README and files starting with _ (convention for meta files)
		if strings.EqualFold(basename, "README.md") || strings.HasPrefix(basename, "_") {
			if l.debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Loader: skipping meta file: %s\n", basename)
			}
			continue
		}
		commandFiles = append(commandFiles, file)
		if l.debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Loader: loading command file: %s\n", basename)
		}
	}

	// Parse all command files
	docs, err := l.parser.ParseAll(commandFiles)
	if err != nil {
		// Don't fail completely if some files have errors
		// The parser already collected docs that did parse
		if l.debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Loader: parse error (partial success): %v\n", err)
		}
		return docs, nil
	}

	if l.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Loader: successfully parsed %d command docs\n", len(docs))
	}

	return docs, nil
}

// LoadSingle loads a single command documentation file
func (l *Loader) LoadSingle(filepath string) (*CommandDoc, error) {
	return l.parser.Parse(filepath)
}

// HasCommands checks if the commands directory exists and has any .md files
func HasCommands() (bool, error) {
	dir, err := GetCommandsDir()
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false, nil
	}

	pattern := filepath.Join(dir, "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return false, err
	}

	// Filter out README.md
	for _, file := range files {
		basename := filepath.Base(file)
		if !strings.EqualFold(basename, "README.md") && !strings.HasPrefix(basename, "_") {
			return true, nil
		}
	}

	return false, nil
}

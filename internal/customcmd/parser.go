package customcmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser handles parsing markdown files with YAML frontmatter
type Parser struct{}

// NewParser creates a new parser
func NewParser() *Parser {
	return &Parser{}
}

// Frontmatter represents the YAML frontmatter in a command doc
type Frontmatter struct {
	Command    string   `yaml:"command"`
	Aliases    []string `yaml:"aliases"`
	Keywords   []string `yaml:"keywords"`
	Categories []string `yaml:"categories"`
	Priority   string   `yaml:"priority"`
	Version    string   `yaml:"version"`
}

// Parse parses a markdown file with frontmatter
func (p *Parser) Parse(filepath string) (*CommandDoc, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for modification time
	info, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Read file content
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse frontmatter and content
	frontmatter, content, err := p.parseFrontmatter(lines)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Parse examples from content
	examples := p.parseExamples(content)

	doc := &CommandDoc{
		Filename:   filepath,
		Command:    frontmatter.Command,
		Aliases:    frontmatter.Aliases,
		Keywords:   frontmatter.Keywords,
		Categories: frontmatter.Categories,
		Priority:   frontmatter.Priority,
		Version:    frontmatter.Version,
		Content:    content,
		Examples:   examples,
		UpdatedAt:  info.ModTime(),
	}

	return doc, nil
}

// parseFrontmatter extracts YAML frontmatter and returns it with the remaining content
func (p *Parser) parseFrontmatter(lines []string) (*Frontmatter, string, error) {
	if len(lines) == 0 {
		return nil, "", fmt.Errorf("empty file")
	}

	// Check for frontmatter delimiter
	if strings.TrimSpace(lines[0]) != "---" {
		return nil, strings.Join(lines, "\n"), nil // No frontmatter, return all as content
	}

	// Find end of frontmatter
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return nil, "", fmt.Errorf("unclosed frontmatter")
	}

	// Parse YAML frontmatter
	frontmatterLines := lines[1:endIdx]
	frontmatterYAML := strings.Join(frontmatterLines, "\n")

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &fm); err != nil {
		return nil, "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Content is everything after frontmatter
	contentLines := lines[endIdx+1:]
	content := strings.Join(contentLines, "\n")

	return &fm, content, nil
}

// parseExamples extracts examples from the markdown content
// Looks for patterns like:
//   User: "show me all pods"
//   Command: kubectl get pods -A
func (p *Parser) parseExamples(content string) []Example {
	var examples []Example

	// Pattern to match example blocks
	// Looking for lines like:
	// User: "request text"
	// Command: actual command
	userPattern := regexp.MustCompile(`(?i)(?:User|Request):\s*["'](.+?)["']`)
	commandPattern := regexp.MustCompile(`(?i)Command:\s*(.+)`)

	lines := strings.Split(content, "\n")
	var currentUserRequest string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for user request
		if matches := userPattern.FindStringSubmatch(line); len(matches) > 1 {
			currentUserRequest = matches[1]
			continue
		}

		// Check for command (must follow a user request)
		if currentUserRequest != "" {
			if matches := commandPattern.FindStringSubmatch(line); len(matches) > 1 {
				command := strings.TrimSpace(matches[1])
				// Remove backticks if present
				command = strings.Trim(command, "`")

				examples = append(examples, Example{
					UserRequest: currentUserRequest,
					Command:     command,
				})

				currentUserRequest = "" // Reset
			}
		}
	}

	return examples
}

// ParseAll parses multiple files
func (p *Parser) ParseAll(filepaths []string) ([]CommandDoc, error) {
	var docs []CommandDoc
	var errors []string

	for _, filepath := range filepaths {
		doc, err := p.Parse(filepath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filepath, err))
			continue
		}
		docs = append(docs, *doc)
	}

	if len(errors) > 0 {
		return docs, fmt.Errorf("failed to parse some files:\n%s", strings.Join(errors, "\n"))
	}

	return docs, nil
}

// ExtractCommonPatterns extracts common command patterns from content
// Used for adding to prompts when token budget allows
func ExtractCommonPatterns(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	var patterns []string
	inCodeBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Toggle code block state
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		// Skip empty lines and headers
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Capture lines in code blocks (likely command examples)
		if inCodeBlock && line != "" {
			patterns = append(patterns, line)
			if len(patterns) >= maxLines {
				break
			}
		}
	}

	if len(patterns) == 0 {
		return ""
	}

	return strings.Join(patterns, "\n")
}

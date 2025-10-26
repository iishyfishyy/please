package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ClaudeAgent implements the Agent interface using Claude CLI
type ClaudeAgent struct{}

// NewClaudeAgent creates a new Claude agent
func NewClaudeAgent() *ClaudeAgent {
	return &ClaudeAgent{}
}

// IsClaudeCLIInstalled checks if the claude CLI is available
func IsClaudeCLIInstalled() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// TranslateToCommand translates natural language to a shell command
func (c *ClaudeAgent) TranslateToCommand(ctx context.Context, request string) (string, error) {
	prompt := fmt.Sprintf(`%s

Convert this request into a shell command: "%s"

IMPORTANT: Respond with ONLY the command itself, nothing else. No explanations, no markdown, no code blocks. Just the raw command.`,
		c.buildSystemPrompt(), request)

	return c.callClaude(ctx, prompt)
}

// RefineCommand refines an existing command based on modification request
func (c *ClaudeAgent) RefineCommand(ctx context.Context, originalCommand, modificationRequest string) (string, error) {
	prompt := fmt.Sprintf(`%s

Original command: %s

Modification request: %s

IMPORTANT: Respond with ONLY the modified command itself, nothing else. No explanations, no markdown, no code blocks. Just the raw command.`,
		c.buildSystemPrompt(), originalCommand, modificationRequest)

	return c.callClaude(ctx, prompt)
}

// ExplainCommand provides a human-readable explanation of a shell command
func (c *ClaudeAgent) ExplainCommand(ctx context.Context, command string) (string, error) {
	osInfo := runtime.GOOS
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	prompt := fmt.Sprintf(`You are a helpful assistant that explains shell commands in simple, clear terms.

Environment:
- Operating System: %s
- Shell: %s

Command to explain: %s

Provide a concise explanation that covers:
1. What the command does overall
2. What each part/flag does
3. Any important warnings or notes

Keep it brief but informative. Use plain language that non-experts can understand.`,
		osInfo, shell, command)

	return c.callClaude(ctx, prompt)
}

// gatherContext collects environment context for better command generation
func (c *ClaudeAgent) gatherContext() string {
	var context strings.Builder

	// Current working directory
	if cwd, err := os.Getwd(); err == nil {
		context.WriteString(fmt.Sprintf("- Current directory: %s\n", cwd))
	}

	// Directory contents summary (token-efficient)
	if summary := c.summarizeDirectory(); summary != "" {
		context.WriteString(fmt.Sprintf("- Files present: %s\n", summary))
	}

	return context.String()
}

// summarizeDirectory creates a compact summary of current directory contents
func (c *ClaudeAgent) summarizeDirectory() string {
	entries, err := os.ReadDir(".")
	if err != nil || len(entries) == 0 {
		return ""
	}

	// Count by extension and directories
	counts := make(map[string]int)
	dirCount := 0

	for _, entry := range entries {
		// Skip hidden files to save tokens
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if entry.IsDir() {
			dirCount++
		} else {
			ext := filepath.Ext(entry.Name())
			if ext != "" {
				counts[ext]++
			} else {
				counts["[no ext]"]++
			}
		}
	}

	// Build compact summary (limit to top 5 file types)
	var parts []string
	if dirCount > 0 {
		parts = append(parts, fmt.Sprintf("%d directories", dirCount))
	}

	// Sort extensions by count (descending)
	type extCount struct {
		ext   string
		count int
	}
	var sorted []extCount
	for ext, count := range counts {
		sorted = append(sorted, extCount{ext, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	// Add top 5 file types
	for i, ec := range sorted {
		if i >= 5 {
			break
		}
		parts = append(parts, fmt.Sprintf("%d %s files", ec.count, ec.ext))
	}

	return strings.Join(parts, ", ")
}

// buildSystemPrompt creates the system prompt for Claude with context
func (c *ClaudeAgent) buildSystemPrompt() string {
	osInfo := runtime.GOOS
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	// Gather runtime context
	contextInfo := c.gatherContext()

	var contextSection string
	if contextInfo != "" {
		contextSection = "\nContext:\n" + contextInfo
	}

	return fmt.Sprintf(`You are a command-line expert that translates natural language into shell commands.

Environment:
- Operating System: %s
- Shell: %s%s

CRITICAL RULES:
1. Output ONLY the raw command - no explanations, no markdown, no code blocks
2. Generate safe, correct shell commands for the current environment
3. Prefer portable commands when possible (use standard Unix/Linux utilities)
4. Make reasonable assumptions for ambiguous requests
5. Consider the current directory context when generating commands

SAFETY GUIDELINES:
- Avoid destructive commands without clear intent (rm -rf, dd, mkfs, etc.)
- Use common utilities that are widely available
- Respect the user's shell and OS capabilities
- For dangerous operations, ensure the request explicitly indicates intent

EXAMPLES:
Request: "list all files"
Command: ls -la

Request: "find javascript files modified today"
Command: find . -name "*.js" -mtime -1

Request: "count lines in go files"
Command: find . -name "*.go" -exec wc -l {} + | tail -1

Request: "show git log"
Command: git log -10 --oneline

Remember: Respond with ONLY the command itself, nothing else.`, osInfo, shell, contextSection)
}

// callClaude calls the Claude CLI with the given prompt
func (c *ClaudeAgent) callClaude(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, "claude", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to call claude CLI: %w\nStderr: %s", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("claude CLI returned empty response")
	}

	return output, nil
}

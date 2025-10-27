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
type ClaudeAgent struct {
	customCmdGetter CustomDocGetter
	debug           bool
}

// CustomDocGetter is a function type for getting custom command docs
// This avoids circular dependencies
type CustomDocGetter func(request string, maxDocs int) []CustomCommandDoc

// CustomCommandDoc represents a custom command document (simplified for agent)
type CustomCommandDoc struct {
	Command  string
	Content  string
	Examples []CommandExample
}

// CommandExample represents a command example
type CommandExample struct {
	UserRequest string
	Command     string
}

// NewClaudeAgent creates a new Claude agent
func NewClaudeAgent() *ClaudeAgent {
	return &ClaudeAgent{}
}

// SetCustomDocGetter sets the custom command doc getter function
func (c *ClaudeAgent) SetCustomDocGetter(getter CustomDocGetter) {
	c.customCmdGetter = getter
}

// SetDebug enables or disables debug logging
func (c *ClaudeAgent) SetDebug(debug bool) {
	c.debug = debug
}

// IsClaudeCLIInstalled checks if the claude CLI is available
func IsClaudeCLIInstalled() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// TranslateToCommand translates natural language to a shell command
func (c *ClaudeAgent) TranslateToCommand(ctx context.Context, request string) (string, error) {
	// Get relevant custom commands if available
	customDocs := c.getRelevantCustomDocs(request)
	if c.debug && len(customDocs) > 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG] Agent: retrieved %d custom command docs\n", len(customDocs))
	}
	customContext := c.buildCustomCommandContext(customDocs)
	if c.debug && customContext != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Agent: custom command context length: %d chars\n", len(customContext))
	}

	systemPrompt := c.buildSystemPrompt()
	prompt := fmt.Sprintf(`%s
%s
Convert this request into a shell command: "%s"

IMPORTANT: Respond with ONLY the command itself, nothing else. No explanations, no markdown, no code blocks. Just the raw command.`,
		systemPrompt, customContext, request)

	if c.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Agent: built prompt (%d chars)\n", len(prompt))
		if len(prompt) < 500 {
			fmt.Fprintf(os.Stderr, "[DEBUG] Agent: prompt preview: %q\n", prompt)
		}
	}

	return c.callClaude(ctx, prompt)
}

// RefineCommand refines an existing command based on modification request
func (c *ClaudeAgent) RefineCommand(ctx context.Context, originalCommand, modificationRequest string) (string, error) {
	if c.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Agent: refining command %q with modification %q\n", originalCommand, modificationRequest)
	}

	prompt := fmt.Sprintf(`%s

Original command: %s

Modification request: %s

DO NOT EXPLAIN. DO NOT USE MARKDOWN. DO NOT ADD COMMENTARY.

WRONG OUTPUT (DO NOT DO THIS):
The modified command to list Go files would be:
find . -name "*.go"

WRONG OUTPUT (DO NOT DO THIS):
` + "```bash" + `
find . -name "*.go"
` + "```" + `

CORRECT OUTPUT (DO THIS):
find . -name "*.go"

YOUR TASK: Output ONLY the modified command. Nothing else. No text before it. No text after it. No markdown. No explanation. Just the raw shell command on a single line.

Modified command:`,
		c.buildSystemPrompt(), originalCommand, modificationRequest)

	return c.callClaude(ctx, prompt)
}

// ExplainCommand provides a human-readable explanation of a shell command
// request is the original user request (used to match custom commands)
func (c *ClaudeAgent) ExplainCommand(ctx context.Context, command string, request string) (string, error) {
	osInfo := runtime.GOOS
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	// Get relevant custom commands if available (for proprietary tools)
	customDocs := c.getRelevantCustomDocs(request)
	if c.debug && len(customDocs) > 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG] Agent: found %d custom command docs for explanation\n", len(customDocs))
	}
	customContext := c.buildCustomCommandContext(customDocs)

	prompt := fmt.Sprintf(`You are a helpful assistant that explains shell commands in simple, clear terms.

Environment:
- Operating System: %s
- Shell: %s
%s
Command to explain: %s

Provide a concise explanation that covers:
1. What the command does overall
2. What each part/flag does
3. Any important warnings or notes

Keep it brief but informative. Use plain language that non-experts can understand.`,
		osInfo, shell, customContext, command)

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

// getRelevantCustomDocs retrieves relevant custom command docs
func (c *ClaudeAgent) getRelevantCustomDocs(request string) []CustomCommandDoc {
	if c.customCmdGetter == nil {
		return nil
	}

	// Get up to 3 most relevant docs
	return c.customCmdGetter(request, 3)
}

// buildCustomCommandContext builds the custom commands section of the prompt
func (c *ClaudeAgent) buildCustomCommandContext(docs []CustomCommandDoc) string {
	if len(docs) == 0 {
		return ""
	}

	var context strings.Builder
	context.WriteString("\n\nCUSTOM COMMANDS AVAILABLE:\n")
	context.WriteString("The following custom/internal tools are available:\n\n")

	for _, doc := range docs {
		context.WriteString(fmt.Sprintf("## %s\n", doc.Command))

		// Include examples (most useful for matching)
		if len(doc.Examples) > 0 {
			context.WriteString("Examples:\n")
			for i, ex := range doc.Examples {
				if i >= 5 { // Limit to 5 examples per command to save tokens
					break
				}
				context.WriteString(fmt.Sprintf("  User: \"%s\"\n", ex.UserRequest))
				context.WriteString(fmt.Sprintf("  Command: %s\n", ex.Command))
			}
		}

		// Include common patterns (extract from content, limited)
		patterns := extractCommonPatterns(doc.Content, 10)
		if patterns != "" {
			context.WriteString("\nCommon patterns:\n")
			for _, line := range strings.Split(patterns, "\n") {
				if strings.TrimSpace(line) != "" {
					context.WriteString("  " + line + "\n")
				}
			}
		}

		context.WriteString("\n")
	}

	return context.String()
}

// extractCommonPatterns extracts command patterns from markdown content
func extractCommonPatterns(content string, maxLines int) string {
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

// callClaude calls the Claude CLI with the given prompt
func (c *ClaudeAgent) callClaude(ctx context.Context, prompt string) (string, error) {
	if c.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Agent: calling Claude CLI with prompt (%d chars)\n", len(prompt))
		// Log full prompt for transparency
		if len(prompt) <= 3000 {
			fmt.Fprintf(os.Stderr, "[DEBUG] Agent: full prompt:\n---\n%s\n---\n", prompt)
		} else {
			// For very long prompts, show first 2000 and last 500 chars
			fmt.Fprintf(os.Stderr, "[DEBUG] Agent: full prompt (truncated):\n---\n%s\n\n... [%d chars omitted] ...\n\n%s\n---\n",
				prompt[:2000], len(prompt)-2500, prompt[len(prompt)-500:])
		}
	}

	cmd := exec.CommandContext(ctx, "claude", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if c.debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Agent: Claude CLI failed: %v\n", err)
			if stderr.String() != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Agent: stderr: %s\n", stderr.String())
			}
		}
		return "", fmt.Errorf("failed to call claude CLI: %w\nStderr: %s", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if c.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Agent: received response (%d chars): %q\n", len(output), output)
	}

	if output == "" {
		return "", fmt.Errorf("claude CLI returned empty response")
	}

	return output, nil
}

package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

// buildSystemPrompt creates the system prompt for Claude
func (c *ClaudeAgent) buildSystemPrompt() string {
	osInfo := runtime.GOOS
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	return fmt.Sprintf(`You are a helpful assistant that translates natural language requests into shell commands.

Environment:
- Operating System: %s
- Shell: %s

Guidelines:
1. Generate safe, correct shell commands
2. Use common Unix/Linux utilities when possible
3. Prefer portable commands over OS-specific ones when applicable
4. Do not generate destructive commands without clear intent
5. Return ONLY the command - no explanations, no markdown formatting, no code blocks
6. If the request is ambiguous, make reasonable assumptions for the most common use case

Examples:
Request: "list all files"
Response: ls -la

Request: "find all javascript files"
Response: find . -name "*.js"

Request: "show disk usage"
Response: df -h`, osInfo, shell)
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

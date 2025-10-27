package agent

import "context"

// Agent represents an LLM agent that can translate natural language to shell commands
type Agent interface {
	// TranslateToCommand takes a natural language request and returns a shell command
	TranslateToCommand(ctx context.Context, request string) (string, error)

	// RefineCommand takes a command and modification request and returns a refined command
	RefineCommand(ctx context.Context, originalCommand, modificationRequest string) (string, error)

	// ExplainCommand takes a command and original request, returns a human-readable explanation
	// request is used to match custom command documentation for context
	ExplainCommand(ctx context.Context, command string, request string) (string, error)
}

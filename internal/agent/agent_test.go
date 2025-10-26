package agent

import (
	"context"
	"testing"
)

// TestAgentInterface ensures implementations satisfy the Agent interface
func TestAgentInterface(t *testing.T) {
	var _ Agent = (*ClaudeAgent)(nil)
	var _ Agent = (*MockAgent)(nil)
}

// MockAgent for testing code that depends on Agent interface
type MockAgent struct {
	TranslateFn func(context.Context, string) (string, error)
	RefineFn    func(context.Context, string, string) (string, error)
}

func (m *MockAgent) TranslateToCommand(ctx context.Context, request string) (string, error) {
	if m.TranslateFn != nil {
		return m.TranslateFn(ctx, request)
	}
	return "echo mock", nil
}

func (m *MockAgent) RefineCommand(ctx context.Context, originalCommand, modificationRequest string) (string, error) {
	if m.RefineFn != nil {
		return m.RefineFn(ctx, originalCommand, modificationRequest)
	}
	return "echo refined", nil
}

// Example of how to use MockAgent in tests
func ExampleMockAgent() {
	// Create a mock agent with custom behavior
	mock := &MockAgent{
		TranslateFn: func(ctx context.Context, req string) (string, error) {
			if req == "list files" {
				return "ls -la", nil
			}
			return "echo unknown", nil
		},
	}

	// Use the mock in your test
	cmd, _ := mock.TranslateToCommand(context.Background(), "list files")
	println(cmd) // Output: ls -la
}

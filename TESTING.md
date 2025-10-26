# Testing Guide for `please`

## Overview

This document provides comprehensive guidance for testing the `please` command-line tool. While the project currently has minimal test coverage, this guide establishes patterns and best practices for implementing a robust test suite.

## Testing Philosophy

### Core Principles

1. **Safety First**: Never execute dangerous commands in tests
2. **Determinism**: Tests should be reliable and reproducible
3. **Isolation**: Tests should not depend on external services (except where necessary)
4. **Speed**: Fast tests enable rapid iteration
5. **Readability**: Tests serve as documentation

### Test Pyramid

```
           ┌─────────────┐
           │   E2E Tests │  ← Few, slow, high confidence
           ├─────────────┤
           │Integration  │  ← Some, moderate speed
           │   Tests     │
           ├─────────────┤
           │    Unit     │  ← Many, fast, focused
           │   Tests     │
           └─────────────┘
```

## Project Structure

### Proposed Test Organization

```
please/
├── cmd/please/
│   └── main_test.go              # CLI integration tests
├── internal/
│   ├── agent/
│   │   ├── agent_test.go         # Interface contract tests
│   │   ├── claude_test.go        # Claude agent tests
│   │   └── testdata/
│   │       ├── prompts/          # Sample prompts
│   │       └── responses/        # Expected responses
│   ├── config/
│   │   ├── config_test.go        # Config load/save/validation
│   │   └── testdata/
│   │       ├── valid_config.json
│   │       └── invalid_config.json
│   ├── executor/
│   │   └── executor_test.go      # Safe command execution tests
│   ├── history/
│   │   ├── history_test.go       # History persistence tests
│   │   └── testdata/
│   │       └── sample_history.json
│   └── ui/
│       └── prompt_test.go        # UI logic tests (mocked)
└── TESTING.md                    # This file
```

## Unit Testing

### Package: internal/agent

#### Testing the Agent Interface

**File**: `internal/agent/agent_test.go`

```go
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
```

#### Testing Claude Agent (Mocked)

**File**: `internal/agent/claude_test.go`

```go
package agent

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestClaudeAgent_buildSystemPrompt(t *testing.T) {
	agent := NewClaudeAgent()
	prompt := agent.buildSystemPrompt()

	// Verify essential components
	if !strings.Contains(prompt, runtime.GOOS) {
		t.Error("System prompt should include OS information")
	}

	if !strings.Contains(prompt, "ONLY the raw command") {
		t.Error("System prompt should enforce format requirements")
	}

	if !strings.Contains(prompt, "SAFETY GUIDELINES") {
		t.Error("System prompt should include safety guidelines")
	}
}

func TestClaudeAgent_gatherContext(t *testing.T) {
	agent := NewClaudeAgent()
	context := agent.gatherContext()

	// Should include current directory
	if !strings.Contains(context, "Current directory:") {
		t.Error("Context should include current directory")
	}
}

func TestClaudeAgent_summarizeDirectory(t *testing.T) {
	// Create temp directory with known files
	tmpDir, err := os.MkdirTemp("", "please-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(tmpDir+"/test1.go", []byte("package main"), 0644)
	os.WriteFile(tmpDir+"/test2.go", []byte("package main"), 0644)
	os.WriteFile(tmpDir+"/test.md", []byte("# Test"), 0644)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	agent := NewClaudeAgent()
	summary := agent.summarizeDirectory()

	// Should mention .go files
	if !strings.Contains(summary, ".go") {
		t.Errorf("Summary should mention .go files, got: %s", summary)
	}
}

func TestIsClaudeCLIInstalled(t *testing.T) {
	// This test checks if claude CLI is in PATH
	// Result depends on environment
	installed := IsClaudeCLIInstalled()
	t.Logf("Claude CLI installed: %v", installed)
}

// Table-driven test for context gathering edge cases
func TestClaudeAgent_gatherContext_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (cleanup func())
		expectContext bool
	}{
		{
			name: "Empty directory",
			setupFunc: func() func() {
				tmpDir, _ := os.MkdirTemp("", "please-test")
				oldCwd, _ := os.Getwd()
				os.Chdir(tmpDir)
				return func() {
					os.Chdir(oldCwd)
					os.RemoveAll(tmpDir)
				}
			},
			expectContext: true, // Should still have current directory
		},
		{
			name: "Directory with only hidden files",
			setupFunc: func() func() {
				tmpDir, _ := os.MkdirTemp("", "please-test")
				os.WriteFile(tmpDir+"/.hidden", []byte("test"), 0644)
				oldCwd, _ := os.Getwd()
				os.Chdir(tmpDir)
				return func() {
					os.Chdir(oldCwd)
					os.RemoveAll(tmpDir)
				}
			},
			expectContext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupFunc()
			defer cleanup()

			agent := NewClaudeAgent()
			context := agent.gatherContext()

			if tt.expectContext && context == "" {
				t.Error("Expected context, got empty string")
			}

			if !tt.expectContext && context != "" {
				t.Errorf("Expected no context, got: %s", context)
			}
		})
	}
}
```

### Package: internal/config

**File**: `internal/config/config_test.go`

```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_LoadNotExists(t *testing.T) {
	// Setup: Use temporary home directory
	tmpDir, err := os.MkdirTemp("", "please-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Test: Load when config doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Errorf("Load() should not error when config missing, got: %v", err)
	}
	if cfg != nil {
		t.Error("Load() should return nil when config doesn't exist")
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "please-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Save config
	original := &Config{Agent: AgentClaude}
	if err := Save(original); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load config
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.Agent != original.Agent {
		t.Errorf("Expected agent %s, got %s", original.Agent, loaded.Agent)
	}
}

func TestConfig_InvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "please-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create invalid config file
	configDir := filepath.Join(tmpDir, ConfigDirName)
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, ConfigFileName)
	os.WriteFile(configPath, []byte("invalid json{"), 0644)

	// Attempt to load
	_, err = Load()
	if err == nil {
		t.Error("Load() should fail with invalid JSON")
	}
}

func TestConfig_Exists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "please-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Should not exist initially
	exists, err := Exists()
	if err != nil {
		t.Fatalf("Exists() failed: %v", err)
	}
	if exists {
		t.Error("Config should not exist initially")
	}

	// Save config
	cfg := &Config{Agent: AgentClaude}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Should exist now
	exists, err = Exists()
	if err != nil {
		t.Fatalf("Exists() failed: %v", err)
	}
	if !exists {
		t.Error("Config should exist after saving")
	}
}
```

### Package: internal/executor

**File**: `internal/executor/executor_test.go`

```go
package executor

import (
	"runtime"
	"strings"
	"testing"
)

func TestExecute_Success(t *testing.T) {
	// Use safe, simple commands
	err := Execute("echo test")
	if err != nil {
		t.Errorf("Execute() failed: %v", err)
	}
}

func TestExecute_Failure(t *testing.T) {
	// Command that should fail
	err := Execute("false")
	if err == nil {
		t.Error("Execute() should fail for 'false' command")
	}
}

func TestExecute_CommandNotFound(t *testing.T) {
	// Non-existent command
	err := Execute("nonexistent-command-xyz-12345")
	if err == nil {
		t.Error("Execute() should fail for non-existent command")
	}
}

func TestExecute_EmptyCommand(t *testing.T) {
	// Empty command
	err := Execute("")
	// Behavior may vary by shell, but shouldn't crash
	t.Logf("Empty command result: %v", err)
}

// Platform-specific tests
func TestExecute_Platform(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on Windows")
	}

	// Test shell detection
	err := Execute("echo $SHELL")
	if err != nil {
		t.Errorf("Execute() failed: %v", err)
	}
}
```

### Package: internal/history

**File**: `internal/history/history_test.go`

```go
package history

import (
	"os"
	"testing"
	"time"
)

func TestHistory_AddEntry(t *testing.T) {
	hist := &History{Entries: []Entry{}}

	entry := NewEntry("list files", "ls -la", true, nil)
	hist.AddEntry(entry)

	if len(hist.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(hist.Entries))
	}

	if hist.Entries[0].OriginalRequest != "list files" {
		t.Errorf("Expected 'list files', got '%s'", hist.Entries[0].OriginalRequest)
	}
}

func TestHistory_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "please-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create and save history
	hist := &History{Entries: []Entry{
		NewEntry("test command", "echo test", true, nil),
	}}

	if err := hist.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load history
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(loaded.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(loaded.Entries))
	}
}

func TestNewEntry(t *testing.T) {
	entry := NewEntry("test", "echo test", true, []string{"modification"})

	if entry.OriginalRequest != "test" {
		t.Error("Original request not set correctly")
	}

	if entry.FinalCommand != "echo test" {
		t.Error("Final command not set correctly")
	}

	if !entry.Executed {
		t.Error("Executed flag not set correctly")
	}

	if len(entry.Modifications) != 1 {
		t.Error("Modifications not set correctly")
	}

	// Timestamp should be recent
	if time.Since(entry.Timestamp) > time.Minute {
		t.Error("Timestamp seems incorrect")
	}
}
```

## Integration Testing

### CLI Integration Tests

**File**: `cmd/please/main_test.go`

```go
package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCLI_Version(t *testing.T) {
	// Build binary first (or use installed version)
	cmd := exec.Command("please", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("Skipping: please binary not available: %v", err)
	}

	if !strings.Contains(string(output), "please") {
		t.Errorf("Version output unexpected: %s", output)
	}
}

func TestCLI_Help(t *testing.T) {
	cmd := exec.Command("please", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("Skipping: please binary not available: %v", err)
	}

	expectedTerms := []string{"please", "command", "configure"}
	for _, term := range expectedTerms {
		if !strings.Contains(string(output), term) {
			t.Errorf("Help output should contain '%s'", term)
		}
	}
}
```

## Testing Best Practices

### 1. Use Table-Driven Tests

```go
func TestSomething(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"valid input", "test", "result", false},
		{"invalid input", "", "", true},
		{"edge case", "edge", "edge-result", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FunctionUnderTest(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}

			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
```

### 2. Test Helpers for Common Setup

```go
// testhelpers/config.go
package testhelpers

import (
	"os"
	"testing"
)

func SetupTestConfig(t *testing.T) (cleanup func()) {
	tmpDir, err := os.MkdirTemp("", "please-test")
	if err != nil {
		t.Fatal(err)
	}

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	return func() {
		os.Setenv("HOME", oldHome)
		os.RemoveAll(tmpDir)
	}
}

// Usage:
func TestSomething(t *testing.T) {
	cleanup := testhelpers.SetupTestConfig(t)
	defer cleanup()

	// Test code here
}
```

### 3. Mock External Dependencies

```go
// Don't call actual Claude CLI in tests
type mockClaudeCall func(ctx context.Context, prompt string) (string, error)

var claudeCall mockClaudeCall = actualClaudeCall

func (c *ClaudeAgent) callClaude(ctx context.Context, prompt string) (string, error) {
	return claudeCall(ctx, prompt)
}

// In tests:
func TestTranslate(t *testing.T) {
	oldCall := claudeCall
	defer func() { claudeCall = oldCall }()

	claudeCall = func(ctx context.Context, prompt string) (string, error) {
		return "ls -la", nil
	}

	// Test code
}
```

### 4. Use testdata/ for Fixtures

```
internal/agent/testdata/
├── prompts/
│   ├── simple_request.txt
│   └── complex_request.txt
└── responses/
    ├── simple_response.txt
    └── complex_response.txt
```

```go
func loadTestData(t *testing.T, filename string) string {
	data, err := os.ReadFile("testdata/" + filename)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
```

## Running Tests

### Basic Commands

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./internal/config

# Run specific test
go test -run TestConfig_SaveAndLoad ./internal/config

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### CI/CD Integration

**File**: `.github/workflows/test.yml`

```yaml
name: Test

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.21', '1.22']

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

## Coverage Goals

### Target Coverage

- **Overall**: 70%+
- **Critical paths**: 90%+ (agent, executor)
- **Configuration**: 80%+
- **UI**: 60%+ (harder to test)

### Check Coverage

```bash
# View coverage by package
go test -cover ./...

# Detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# HTML coverage visualization
go tool cover -html=coverage.out
```

## Testing Checklist

Before submitting code:

- [ ] All existing tests pass
- [ ] New features have tests
- [ ] Edge cases are covered
- [ ] Error paths are tested
- [ ] No tests rely on external services (without mocks)
- [ ] Tests are deterministic (no race conditions)
- [ ] Tests clean up after themselves
- [ ] Documentation is updated

## Future Enhancements

### Short-term
- [ ] Set up CI/CD with GitHub Actions
- [ ] Add test coverage reporting
- [ ] Create test helpers package
- [ ] Add more unit tests for each package

### Medium-term
- [ ] Add integration tests for full workflows
- [ ] Add benchmarks for performance-critical code
- [ ] Implement property-based testing
- [ ] Add mutation testing

### Long-term
- [ ] Add E2E tests with real Claude CLI (optional)
- [ ] Performance regression testing
- [ ] Fuzz testing for input validation

## Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Advanced Testing Patterns](https://medium.com/@matryer/5-simple-tips-and-tricks-for-writing-unit-tests-in-golang-619653f90742)

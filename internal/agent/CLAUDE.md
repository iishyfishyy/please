# internal/agent - LLM Agent Abstraction

## Overview

The `agent` package provides an abstraction layer for LLM providers, allowing `please` to work with different AI backends while maintaining a consistent interface. Currently implements Claude CLI integration, with architecture designed for easy extension.

## Architecture

### Agent Interface (agent.go)

```go
type Agent interface {
    // TranslateToCommand converts natural language to shell command
    TranslateToCommand(ctx context.Context, request string) (string, error)

    // RefineCommand modifies existing command based on user feedback
    RefineCommand(ctx context.Context, originalCommand, modificationRequest string) (string, error)
}
```

**Design Philosophy**:
- **Simple Interface**: Only two methods, focused on core functionality
- **Context-Aware**: Accept context.Context for cancellation/timeouts
- **Stateless**: Each call is independent (no session management in interface)
- **Error Transparent**: Return errors for caller to handle

### Claude Implementation (claude.go)

**Current Implementation**:
```go
type ClaudeAgent struct{}

func NewClaudeAgent() *ClaudeAgent
func IsClaudeCLIInstalled() bool
func (c *ClaudeAgent) TranslateToCommand(ctx, request) (command, error)
func (c *ClaudeAgent) RefineCommand(ctx, originalCommand, modificationRequest) (command, error)
func (c *ClaudeAgent) buildSystemPrompt() string
func (c *ClaudeAgent) callClaude(ctx, prompt) (output, error)
```

## How It Works

### 1. TranslateToCommand Flow

```
User Request: "find large files over 100MB"
        ↓
buildSystemPrompt() ← Creates context-aware system prompt
        ↓
Format full prompt with request
        ↓
callClaude() ← Executes: claude "{prompt}"
        ↓
Parse output, trim whitespace
        ↓
Return: "find . -type f -size +100M"
```

### 2. RefineCommand Flow

```
Original: "ls -la"
Modification: "only show hidden files"
        ↓
buildSystemPrompt() ← Same context gathering
        ↓
Format prompt with original + modification
        ↓
callClaude() ← LLM understands intent
        ↓
Return: "ls -lad .*"
```

### 3. System Prompt Structure (buildSystemPrompt)

**Current Components**:
1. **Role Definition**: "You are a helpful assistant that translates..."
2. **Environment Context**: OS, Shell
3. **Guidelines**: Safety, portability, format requirements
4. **Examples**: Few-shot learning with common patterns

**Environment Detection**:
```go
osInfo := runtime.GOOS                    // "darwin", "linux", "windows"
shell := os.Getenv("SHELL")               // "/bin/zsh", "/bin/bash", etc.
if shell == "" { shell = "/bin/sh" }
```

## Prompt Engineering Best Practices

### Current Prompt Analysis

**Strengths**:
- Clear role definition
- Environment awareness (OS, shell)
- Safety guidelines
- Few-shot examples
- Explicit format requirements (no markdown, no explanations)

**Limitations**:
- No current directory context
- No file type awareness
- No command history patterns
- Static examples (not adaptive)
- No token optimization

### Recommended Enhancements

#### 1. Add Context Gathering Helper

```go
// Add to ClaudeAgent
func (c *ClaudeAgent) gatherContext(ctx context.Context) (string, error) {
    var context strings.Builder

    // Current directory
    if cwd, err := os.Getwd(); err == nil {
        context.WriteString(fmt.Sprintf("Current directory: %s\n", cwd))
    }

    // File types present (token-efficient summary)
    if files := summarizeDirectory(); files != "" {
        context.WriteString(fmt.Sprintf("Files present: %s\n", files))
    }

    // Recent history patterns (future enhancement)
    // if patterns := getRecentPatterns(); patterns != "" {
    //     context.WriteString(fmt.Sprintf("Recent patterns: %s\n", patterns))
    // }

    return context.String(), nil
}

func summarizeDirectory() string {
    // Read directory, count file types
    // Return: "5 .go files, 3 .md files, 1 .json"
    // Keep it brief to save tokens
}
```

#### 2. Enhanced System Prompt

```go
func (c *ClaudeAgent) buildSystemPrompt() string {
    osInfo := runtime.GOOS
    shell := os.Getenv("SHELL")
    if shell == "" { shell = "/bin/sh" }

    // Gather runtime context
    contextInfo, _ := c.gatherContext(context.Background())

    return fmt.Sprintf(`You are a command-line expert that translates natural language into shell commands.

ENVIRONMENT:
- Operating System: %s
- Shell: %s
%s

CRITICAL RULES:
1. Output ONLY the raw command - no explanations, no markdown, no code blocks
2. Generate safe, correct shell commands for the current environment
3. Prefer portable commands when possible
4. Make reasonable assumptions for ambiguous requests
5. Consider the current directory context when generating commands

SAFETY GUIDELINES:
- Avoid destructive commands without clear intent (rm -rf, dd, etc.)
- Use common Unix/Linux utilities when available
- Respect the user's shell and OS capabilities

EXAMPLES:
Request: "list all files"
Command: ls -la

Request: "find javascript files modified today"
Command: find . -name "*.js" -mtime -1

Request: "count lines in go files"
Command: find . -name "*.go" -exec wc -l {} + | tail -1

Remember: Respond with ONLY the command itself.`,
        osInfo, shell, contextInfo)
}
```

#### 3. Token-Efficient Context

**Strategy**: Prioritize most relevant context, omit unnecessary details

```go
func summarizeDirectory() string {
    entries, err := os.ReadDir(".")
    if err != nil || len(entries) == 0 {
        return ""
    }

    // Count by extension (limit to top 5)
    counts := make(map[string]int)
    for _, entry := range entries {
        if entry.IsDir() {
            counts["directories"]++
        } else {
            ext := filepath.Ext(entry.Name())
            if ext != "" {
                counts[ext]++
            }
        }
    }

    // Build compact summary
    var parts []string
    for ext, count := range counts {
        if count > 0 {
            parts = append(parts, fmt.Sprintf("%d %s", count, ext))
        }
        if len(parts) >= 5 {  // Limit to save tokens
            break
        }
    }

    return strings.Join(parts, ", ")
}
```

## Adding a New LLM Provider

### Step-by-Step Guide

#### 1. Create Provider File

Create `internal/agent/{provider}.go`:

```go
package agent

import "context"

type OpenAIAgent struct {
    apiKey string
    // other config
}

func NewOpenAIAgent(apiKey string) *OpenAIAgent {
    return &OpenAIAgent{apiKey: apiKey}
}

func (o *OpenAIAgent) TranslateToCommand(ctx context.Context, request string) (string, error) {
    // Build prompt
    // Call OpenAI API
    // Parse response
    // Return command
}

func (o *OpenAIAgent) RefineCommand(ctx context.Context, originalCommand, modificationRequest string) (string, error) {
    // Similar to above
}
```

#### 2. Update Config Types

In `internal/config/config.go`:

```go
const (
    AgentClaude AgentType = "claude-code"
    AgentOpenAI AgentType = "openai"    // Add new agent
)
```

#### 3. Update Main Command

In `cmd/please/main.go`:

```go
// In runConfigure()
agentChoice, err := ui.ConfigureAgent()  // Update to include new option

// In runCommand()
switch cfg.Agent {
case config.AgentClaude:
    ag = agent.NewClaudeAgent()
case config.AgentOpenAI:
    ag = agent.NewOpenAIAgent(cfg.OpenAIKey)  // Handle config
default:
    return fmt.Errorf("unknown agent type: %s", cfg.Agent)
}
```

#### 4. Update UI

In `internal/ui/prompt.go`:

```go
func ConfigureAgent() (string, error) {
    var agent string
    prompt := &survey.Select{
        Message: "Select an LLM agent:",
        Options: []string{"Claude Code", "OpenAI"},  // Add option
        Default: "Claude Code",
    }
    // ...
}
```

#### 5. Add Validation

```go
func IsOpenAIConfigured() bool {
    // Check API key, validate connection, etc.
}
```

### Provider Implementation Checklist

When implementing a new provider:

- [ ] Implement `Agent` interface
- [ ] Add provider detection/validation
- [ ] Handle authentication (API keys, CLI tools, etc.)
- [ ] Build appropriate system prompts for the model
- [ ] Test with various command types
- [ ] Handle errors gracefully
- [ ] Consider rate limiting
- [ ] Add timeout handling
- [ ] Document configuration requirements
- [ ] Update README.md with setup instructions

## Prompt Engineering Guidelines

### Format Requirements

**Critical**: Enforce strict output format to prevent parsing issues

```
IMPORTANT: Respond with ONLY the command itself, nothing else.
No explanations, no markdown, no code blocks. Just the raw command.
```

**Why**: Survey prompts and execution expect raw commands without formatting

### Safety in Prompts

Include safety guidelines in system prompt:

```
SAFETY GUIDELINES:
- Avoid destructive commands without clear intent (rm -rf, dd, etc.)
- Do not generate commands that modify system files
- Prefer safer alternatives when available
```

**Limitations**: LLM may still generate dangerous commands - user review is final safety check

### Examples Selection

**Few-shot learning improves accuracy**:

```go
// Good: Diverse, common use cases
"list all files" → "ls -la"
"find javascript files" → "find . -name '*.js'"
"show disk usage" → "df -h"

// Better: Context-aware examples
// If in git repo:
"show recent commits" → "git log -10 --oneline"

// If Go files present:
"run tests" → "go test ./..."
```

### Context Prioritization

**Token Budget**: ~1000 tokens for system prompt

**Priority Order**:
1. Role and format requirements (essential)
2. Environment (OS, shell) - 50 tokens
3. Current directory - 20 tokens
4. File types present - 30 tokens
5. Safety guidelines - 100 tokens
6. Examples - 200 tokens
7. Recent patterns (future) - 100 tokens

## Error Handling

### Common Error Scenarios

1. **Claude CLI Not Installed**
   ```go
   IsClaudeCLIInstalled() // Check before use
   ```

2. **Authentication Failed**
   ```go
   // Test in configure step
   _, err := testAgent.TranslateToCommand(ctx, "echo hello")
   ```

3. **Empty Response**
   ```go
   if output == "" {
       return "", fmt.Errorf("claude CLI returned empty response")
   }
   ```

4. **Command Execution Failed**
   ```go
   if err := cmd.Run(); err != nil {
       return "", fmt.Errorf("failed to call claude CLI: %w\nStderr: %s", err, stderr.String())
   }
   ```

### Best Practices

- **Wrap Errors**: Always use `fmt.Errorf("context: %w", err)`
- **Include Details**: Stderr output helps debugging
- **User-Friendly**: Main.go translates to user-friendly messages
- **Don't Panic**: Return errors, let caller handle

## Testing Strategy

### Mock Agent for Testing

```go
// agent_test.go
type MockAgent struct {
    TranslateFn func(context.Context, string) (string, error)
    RefineFn    func(context.Context, string, string) (string, error)
}

func (m *MockAgent) TranslateToCommand(ctx context.Context, request string) (string, error) {
    if m.TranslateFn != nil {
        return m.TranslateFn(ctx, request)
    }
    return "echo test", nil
}

// Use in tests:
mockAgent := &MockAgent{
    TranslateFn: func(ctx context.Context, req string) (string, error) {
        if req == "list files" {
            return "ls -la", nil
        }
        return "", fmt.Errorf("unexpected request")
    },
}
```

### Test Coverage Areas

1. **Interface Contract**
   - Both methods return valid commands
   - Errors propagate correctly
   - Context cancellation works

2. **Claude Agent**
   - System prompt includes all required components
   - CLI detection works
   - Output parsing handles various formats
   - Error messages are informative

3. **Prompt Engineering**
   - Context gathering doesn't fail on edge cases
   - Token limits respected
   - Examples are valid

## Performance Considerations

### Latency Sources

1. **Claude CLI Startup**: ~500ms-2s
2. **LLM Processing**: ~1-3s
3. **Context Gathering**: ~10-50ms

**Total**: 2-5 seconds per command generation

### Optimization Opportunities

1. **Context Caching**: Cache directory summaries for 10s
2. **Parallel Calls**: Don't block on slow context gathering
3. **Streaming** (future): Stream LLM output for faster UX

## Security Considerations

### Prompt Injection

**Risk**: User input is included in prompts
**Mitigation**:
- Claude CLI handles API interaction
- No direct API key exposure
- Format requirements prevent code injection

### Command Safety

**Risk**: LLM generates destructive commands
**Mitigation**:
- User review before execution (primary defense)
- Safety guidelines in prompt (secondary)
- Future: Command pattern detection

### Data Privacy

- No command history sent to LLM (currently)
- User requests visible to Claude API
- Future: Option to disable telemetry

## Future Enhancements

### Short-term
- [ ] Implement context gathering helper
- [ ] Add directory summarization
- [ ] Optimize token usage
- [ ] Add comprehensive tests

### Medium-term
- [ ] Command history patterns in prompts
- [ ] Adaptive examples based on context
- [ ] Caching for repeated requests
- [ ] Streaming output support

### Long-term
- [ ] Fine-tuned model for shell commands
- [ ] Local LLM support (privacy-focused)
- [ ] Multi-turn conversation for complex tasks
- [ ] Learning from user corrections

## Quick Reference

### Adding Context to Prompts
1. Add to `buildSystemPrompt()` in claude.go
2. Keep token-efficient (< 100 tokens)
3. Test with various scenarios
4. Verify format still works

### Debugging Prompts
```bash
# Test Claude CLI directly
claude "You are a shell expert. Convert: find large files"

# Check what's being sent
# Add logging to callClaude() temporarily
fmt.Fprintf(os.Stderr, "Prompt: %s\n", prompt)
```

### Testing New Providers
1. Implement Agent interface
2. Test with: "echo hello" (simple)
3. Test with: "find javascript files modified today" (complex)
4. Test refinement: "list files" → "only hidden ones"
5. Test error handling: invalid requests

## Related Documentation

- **Root CLAUDE.md**: Overall architecture
- **internal/config/CLAUDE.md**: Adding config for new providers
- **internal/executor/CLAUDE.md**: How commands are executed

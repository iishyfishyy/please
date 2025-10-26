# please - Natural Language Terminal Interface

## Project Overview

**please** is a natural language interface for the terminal that translates plain English requests into shell commands using Claude AI. It provides an interactive, safe-by-default command generation workflow with iterative refinement and history tracking.

### Core Philosophy
- **Safety First**: Always show commands before execution and ask for confirmation
- **Iterative Refinement**: Allow users to modify commands with natural language
- **Simplicity**: Hide complexity, expose natural language interface
- **Portability**: Work across macOS, Linux, and Windows

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     User Input                              │
│              "find large files over 100MB"                  │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              cmd/please/main.go (Cobra CLI)                 │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Commands:                                            │   │
│  │ - Root command: Natural language → shell command    │   │
│  │ - configure: Setup wizard for Claude CLI            │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────┬────────────────────────────┬───────────────────┘
             │                            │
             ▼                            ▼
┌────────────────────────┐    ┌──────────────────────────┐
│  internal/config/      │    │  internal/history/       │
│  ┌──────────────────┐  │    │  ┌────────────────────┐  │
│  │ ~/.please/       │  │    │  │ ~/.please/         │  │
│  │   config.json    │  │    │  │   history.json     │  │
│  │ - Agent type     │  │    │  │ - Past commands    │  │
│  │ - Future configs │  │    │  │ - Modifications    │  │
│  └──────────────────┘  │    │  │ - Execution status │  │
└────────────────────────┘    │  └────────────────────┘  │
                              └──────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│              internal/agent/agent.go (Interface)            │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ type Agent interface {                               │   │
│  │   TranslateToCommand(ctx, request) (command, err)    │   │
│  │   RefineCommand(ctx, cmd, modification) (cmd, err)   │   │
│  │ }                                                     │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│          internal/agent/claude.go (Implementation)          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ - Calls Claude CLI via exec.Command()                │   │
│  │ - Builds context-aware system prompts                │   │
│  │ - Handles OS/shell detection                         │   │
│  │ - Enforces safety guidelines                         │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                 internal/ui/prompt.go                       │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Interactive prompts using AlecAivazis/survey:        │   │
│  │ - ConfirmCommand: Run it / Modify it / Cancel        │   │
│  │ - PromptForModification: Natural language refinement │   │
│  │ - Colorized output with fatih/color                  │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│             internal/executor/executor.go                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ - Detects user's shell (SHELL env or /bin/sh)       │   │
│  │ - Executes: shell -c "command"                       │   │
│  │ - Streams stdout/stderr to user's terminal           │   │
│  │ - Cross-platform (Unix shells, Windows cmd)          │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Package Responsibilities

### cmd/please/
**Purpose**: CLI entry point using spf13/cobra framework

**Key Files**:
- `main.go`: Defines commands, orchestrates workflow

**Responsibilities**:
- Parse command-line arguments
- Initialize configuration and history
- Coordinate agent, UI, and executor interactions
- Handle main command loop (generate → review → modify/run/cancel)

### internal/agent/
**Purpose**: LLM agent abstraction and implementations

**Key Files**:
- `agent.go`: Agent interface definition
- `claude.go`: Claude CLI implementation

**Responsibilities**:
- Define agent interface for future LLM providers
- Implement Claude CLI integration
- Build context-aware prompts
- Handle LLM communication and error handling

**See**: internal/agent/CLAUDE.md for detailed guidance

### internal/config/
**Purpose**: Configuration management

**Key Files**:
- `config.go`: Config struct, load/save operations

**Responsibilities**:
- Manage ~/.please/config.json
- Define configuration schema
- Handle config migrations (future)
- Provide config validation

**See**: internal/config/CLAUDE.md for extension patterns

### internal/executor/
**Purpose**: Safe command execution

**Key Files**:
- `executor.go`: Execute shell commands

**Responsibilities**:
- Detect user's shell environment
- Execute commands in appropriate shell
- Stream output to user's terminal
- Handle cross-platform compatibility

**See**: internal/executor/CLAUDE.md for security patterns

### internal/history/
**Purpose**: Command history tracking

**Key Files**:
- `history.go`: History struct, persistence

**Responsibilities**:
- Track all command generation attempts
- Record modifications and execution status
- Persist to ~/.please/history.json
- Enable future features (suggestions, analytics)

### internal/ui/
**Purpose**: User interaction and display

**Key Files**:
- `prompt.go`: Interactive prompts and displays

**Responsibilities**:
- Show generated commands with formatting
- Prompt for user action (run/modify/cancel)
- Collect modification requests
- Display success/error/info messages

## Development Workflows

### Setup Development Environment
```bash
# Clone the repository
git clone https://github.com/iishyfishyy/please.git
cd please

# Install dependencies
go mod download

# Build
go build -o please ./cmd/please

# Test (run tests when available)
go test ./...

# Install locally
go install ./cmd/please
```

### Adding a New Feature
1. Identify affected packages
2. Update interfaces if needed (maintain backward compatibility)
3. Implement feature
4. Add tests (see Testing Strategy below)
5. Update relevant CLAUDE.md files
6. Test manually with `please configure` and various commands
7. Update README.md if user-facing

### Adding a New LLM Provider
See internal/agent/CLAUDE.md for detailed guide. Summary:
1. Create new file: `internal/agent/{provider}.go`
2. Implement `Agent` interface
3. Add agent type to `internal/config/config.go`
4. Update configuration wizard in `cmd/please/main.go`
5. Add detection/validation for provider's CLI

### Debugging
```bash
# Build with debug info
go build -o please ./cmd/please

# Run with verbose output (add to code as needed)
PLEASE_DEBUG=1 ./please "your command"

# Check configuration
cat ~/.please/config.json

# Check history
cat ~/.please/history.json

# Test Claude CLI directly
claude "echo hello"
```

## Code Conventions

### Error Handling
- Always use `fmt.Errorf("context: %w", err)` to wrap errors
- Provide user-friendly error messages in UI
- Log technical details for debugging
- Never panic in user-facing code

### Context Usage
- Pass `context.Context` to all LLM calls
- Support cancellation for long-running operations
- Use `context.Background()` at CLI entry points
- Consider timeouts for future enhancements

### Testing Strategy (To Be Implemented)
```
please/
├── cmd/please/
│   └── main_test.go          # Integration tests
├── internal/agent/
│   ├── agent_test.go          # Interface contract tests
│   ├── claude_test.go         # Claude agent tests (mocked)
│   └── testdata/              # Test fixtures
├── internal/config/
│   └── config_test.go         # Config load/save/validation
├── internal/executor/
│   └── executor_test.go       # Shell execution tests (safe)
├── internal/history/
│   └── history_test.go        # History persistence tests
└── internal/ui/
    └── prompt_test.go         # UI logic tests (mocked survey)
```

**Testing Principles**:
- Mock external dependencies (Claude CLI, survey prompts)
- Test error paths thoroughly
- Use table-driven tests for multiple scenarios
- Test cross-platform behavior with build tags
- Never execute dangerous commands in tests

### Code Style
- Follow standard Go conventions (gofmt, golint)
- Keep functions focused and small
- Use descriptive variable names
- Comment complex logic, not obvious code
- Prefer composition over inheritance
- Keep interfaces small and focused

## Security Considerations

### Command Execution Safety
1. **Always Confirm**: Never execute without user approval
2. **Display Clearly**: Show exact command before running
3. **No Hidden Execution**: All commands visible to user
4. **Shell Injection**: Trust LLM output is safe, but warn users
5. **Destructive Commands**: Rely on user review

### Configuration Security
- No API keys stored (authentication via Claude CLI)
- Config files in user's home directory (0644)
- No sensitive data in history

### Future Considerations
- Add command allowlist/denylist
- Implement dry-run mode
- Add command explanation before execution
- Pattern detection for dangerous operations (rm -rf, dd, etc.)

## Dependencies

### Direct Dependencies
- **github.com/spf13/cobra**: CLI framework
- **github.com/AlecAivazis/survey/v2**: Interactive prompts
- **github.com/fatih/color**: Terminal colors

### External Tools
- **Claude CLI**: Required for LLM functionality
  - Install: https://github.com/anthropics/claude-cli
  - Authentication via `claude auth`
  - Called via `exec.Command("claude", prompt)`

## Release Process

See RELEASING.md for detailed guide. Summary:

1. **Tag Version**: `git tag -a v1.0.0 -m "Release v1.0.0"`
2. **Push Tag**: `git push origin v1.0.0`
3. **Automated**: GitHub Actions runs GoReleaser
4. **Outputs**:
   - GitHub release with binaries (macOS, Linux, Windows)
   - Homebrew formula update (requires setup)
   - Checksums and changelog

## Common Tasks for Claude Code

### "Add support for {LLM provider}"
1. Read internal/agent/CLAUDE.md
2. Create internal/agent/{provider}.go
3. Implement Agent interface
4. Update config.go with new agent type
5. Update configure command in main.go
6. Test with provider's CLI

### "Improve command generation accuracy"
1. Enhance buildSystemPrompt() in claude.go
2. Add context gathering (see internal/agent/CLAUDE.md)
3. Test with various command types
4. Consider token efficiency

### "Add configuration option {feature}"
1. Read internal/config/CLAUDE.md
2. Update Config struct
3. Update Load/Save logic
4. Handle backward compatibility
5. Update configure wizard

### "Fix cross-platform issue with {feature}"
1. Check internal/executor/CLAUDE.md
2. Test on target platform
3. Use runtime.GOOS for platform detection
4. Consider shell differences

### "Add tests for {package}"
1. Create {package}_test.go
2. Follow testing patterns in this file
3. Mock external dependencies
4. Use table-driven tests
5. Run with `go test ./...`

## Future Roadmap

### Near-term
- [ ] Add comprehensive test suite
- [ ] Support for additional LLM providers (Codex, Goose)
- [ ] Command explanation mode
- [ ] Dry-run mode
- [ ] Shell completion

### Medium-term
- [ ] Command history-based suggestions
- [ ] Alias creation for frequent commands
- [ ] Multi-step command workflows
- [ ] Context-aware suggestions (based on directory contents)

### Long-term
- [ ] Learning from user corrections
- [ ] Integration with system package managers
- [ ] Plugin system for custom agents
- [ ] Web dashboard for history/analytics

## Getting Help

- **Documentation**: README.md, RELEASING.md, package CLAUDE.md files
- **Issues**: https://github.com/iishyfishyy/please/issues
- **Code Examples**: Read existing implementations in internal/

## Quick Reference for Claude Code Sessions

When working on this project:
1. **Read package CLAUDE.md first** for specific guidance
2. **Check existing patterns** in similar code
3. **Test manually** with `please configure` and various commands
4. **Consider cross-platform** compatibility (macOS, Linux, Windows)
5. **Think about token efficiency** in prompt engineering
6. **Prioritize user safety** in command execution
7. **Add tests** when implementing new features

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
│  │ - configure: Setup wizard (Claude + Custom Cmds)    │   │
│  │ - index: Index custom command documentation         │   │
│  │ - list-commands: Show all custom commands           │   │
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
│  │ - Custom cmds    │  │    │  │ - Modifications    │  │
│  │   - Provider     │  │    │  │ - Execution status │  │
│  │   - Matching     │  │    │  └────────────────────┘  │
│  └──────────────────┘  │    └──────────────────────────┘
└───────────┬────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│            internal/customcmd/                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Custom Command Manager (RAG System)                  │   │
│  │ - Loader: Scans ~/.please/commands/*.md              │   │
│  │ - Parser: Extracts YAML frontmatter + examples       │   │
│  │ - Matcher: Keyword-based or hybrid semantic search   │   │
│  │ - Embeddings: Ollama (local) or OpenAI (API)         │   │
│  │ - VectorStore: In-memory cosine similarity search    │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
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
- Custom command configuration (embedding provider, matching strategy)

**See**: internal/config/CLAUDE.md for extension patterns

### internal/customcmd/
**Purpose**: RAG-powered custom command documentation system

**Key Files**:
- `customcmd.go`: Manager for loading and retrieving custom commands
- `loader.go`: Scans and loads .md files from ~/.please/commands/
- `parser.go`: Parses YAML frontmatter and examples
- `matcher.go`: Keyword-based matching with scoring
- `semantic.go`: Hybrid semantic search (keyword + embeddings)
- `setup.go`: Automated setup for Ollama and OpenAI
- `embeddings/`: Embedding providers (Ollama, OpenAI)
- `vectorstore/`: In-memory vector storage with cosine similarity

**Responsibilities**:
- Load custom command documentation from markdown files
- Parse YAML frontmatter (command, aliases, keywords, priority, etc.)
- Extract user request → command examples
- Match user requests to relevant custom commands
- Provide context to LLM agent for proprietary/internal tools
- Support keyword-only or hybrid semantic matching
- Manage embedding generation (local or API-based)
- Handle auto-indexing and manual reindexing

**See**: internal/customcmd/CLAUDE.md for detailed implementation guide

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

## Custom Commands Feature

### Overview

The custom commands feature allows users to teach `please` about proprietary, internal, or specialized tools by providing documentation in simple markdown files. This uses **RAG (Retrieval Augmented Generation)** to enhance the LLM's knowledge with your custom tool documentation.

**Key Benefits**:
- 🎯 Support for internal/proprietary tools unknown to Claude
- 📚 Document-based knowledge (no code changes needed)
- 🔍 Smart matching: keyword-based (fast) or semantic (accurate)
- 🏠 Privacy options: local embeddings (Ollama) or API (OpenAI)
- 🚀 Auto-indexing with staleness detection

### How It Works

```
User Request: "deploy my app to staging"
                    ↓
    ┌───────────────────────────────┐
    │ Custom Command Matcher        │
    │ - Scans ~/.please/commands/   │
    │ - Keyword or semantic search  │
    └───────────────┬───────────────┘
                    ↓
    ┌───────────────────────────────┐
    │ Finds: deploy-tool.md         │
    │ - Command: deploy-tool        │
    │ - Keywords: deploy, staging   │
    │ - Examples: 15 request→cmd    │
    └───────────────┬───────────────┘
                    ↓
    ┌───────────────────────────────┐
    │ Agent receives context:       │
    │ "The user has deploy-tool...  │
    │  Example: 'deploy to staging' │
    │  → 'deploy-tool --env=staging'│
    └───────────────┬───────────────┘
                    ↓
    Generates accurate command using custom context!
```

### Setup

**Step 1: Configure Custom Commands**

```bash
please configure
# Select: "Would you like to configure custom commands?"
# Choose provider:
#   - None (keyword-only matching)
#   - Ollama (local embeddings, private)
#   - OpenAI (API-based, accurate)
```

**Ollama Setup** (Recommended for privacy):
- Tool offers to install Ollama via Homebrew (macOS) or script (Linux)
- Automatically downloads `nomic-embed-text` model (384 dimensions)
- Tests connection to localhost:11434

**OpenAI Setup** (Recommended for accuracy):
- Enter API key (or use `OPENAI_API_KEY` env var)
- Choose to store in config or use env var
- Tests connection and validates key

**Step 2: Create Command Documentation**

```bash
# Directory is auto-created during setup
cd ~/.please/commands/

# Create a markdown file for your tool
# Filename: {command-name}.md
```

**Example: kubectl.md**

```markdown
---
command: kubectl
aliases: ["k8s", "kube"]
keywords: ["kubernetes", "pods", "deployments", "services"]
categories: ["devops", "containers"]
priority: high
version: "1.28"
---

# Kubernetes kubectl

kubectl is the Kubernetes command-line tool.

## Common Patterns

- List pods: `kubectl get pods`
- Describe resource: `kubectl describe {type} {name}`
- Apply manifest: `kubectl apply -f {file}`

## Examples

**User**: "show me all pods"
**Command**: `kubectl get pods`

**User**: "get logs from nginx pod"
**Command**: `kubectl logs nginx`

**User**: "deploy from manifest.yaml"
**Command**: `kubectl apply -f manifest.yaml`
```

**Step 3: Index Your Commands**

```bash
# Manual indexing
please index

# Auto-indexing
# Happens automatically on first use or when files change
```

**Step 4: Use Custom Commands**

```bash
please "show all pods"
# Will match kubectl.md and generate: kubectl get pods

please "deploy my app to staging"
# If you have deploy-tool.md, will use that context
```

### Matching Strategies

**1. Keyword-Only (Default, No Embeddings)**

Fast, no dependencies, works offline.

Scoring algorithm:
- Command name match: 100 points
- Alias match: 80 points
- Keyword match: 10 points each
- Example similarity: 15 points per word overlap
- Priority multipliers: high=1.3x, medium=1.1x

**2. Hybrid (Keyword + Semantic)**

Best accuracy, requires Ollama or OpenAI.

Strategy:
1. Try keyword matching first (fast path)
2. If scores meet threshold, use keyword results
3. Otherwise, fall back to semantic search
4. Semantic uses vector embeddings + cosine similarity

**Configuration** (in ~/.please/config.json):

```json
{
  "customCommands": {
    "enabled": true,
    "provider": "ollama",  // or "openai" or "none"
    "matching": {
      "strategy": "hybrid",  // or "keyword" or "semantic"
      "maxDocsToRetrieve": 3,
      "scoreThreshold": 50
    },
    "ollama": {
      "baseURL": "http://localhost:11434",
      "model": "nomic-embed-text"
    }
  }
}
```

### File Format Specification

**YAML Frontmatter** (required):
```yaml
---
command: tool-name          # Primary command name (required)
aliases: ["alt1", "alt2"]   # Alternative names (optional)
keywords: ["key1", "key2"]  # Keywords for matching (optional)
categories: ["cat1"]        # Categories (optional)
priority: high              # high/medium/low (optional)
version: "1.0"              # Tool version (optional)
---
```

**Markdown Content**:
- Headers, paragraphs, code blocks (all indexed)
- Examples section (special parsing):

```markdown
## Examples

**User**: "natural language request"
**Command**: `actual command`

**User**: "another request"
**Command**: `another command`
```

**Parsing**:
- Lines starting with `**User**:` → user request
- Lines starting with `**Command**:` → command (backticks removed)
- Up to 5 examples per command sent to LLM (token budget)

### Token Budget Management

To keep prompts efficient:
- **Max 3 commands** matched per request
- **Max 5 examples** per command sent to LLM
- **Max 10 common patterns** extracted from content
- Automatic truncation if content is too long

### CLI Commands

```bash
# Index custom commands
please index
# Output:
# ✓ Indexed 5 commands from ~/.please/commands/
# - kubectl (15 examples)
# - docker (12 examples)
# - ...

# List all custom commands
please list-commands
# Output:
# Custom Commands (Provider: ollama)
#
# kubectl (k8s, kube)
#   Categories: devops, containers
#   Priority: high
#   Examples: 15
#   File: kubectl.md
#
# docker
#   Categories: devops, containers
#   Examples: 12
#   File: docker.md

# Reconfigure custom commands
please configure
```

### Auto-Indexing Behavior

The manager automatically detects when indexing is needed:

1. **First Use**: No index exists → prompts to run `please index`
2. **Stale Index**: Files modified since last index → prompts to reindex
3. **Manual**: User runs `please index` explicitly

**Detection Logic**:
```go
func (m *Manager) NeedsReindex() bool {
    // Check if any .md file is newer than index timestamp
    // Return true if stale
}
```

### Advanced: Embedding Providers

**Ollama** (Local, Private):
- Model: `nomic-embed-text` (384 dimensions)
- API: `http://localhost:11434/api/embeddings`
- Cost: Free, runs locally
- Latency: ~50-100ms per embedding
- Privacy: 100% local, no data sent externally

**OpenAI** (API, Accurate):
- Model: `text-embedding-3-small` (1536 dimensions)
- API: `https://api.openai.com/v1/embeddings`
- Cost: $0.02 per 1M tokens (~$0.0001 per index)
- Latency: ~100-200ms per embedding
- Privacy: Data sent to OpenAI

**Vector Store**:
- In-memory storage (no persistence yet)
- Cosine similarity for search
- Thread-safe with sync.RWMutex

**Similarity Calculation**:
```go
func cosineSimilarity(a, b []float32) float32 {
    dotProduct := sum(a[i] * b[i])
    normA := sqrt(sum(a[i] * a[i]))
    normB := sqrt(sum(b[i] * b[i]))
    return dotProduct / (normA * normB)
}
```

### Troubleshooting

**"No custom commands found"**:
- Run `please index` to index your commands
- Check `~/.please/commands/` has .md files
- Ensure files have valid YAML frontmatter

**"Failed to connect to Ollama"**:
- Verify Ollama is running: `ollama serve` or check if service is active
- Test manually: `curl http://localhost:11434/api/embeddings -d '{"model":"nomic-embed-text","prompt":"test"}'`
- Check config has correct base URL

**"OpenAI API error"**:
- Verify API key is valid
- Check `OPENAI_API_KEY` env var or config
- Ensure account has credits

**"Commands not matching"**:
- Check keywords in frontmatter match your query words
- Add more examples to your .md files
- Try hybrid or semantic strategy if using keyword-only
- Increase `scoreThreshold` in config if getting too many irrelevant results

### Best Practices

1. **Comprehensive Examples**: Add 10-15 diverse examples per command
2. **Good Keywords**: Include synonyms, abbreviations, related terms
3. **Clear Aliases**: Add common alternative names
4. **Priority Levels**: Set priority=high for most-used tools
5. **Categories**: Group related commands for future features
6. **Regular Updates**: Keep docs current as tools evolve
7. **Test Matching**: Use `please list-commands` to verify indexing

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

### "Add custom command documentation for {tool}"
1. Create ~/.please/commands/{tool}.md
2. Add YAML frontmatter (command, keywords, aliases, priority)
3. Add 10-15 diverse examples with User/Command pattern
4. Run `please index` to index the new documentation
5. Test with `please "{natural language query}"`
6. Verify with `please list-commands`

### "Add new embedding provider"
1. Read internal/customcmd/embeddings/embedder.go interface
2. Create internal/customcmd/embeddings/{provider}.go
3. Implement Embedder interface (Embed, EmbedBatch, Dimensions, Name)
4. Add provider type to config.go EmbeddingProvider enum
5. Update setup.go with provider-specific setup logic
6. Add connection testing
7. Test with `please configure`

### "Improve custom command matching accuracy"
1. Review matcher.go scoring algorithm
2. Add more weight to certain match types
3. Consider adding fuzzy matching for typos
4. Test with edge cases (synonyms, abbreviations)
5. Monitor false positives/negatives

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

### Completed ✅
- [x] Custom commands with RAG (Week 1-2 implementation)
- [x] Keyword-based matching
- [x] Hybrid semantic search (Ollama + OpenAI)
- [x] Auto-indexing with staleness detection
- [x] Command documentation via markdown files
- [x] Setup wizard for embedding providers

### Near-term
- [ ] Add comprehensive test suite
  - [ ] Unit tests for keyword matcher
  - [ ] Unit tests for embedders (mocked)
  - [ ] Integration tests for hybrid matcher
  - [ ] End-to-end tests for custom commands
- [ ] Persistent vector store (save index to disk)
- [ ] Support for additional LLM providers (Codex, Goose)
- [ ] Command explanation mode
- [ ] Dry-run mode
- [ ] Shell completion
- [ ] Additional embedding providers (Cohere, local models)

### Medium-term
- [ ] Command history-based suggestions
- [ ] Alias creation for frequent commands
- [ ] Multi-step command workflows
- [ ] Context-aware suggestions (based on directory contents)
- [ ] Custom command sharing/marketplace
- [ ] Chunking strategy for long documentation
- [ ] Incremental indexing (only index changed files)
- [ ] Command usage analytics

### Long-term
- [ ] Learning from user corrections
- [ ] Integration with system package managers
- [ ] Plugin system for custom agents
- [ ] Web dashboard for history/analytics
- [ ] Collaborative custom command repositories
- [ ] Auto-discovery of tools in PATH
- [ ] Integration with man pages and --help output

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

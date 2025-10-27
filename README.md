# please

A natural language interface for your terminal. Simply describe what you want to do in plain English, and `please` will translate it into the right command and execute it.

## Features

- 🗣️ **Natural language commands** - No need to remember complex syntax
- 🤖 **Powered by Claude** - Uses Claude AI for accurate command translation
- ✅ **Interactive confirmation** - Review commands before execution
- ✏️ **Iterative refinement** - Modify commands with natural language
- 📝 **Command history** - Tracks all your requests and executions
- 🔒 **Safe by default** - Always asks for confirmation before running
- 📚 **Custom commands** - Teach `please` about your internal/proprietary tools
- 🔍 **Smart matching** - Keyword or semantic search for custom commands
- 🏠 **Privacy options** - Local embeddings (Ollama) or cloud API (OpenAI)

## Installation

### Prerequisites

- Claude CLI ([installation instructions](https://github.com/anthropics/claude-cli))

First, install and authenticate with Claude CLI:

```bash
# Install Claude CLI (follow official instructions)
# Then authenticate:
claude auth
```

### Quick Install

Choose your preferred installation method:

#### Homebrew (macOS/Linux)

```bash
brew install iishyfishyy/tap/please
```

#### Install Script (macOS/Linux/Windows)

```bash
curl -sSL https://raw.githubusercontent.com/iishyfishyy/please/main/install.sh | bash
```

#### Go Install

```bash
go install github.com/iishyfishyy/please/cmd/please@latest
```

#### Download Binary

Download the latest release for your platform from the [releases page](https://github.com/iishyfishyy/please/releases/latest).

#### Build from Source

```bash
git clone https://github.com/iishyfishyy/please.git
cd please
go build -o please ./cmd/please
sudo mv please /usr/local/bin/
```

## Quick Start

### 1. Configure

Run the configuration wizard:

```bash
please configure
```

This will:
- Check if Claude CLI is installed and authenticated
- Verify the connection is working
- Save your configuration to `~/.please/config.json`

### 2. Use it

Simply prefix your request with `please`:

```bash
please "list all files"
please "find all javascript files modified in the last week"
please "show disk usage sorted by size"
please "create a git branch called feature-x"
please "compress all images in this directory"
```

### 3. Review and execute

`please` will:
1. 🤔 Generate the appropriate command
2. 📋 Show you the command for review
3. ❓ Ask what you want to do:
   - **Run it** - Execute the command
   - **Modify it** - Refine with natural language
   - **Cancel** - Exit without running

### Example session

```bash
$ please "find large files over 100MB"

Thinking...

Generated command:
  find . -type f -size +100M

What would you like to do?
> Run it
  Modify it
  Cancel

./videos/demo.mp4
./datasets/training.csv
```

### Modifying commands

If the generated command isn't quite right, select "Modify it":

```bash
$ please "list all files"

Generated command:
  ls -la

What would you like to do?
  Run it
> Modify it
  Cancel

How would you like to modify the command?
> only show hidden files

Generated command:
  ls -lad .*

What would you like to do?
> Run it
```

## Configuration

Configuration is stored in `~/.please/config.json`:

```json
{
  "agent": "claude-code",
  "customCommands": {
    "enabled": true,
    "provider": "ollama",
    "matching": {
      "strategy": "hybrid",
      "maxDocsToRetrieve": 3,
      "scoreThreshold": 50
    }
  }
}
```

Authentication is handled by the Claude CLI, so no API keys are stored in the config (unless you choose to store OpenAI key there).

## Custom Commands

### What are custom commands?

Custom commands allow you to teach `please` about your proprietary, internal, or specialized tools that Claude might not know about. Simply create markdown files with examples, and `please` will use them to generate accurate commands.

**Perfect for**:
- 🏢 Internal company tools
- 🔧 Custom scripts and utilities
- 📦 Specialized domain tools
- 🚀 Deployment pipelines

### Quick Start

**1. Configure custom commands**:

```bash
please configure
# Choose: "Would you like to configure custom commands?"
# Select provider:
#   - None (keyword-only matching, fast, no dependencies)
#   - Ollama (local embeddings, privacy-focused, free)
#   - OpenAI (cloud embeddings, most accurate, requires API key)
```

**2. Create a command file** in `~/.please/commands/`:

```bash
cd ~/.please/commands/
cat > deploy-tool.md << 'EOF'
---
command: deploy-tool
aliases: ["deploy", "dt"]
keywords: ["deploy", "staging", "production", "release"]
priority: high
---

# Internal Deployment Tool

Our custom deployment tool for managing releases.

## Examples

**User**: "deploy to staging"
**Command**: `deploy-tool --env=staging --confirm`

**User**: "deploy version 1.2.3 to production"
**Command**: `deploy-tool --env=production --version=1.2.3 --confirm`

**User**: "rollback production"
**Command**: `deploy-tool --env=production --rollback`
EOF
```

**3. Index your commands**:

```bash
please index
```

**4. Use it**:

```bash
$ please "deploy to staging"

Generated command:
  deploy-tool --env=staging --confirm

# The LLM now knows about your internal tool!
```

### File Format

Create markdown files in `~/.please/commands/{tool-name}.md`:

```markdown
---
command: kubectl              # Primary command name (required)
aliases: ["k8s", "kube"]     # Alternative names (optional)
keywords: ["kubernetes", "pods", "deployments"]  # Search keywords
categories: ["devops"]       # Organizational categories
priority: high               # high/medium/low (affects matching)
---

# Command Description

Brief description of what this tool does.

## Examples

**User**: "natural language request"
**Command**: `actual command to run`

**User**: "another request"
**Command**: `another command`
```

### Matching Strategies

**Keyword-only** (Default):
- Fast, no dependencies
- Works offline
- Good for exact keyword matches

**Hybrid** (Recommended):
- Tries keyword matching first (fast)
- Falls back to semantic search if needed
- Requires Ollama (local) or OpenAI (API)
- Best accuracy

**Semantic-only**:
- Always uses semantic search
- Understands synonyms and paraphrasing
- Slower but most accurate

Configure in `~/.please/config.json`:
```json
{
  "customCommands": {
    "matching": {
      "strategy": "hybrid"  // or "keyword" or "semantic"
    }
  }
}
```

### Embedding Providers

**Ollama** (Local, Private):
- ✅ 100% local, no data sent externally
- ✅ Free, no API costs
- ✅ Fast (~50-100ms per search)
- ⚠️ Requires Ollama installed (`brew install ollama`)
- Model: `nomic-embed-text` (auto-downloaded)

**OpenAI** (Cloud, Accurate):
- ✅ Most accurate semantic matching
- ✅ No local setup required
- ✅ Very cheap ($0.02 per 1M tokens)
- ⚠️ Requires API key
- ⚠️ Data sent to OpenAI
- Model: `text-embedding-3-small`

Setup during `please configure`:
```bash
please configure
# Select embedding provider
# Ollama: Tool offers to install and configure automatically
# OpenAI: Enter API key (stored in env var or config)
```

### CLI Commands

```bash
# Index/reindex custom commands
please index

# List all indexed commands
please list-commands

# Reconfigure custom commands
please configure
```

### Best Practices

1. **Add 10-15 examples per command** - More examples = better matching
2. **Use good keywords** - Include synonyms, abbreviations, related terms
3. **Add aliases** - Common alternative names for your tool
4. **Set priority** - `priority: high` for frequently used tools
5. **Keep docs updated** - Reindex after changes with `please index`

### Example: kubectl

See `~/.please/commands/kubectl.md` (created during setup) for a comprehensive example with 15+ kubectl patterns.

### Troubleshooting

**"No custom commands found"**:
```bash
# Ensure commands directory exists and has .md files
ls ~/.please/commands/

# Reindex
please index
```

**"Failed to connect to Ollama"**:
```bash
# Start Ollama service
ollama serve

# Or check if already running
ps aux | grep ollama
```

**"Commands not matching"**:
- Add more keywords to frontmatter
- Add more examples to your .md files
- Try hybrid or semantic strategy
- Lower scoreThreshold in config

## Command History

All requests and executions are logged to `~/.please/history.json`:

```json
{
  "entries": [
    {
      "timestamp": "2025-01-15T10:30:00Z",
      "original_request": "find large files",
      "final_command": "find . -type f -size +100M",
      "executed": true,
      "modifications": []
    }
  ]
}
```

## How it works

1. **Input** - You describe what you want in natural language
2. **Translation** - Claude AI translates your request to a shell command
3. **Review** - The command is displayed for your approval
4. **Refinement** (optional) - You can request modifications in natural language
5. **Execution** - The command runs in your current shell context
6. **History** - The session is logged for future reference

## Tips

- Be specific about what you want to accomplish
- You can chain operations in your request
- The tool understands context about your OS and shell
- If a command isn't safe, consider whether you really want to run it
- Check the generated command carefully before executing

## Safety

`please` always shows you the command before execution and asks for confirmation. However:

- ⚠️ **Review commands carefully** before running
- 🔍 **Understand what the command does** before confirming
- 🛡️ **Be cautious with destructive operations** (rm, dd, etc.)
- 💾 **Backup important data** before running unfamiliar commands

## Development

### Project structure

```
please/
├── cmd/please/          # Main entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── agent/           # LLM agent implementations
│   ├── customcmd/       # Custom commands RAG system
│   │   ├── embeddings/  # Embedding providers (Ollama, OpenAI)
│   │   └── vectorstore/ # Vector storage and similarity search
│   ├── history/         # Command history
│   ├── ui/              # Interactive prompts
│   └── executor/        # Command execution
└── README.md
```

### Building

```bash
go build -o please ./cmd/please
```

### Testing

```bash
go test ./...
```

## Roadmap

### Completed ✅
- [x] Custom commands with RAG (keyword + semantic search)
- [x] Ollama integration (local embeddings)
- [x] OpenAI integration (cloud embeddings)
- [x] Auto-indexing with staleness detection
- [x] Markdown-based command documentation

### In Progress 🚧
- [ ] Comprehensive test suite
- [ ] Persistent vector store (save index to disk)

### Planned 📋
- [ ] Support for additional LLM providers (Codex, Goose, etc.)
- [ ] Command suggestions based on history
- [ ] Alias creation for frequently used commands
- [ ] Shell completion
- [ ] Multi-step command workflows
- [ ] Dry-run mode
- [ ] Command explanation mode
- [ ] Additional embedding providers (Cohere, local models)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Powered by [Claude](https://www.anthropic.com/claude) from Anthropic
- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Uses [Survey](https://github.com/AlecAivazis/survey) for interactive prompts

---

**Note**: This tool executes shell commands on your system. Always review generated commands before running them. Use at your own risk.

# please

A natural language interface for your terminal. Simply describe what you want to do in plain English, and `please` will translate it into the right command and execute it.

## Features

- 🗣️ **Natural language commands** - No need to remember complex syntax
- 🤖 **Powered by Claude** - Uses Claude AI for accurate command translation
- ✅ **Interactive confirmation** - Review commands before execution
- ✏️ **Iterative refinement** - Modify commands with natural language
- 📝 **Command history** - Tracks all your requests and executions
- 🔒 **Safe by default** - Always asks for confirmation before running

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
  "agent": "claude-code"
}
```

Authentication is handled by the Claude CLI, so no API keys are stored in the config.

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

- [ ] Support for additional LLM providers (Codex, Goose, etc.)
- [ ] Command suggestions based on history
- [ ] Alias creation for frequently used commands
- [ ] Shell completion
- [ ] Multi-step command workflows
- [ ] Dry-run mode
- [ ] Command explanation mode

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

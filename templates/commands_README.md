# Custom Commands for `please`

This directory contains custom command documentation that teaches `please` about proprietary or internal tools.

## How It Works

When you use `please`, it searches these files for relevant commands based on:
- **Command names** and **aliases**
- **Keywords** that match your request
- **Examples** of similar requests
- **Categories** and priorities

## File Format

Each `.md` file should have YAML frontmatter followed by markdown documentation:

```markdown
---
command: your-command-name
aliases: [alias1, alias2]
keywords: [keyword1, keyword2, keyword3]
categories: [category1, category2]
priority: high
version: "1.0"
---

# Command Name - Brief Description

## Overview
Brief explanation of what this command does.

## Common Patterns

### Pattern Category 1
- Description: `command --flag value`
- Another usage: `command subcommand`

### Pattern Category 2
- More examples

## Examples

User: "natural language request"
Command: the actual command to run

User: "another request"
Command: another command example

## Tips
- Helpful tips and best practices
- Common flags and options
```

## Frontmatter Fields

- **command** (required): Primary command name
- **aliases**: Alternative names (e.g., `k8s` for `kubectl`)
- **keywords**: Words that might appear in requests (e.g., `pods`, `deploy`, `logs`)
- **categories**: Logical groupings (e.g., `devops`, `database`, `ci-cd`)
- **priority**: `high`, `medium`, or `low` (affects matching)
- **version**: Tool version this doc is for

## Examples Section

The examples section is **very important** for matching. Format:

```markdown
User: "show me production logs"
Command: kubectl logs -f deployment/app -n production
```

Or:

```markdown
Request: "deploy to staging"
Command: deploy-tool --env staging --branch main
```

## Tips for Good Documentation

1. **Be specific with keywords**: Include technical terms users might say
2. **Add many examples**: More examples = better matching
3. **Use natural language**: Write requests as users would say them
4. **Include variations**: Same command with different flags/options
5. **Document common tasks**: Focus on what users do most often

## After Adding/Editing

Run `please index` to re-index your commands, or it will auto-index on next use.

## Example Files

See the included example files:
- `kubectl.md` - Kubernetes CLI
- Add more files for your internal tools!

## Need Help?

- Check example files in this directory
- Run `please list-commands` to see indexed commands
- See the full documentation: [Custom Commands Guide](https://github.com/iishyfishyy/please)

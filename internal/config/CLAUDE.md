# internal/config - Configuration Management

## Overview

The `config` package manages persistent configuration for `please`, including agent selection and future settings. Configuration is stored in `~/.please/config.json` in a simple, human-readable format.

## Architecture

### Configuration File Structure

**Location**: `~/.please/config.json`

**Current Schema**:
```json
{
  "agent": "claude-code"
}
```

**Future Schema** (examples):
```json
{
  "agent": "claude-code",
  "openai_key": "sk-...",           // For OpenAI agent
  "goose_path": "/usr/local/bin/goose",
  "preferences": {
    "auto_execute": false,           // Skip confirmation (dangerous!)
    "show_explanations": true,       // Add command explanations
    "history_limit": 1000            // Max history entries
  },
  "safety": {
    "blocked_patterns": [            // Prevent dangerous commands
      "rm -rf /",
      "dd if=.*of=/dev/.*"
    ],
    "require_confirmation": [        // Extra confirmation for these
      "rm -rf",
      "mkfs",
      "fdisk"
    ]
  }
}
```

### Code Structure

**config.go**:
```go
const (
    ConfigDirName  = ".please"
    ConfigFileName = "config.json"
)

type AgentType string
const (
    AgentClaude AgentType = "claude-code"
    // Future agents
)

type Config struct {
    Agent AgentType `json:"agent"`
    // Future fields
}

func GetConfigDir() (string, error)        // ~/.please
func GetConfigPath() (string, error)       // ~/.please/config.json
func Load() (*Config, error)               // Read config
func Save(cfg *Config) error               // Write config
func Exists() (bool, error)                // Check if configured
```

## Configuration Lifecycle

### 1. First Run (Not Configured)

```
User runs: please "list files"
        ↓
Load() returns nil (no config file)
        ↓
Main shows: "No configuration found. Please run 'please configure' first."
        ↓
Exit
```

### 2. Configuration Wizard

```
User runs: please configure
        ↓
Check Claude CLI installed
        ↓
UI prompt: Select agent (currently only Claude)
        ↓
Validate agent (test connection)
        ↓
Create ~/.please/ directory
        ↓
Save config to ~/.please/config.json
        ↓
Show success message
```

### 3. Normal Use

```
User runs: please "find files"
        ↓
Load() reads ~/.please/config.json
        ↓
Parse JSON → Config struct
        ↓
Return config to main
        ↓
Create agent based on config.Agent
        ↓
Execute workflow
```

## Adding New Configuration Options

### Step-by-Step Guide

#### 1. Update Config Struct

```go
// config.go
type Config struct {
    Agent      AgentType `json:"agent"`

    // New field - use pointer for optional values
    AutoExecute *bool `json:"auto_execute,omitempty"`

    // Or with default value
    HistoryLimit int `json:"history_limit,omitempty"`
}
```

**Naming Conventions**:
- Use snake_case in JSON tags (matches JSON conventions)
- Use PascalCase in Go struct (matches Go conventions)
- Use `omitempty` for optional fields
- Use pointers for truly optional booleans (distinguish false from unset)

#### 2. Add Validation (Optional)

```go
// config.go
func (c *Config) Validate() error {
    if c.Agent == "" {
        return fmt.Errorf("agent is required")
    }

    if c.HistoryLimit < 0 {
        return fmt.Errorf("history_limit must be non-negative")
    }

    // Validate new field
    if c.HistoryLimit > 10000 {
        return fmt.Errorf("history_limit too large (max 10000)")
    }

    return nil
}

// Update Load() to validate
func Load() (*Config, error) {
    // ... existing load code ...

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return &cfg, nil
}
```

#### 3. Update Configuration Wizard

```go
// cmd/please/main.go - runConfigure()

func runConfigure(cmd *cobra.Command, args []string) error {
    // ... existing code ...

    // Add new configuration prompt
    autoExecute, err := ui.PromptAutoExecute()  // New UI function
    if err != nil {
        return err
    }

    cfg := &config.Config{
        Agent:       config.AgentClaude,
        AutoExecute: &autoExecute,  // Add to config
    }

    // ... save config ...
}
```

#### 4. Add UI Prompt (if needed)

```go
// internal/ui/prompt.go

func PromptAutoExecute() (bool, error) {
    var autoExecute bool
    prompt := &survey.Confirm{
        Message: "Enable auto-execute? (skips confirmation - not recommended)",
        Default: false,
    }

    if err := survey.AskOne(prompt, &autoExecute); err != nil {
        return false, err
    }

    if autoExecute {
        ShowWarning("⚠️  Auto-execute is dangerous! Commands will run without confirmation.")
    }

    return autoExecute, nil
}
```

#### 5. Use Configuration

```go
// cmd/please/main.go - runCommand()

func runCommand(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    // ... error handling ...

    // Use new config option
    if cfg.AutoExecute != nil && *cfg.AutoExecute {
        // Skip confirmation, execute directly
        if err := executor.Execute(currentCommand); err != nil {
            return err
        }
    } else {
        // Normal flow with confirmation
        action, err := ui.ConfirmCommand(currentCommand)
        // ... rest of flow ...
    }
}
```

## Migration Strategy

### Handling Schema Changes

When adding new fields, maintain backward compatibility:

#### Default Values Approach

```go
type Config struct {
    Agent        AgentType `json:"agent"`
    HistoryLimit int       `json:"history_limit,omitempty"`  // New field
}

func Load() (*Config, error) {
    // ... load JSON ...

    // Apply defaults for missing fields
    if cfg.HistoryLimit == 0 {
        cfg.HistoryLimit = 1000  // Default value
    }

    return &cfg, nil
}
```

#### Version-Based Migration

For breaking changes:

```go
type Config struct {
    Version      int       `json:"version,omitempty"`
    Agent        AgentType `json:"agent"`
    // ... other fields ...
}

const CurrentConfigVersion = 2

func Load() (*Config, error) {
    // ... load JSON ...

    // Migrate if needed
    if cfg.Version < CurrentConfigVersion {
        if err := migrate(cfg); err != nil {
            return nil, fmt.Errorf("config migration failed: %w", err)
        }
        cfg.Version = CurrentConfigVersion
        // Auto-save migrated config
        if err := Save(cfg); err != nil {
            // Log warning but don't fail
            fmt.Fprintf(os.Stderr, "Warning: failed to save migrated config: %v\n", err)
        }
    }

    return &cfg, nil
}

func migrate(cfg *Config) error {
    switch cfg.Version {
    case 0:
        // Migrate v0 → v1
        if cfg.OldField != "" {
            cfg.NewField = convertOldToNew(cfg.OldField)
        }
        fallthrough  // Apply subsequent migrations
    case 1:
        // Migrate v1 → v2
        // ...
    }
    return nil
}
```

### Breaking Changes

If a breaking change is absolutely necessary:

1. **Increment Version**: `CurrentConfigVersion++`
2. **Detect Old Version**: Check `cfg.Version`
3. **Prompt User**: Explain what changed
4. **Offer Migration**: Auto-migrate or prompt for new config
5. **Document**: Update README and CHANGELOG

Example:
```go
func Load() (*Config, error) {
    // ... load JSON ...

    if cfg.Version == 0 && cfg.Agent == "old-agent" {
        ui.ShowWarning("Config format has changed. Please run 'please configure' again.")
        return nil, fmt.Errorf("outdated config, reconfiguration required")
    }

    return &cfg, nil
}
```

## Configuration Best Practices

### 1. Sensible Defaults

```go
// Good: Safe defaults
type Config struct {
    Agent          AgentType `json:"agent"`
    AutoExecute    bool      `json:"auto_execute,omitempty"`     // Defaults to false
    HistoryLimit   int       `json:"history_limit,omitempty"`    // Default applied in Load()
}

func Load() (*Config, error) {
    // ... load ...

    if cfg.HistoryLimit == 0 {
        cfg.HistoryLimit = 1000  // Reasonable default
    }

    return &cfg, nil
}
```

### 2. Validation

```go
// Validate on load, not just save
func Load() (*Config, error) {
    // ... load ...

    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    return &cfg, nil
}

// Comprehensive validation
func (c *Config) Validate() error {
    // Required fields
    if c.Agent == "" {
        return fmt.Errorf("agent is required")
    }

    // Enum validation
    validAgents := []AgentType{AgentClaude, AgentOpenAI}
    if !contains(validAgents, c.Agent) {
        return fmt.Errorf("invalid agent: %s", c.Agent)
    }

    // Range validation
    if c.HistoryLimit < 0 || c.HistoryLimit > 10000 {
        return fmt.Errorf("history_limit must be between 0 and 10000")
    }

    return nil
}
```

### 3. Error Handling

```go
// Distinguish error types
func Load() (*Config, error) {
    configPath, err := GetConfigPath()
    if err != nil {
        return nil, err  // System error
    }

    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return nil, nil  // Not an error - not configured yet
    }

    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("invalid config format: %w", err)
    }

    return &cfg, nil
}
```

### 4. Security

```go
// Secure file permissions
func Save(cfg *Config) error {
    // ...

    // Config may contain sensitive data (API keys)
    if err := os.WriteFile(configPath, data, 0600); err != nil {  // User-only
        return fmt.Errorf("failed to write config file: %w", err)
    }

    return nil
}

// Sensitive data warnings
type Config struct {
    Agent      AgentType `json:"agent"`
    OpenAIKey  string    `json:"openai_key,omitempty"`  // Sensitive!
}

// Don't log sensitive fields
func (c *Config) String() string {
    return fmt.Sprintf("Config{Agent: %s}", c.Agent)  // Omit OpenAIKey
}
```

### 5. Documentation

```go
// Document struct fields
type Config struct {
    // Agent specifies which LLM provider to use
    // Supported values: "claude-code", "openai"
    Agent AgentType `json:"agent"`

    // AutoExecute skips confirmation and runs commands immediately
    // WARNING: This is dangerous! Only enable if you fully trust the LLM.
    AutoExecute bool `json:"auto_execute,omitempty"`

    // HistoryLimit is the maximum number of history entries to keep
    // Default: 1000, Range: 0-10000
    HistoryLimit int `json:"history_limit,omitempty"`
}
```

## Testing Strategy

### Test Cases

```go
// config_test.go

func TestLoad_NotExists(t *testing.T) {
    // Test: Load when config doesn't exist
    // Should return (nil, nil)
}

func TestLoad_Valid(t *testing.T) {
    // Test: Load valid config
    // Should parse correctly
}

func TestLoad_Invalid(t *testing.T) {
    // Test: Load malformed JSON
    // Should return error
}

func TestSave_CreateDirectory(t *testing.T) {
    // Test: Save when directory doesn't exist
    // Should create directory and file
}

func TestValidate_MissingRequired(t *testing.T) {
    // Test: Validate with missing required fields
    // Should return error
}

func TestValidate_InvalidRange(t *testing.T) {
    // Test: Validate with out-of-range values
    // Should return error
}

func TestMigration_V0toV1(t *testing.T) {
    // Test: Migrate from old schema to new
    // Should update fields correctly
}
```

### Mock Configuration

```go
// For testing other packages
func TestWithConfig(t *testing.T) {
    // Create temporary config
    tmpDir, err := os.MkdirTemp("", "please-test")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir)

    // Override config location
    os.Setenv("HOME", tmpDir)
    defer os.Unsetenv("HOME")

    // Create test config
    cfg := &Config{Agent: AgentClaude}
    if err := Save(cfg); err != nil {
        t.Fatal(err)
    }

    // Run tests
    // ...
}
```

## Common Patterns

### 1. Config Singleton (Not Recommended)

```go
// Don't do this - makes testing hard
var globalConfig *Config

func GetGlobalConfig() *Config {
    if globalConfig == nil {
        globalConfig, _ = Load()
    }
    return globalConfig
}
```

**Better**: Pass config as parameter

```go
// Good - explicit dependencies
func runCommand(cfg *Config, args []string) error {
    // Use cfg
}
```

### 2. Environment Variable Overrides

```go
// Useful for testing and advanced users
func Load() (*Config, error) {
    cfg, err := loadFromFile()
    if err != nil {
        return nil, err
    }

    // Override with environment variables
    if agentEnv := os.Getenv("PLEASE_AGENT"); agentEnv != "" {
        cfg.Agent = AgentType(agentEnv)
    }

    return cfg, nil
}
```

### 3. Partial Updates

```go
// Update single field without loading entire config
func UpdateAgent(agent AgentType) error {
    cfg, err := Load()
    if err != nil {
        return err
    }

    cfg.Agent = agent

    if err := cfg.Validate(); err != nil {
        return err
    }

    return Save(cfg)
}
```

## Future Enhancements

### Short-term
- [ ] Add validation framework
- [ ] Support environment variable overrides
- [ ] Add `please config list` command to view config

### Medium-term
- [ ] Per-project config (`.please.json` in project root)
- [ ] Config inheritance (project overrides global)
- [ ] Encrypted storage for API keys
- [ ] Config versioning and migration

### Long-term
- [ ] Remote config sync
- [ ] Config templates for teams
- [ ] GUI config editor

## Quick Reference

### Adding a Boolean Option
```go
// 1. Add to struct
AutoExecute *bool `json:"auto_execute,omitempty"`

// 2. Add UI prompt in configure
autoExecute := ui.PromptBool("Enable auto-execute?", false)
cfg.AutoExecute = &autoExecute

// 3. Use in main
if cfg.AutoExecute != nil && *cfg.AutoExecute { ... }
```

### Adding an API Key
```go
// 1. Add to struct (sensitive!)
OpenAIKey string `json:"openai_key,omitempty"`

// 2. Secure save (0600 permissions)
os.WriteFile(configPath, data, 0600)

// 3. Validate
if c.OpenAIKey != "" && !strings.HasPrefix(c.OpenAIKey, "sk-") {
    return fmt.Errorf("invalid OpenAI key format")
}
```

### Migrating Config Schema
```go
// 1. Add version field
Version int `json:"version,omitempty"`

// 2. Implement migration
func migrate(cfg *Config) error {
    // Transform old → new
}

// 3. Call in Load()
if cfg.Version < CurrentConfigVersion {
    migrate(cfg)
    Save(cfg)
}
```

## Related Documentation

- **Root CLAUDE.md**: Overall architecture
- **internal/agent/CLAUDE.md**: Agent configuration options
- **README.md**: User-facing configuration guide

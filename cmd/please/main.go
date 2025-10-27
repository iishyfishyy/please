package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
	"github.com/iishyfishyy/please/internal/agent"
	"github.com/iishyfishyy/please/internal/config"
	"github.com/iishyfishyy/please/internal/customcmd"
	"github.com/iishyfishyy/please/internal/executor"
	"github.com/iishyfishyy/please/internal/history"
	"github.com/iishyfishyy/please/internal/ui"

	"github.com/spf13/cobra"
)

var (
	// version is set by goreleaser at build time
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// CLI flags
	forceReindex bool
	debug        bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "please [command description]",
		Short:   "Natural language interface for your terminal",
		Long:    "please translates natural language into shell commands using AI",
		Version: version,
		Args:    cobra.MinimumNArgs(1),
		RunE:    runCommand,
	}

	// Add global debug flag
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")

	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure please with your preferred LLM agent",
		RunE:  runConfigure,
	}

	indexCmd := &cobra.Command{
		Use:   "index",
		Short: "Index custom command documentation",
		RunE:  runIndex,
	}
	indexCmd.Flags().BoolVarP(&forceReindex, "force", "f", false, "Force reindexing (bypass cache)")

	listCommandsCmd := &cobra.Command{
		Use:   "list-commands",
		Short: "List indexed custom commands",
		RunE:  runListCommands,
	}

	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(listCommandsCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ConfigStatus represents the current state of the configuration
type ConfigStatus struct {
	HasConfig              bool
	AgentConfigured        bool
	AgentWorking           bool
	AgentType              string
	CustomCommandsEnabled  bool
	CustomCommandsProvider string
	CommandsIndexedCount   int
	LastIndexTime          time.Time
	ConfigPath             string
}

// analyzeCurrentConfig examines the current configuration state
func analyzeCurrentConfig() (*ConfigStatus, *config.Config, error) {
	status := &ConfigStatus{}

	// Get config path
	configPath, _ := config.GetConfigPath()
	status.ConfigPath = configPath

	// Load existing config (if exists)
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg == nil {
		// No config exists yet
		status.HasConfig = false
		return status, nil, nil
	}

	status.HasConfig = true

	// Check agent status
	if cfg.Agent != "" {
		status.AgentConfigured = true
		status.AgentType = string(cfg.Agent)

		// Check if CLI is installed (fast check, no API call)
		status.AgentWorking = agent.IsClaudeCLIInstalled()
	}

	// Check custom commands status
	if cfg.CustomCommands != nil && cfg.CustomCommands.Enabled {
		status.CustomCommandsEnabled = true
		status.CustomCommandsProvider = string(cfg.CustomCommands.Provider)

		// Check if commands are indexed
		manager, err := customcmd.NewManager()
		if err == nil {
			// Load commands from disk to get accurate count
			if err := manager.Load(); err == nil {
				if manager.IsIndexed() {
					status.CommandsIndexedCount = manager.Count()
					status.LastIndexTime = manager.GetIndexTime()
				}
			}
		}
	}

	return status, cfg, nil
}

// runInitialSetup performs first-time setup
func runInitialSetup() error {
	ui.ShowInfo("No configuration found. Let's set up please.\n")

	// Check if Claude CLI is installed
	if !agent.IsClaudeCLIInstalled() {
		ui.ShowError("Claude CLI not found!")
		ui.ShowInfo("\nTo use 'please', you need to install and authenticate with Claude CLI.")
		ui.ShowInfo("Installation instructions: https://github.com/anthropics/claude-cli")
		ui.ShowInfo("\nAfter installing, run 'claude auth' to authenticate, then run 'please configure' again.")
		return nil
	}

	ui.ShowInfo("Setting up Claude CLI...")

	cfg := &config.Config{
		Agent: config.AgentClaude,
	}

	// Verify Claude CLI is working
	ui.ShowInfo("Verifying Claude CLI authentication...")
	testAgent := agent.NewClaudeAgent()
	ctx := context.Background()
	_, err := testAgent.TranslateToCommand(ctx, "echo hello")
	if err != nil {
		ui.ShowError("Failed to communicate with Claude CLI")
		ui.ShowInfo("Please run 'claude auth' to authenticate and try again.")
		return nil
	}
	ui.ShowSuccess("Claude CLI is working!")

	// Custom commands setup
	fmt.Println()
	if err := configureCustomCommandsInitial(cfg); err != nil {
		ui.ShowWarning(fmt.Sprintf("Custom commands setup skipped: %v", err))
		// Continue anyway - custom commands are optional
	}

	// Save configuration
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	configPath, _ := config.GetConfigPath()
	ui.ShowSuccess(fmt.Sprintf("Configuration saved to %s", configPath))
	ui.ShowInfo("\nYou're all set! Try running: please \"list all files\"")

	return nil
}

// displayConfigStatus shows a summary of current configuration
func displayConfigStatus(status *ConfigStatus) {
	fmt.Println()
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	gray := color.New(color.FgHiBlack)

	// Agent status
	fmt.Print("  Agent: ")
	if status.AgentConfigured {
		if status.AgentWorking {
			green.Printf("%s ✓ (installed)\n", status.AgentType)
		} else {
			red.Printf("%s ✗ (not installed)\n", status.AgentType)
		}
	} else {
		gray.Println("Not configured")
	}

	// Custom commands status
	fmt.Print("  Custom Commands: ")
	if status.CustomCommandsEnabled {
		if status.CommandsIndexedCount > 0 {
			green.Printf("Enabled (%s, %d commands)\n", status.CustomCommandsProvider, status.CommandsIndexedCount)
		} else {
			green.Printf("Enabled (%s, not indexed)\n", status.CustomCommandsProvider)
		}
	} else {
		gray.Println("Disabled")
	}

	fmt.Println()
}

// configureAgentMenu shows agent configuration options
func configureAgentMenu(cfg *config.Config) error {
	ui.ShowSection("Agent Setup")

	// Check current status
	fmt.Println()
	ui.ShowInfo("Checking Claude CLI authentication...")
	agentWorking := false
	if agent.IsClaudeCLIInstalled() {
		testAgent := agent.NewClaudeAgent()
		ctx := context.Background()
		_, err := testAgent.TranslateToCommand(ctx, "echo test")
		agentWorking = (err == nil)
	}

	// Show current status
	fmt.Println()
	if agentWorking {
		ui.ShowSuccess("Current: Claude CLI ✓ (authenticated)")
	} else {
		ui.ShowWarning("Current: Claude CLI (authentication issue)")
	}
	fmt.Println()

	// Show options
	options := []string{
		"Re-verify Claude CLI authentication",
		"Back to main menu",
	}

	selected, err := ui.ShowMenu("Actions:", options)
	if err != nil {
		return err
	}

	switch selected {
	case 0: // Re-verify
		ui.ShowInfo("Verifying Claude CLI authentication...")
		testAgent := agent.NewClaudeAgent()
		ctx := context.Background()
		_, err := testAgent.TranslateToCommand(ctx, "echo hello")
		if err != nil {
			ui.ShowError("Failed to communicate with Claude CLI")
			ui.ShowInfo("Please run 'claude auth' to authenticate and try again.")
		} else {
			ui.ShowSuccess("Claude CLI is working!")
		}
	case 1: // Back
		return nil
	}

	return nil
}

// viewCurrentConfiguration displays the full configuration
func viewCurrentConfiguration(status *ConfigStatus, cfg *config.Config) {
	ui.ShowSection("Current Configuration")

	fmt.Println()

	// Agent section
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("Agent: Claude CLI")
	if status.AgentWorking {
		fmt.Println("  Status: ✓ Installed")
	} else {
		fmt.Println("  Status: ✗ Not installed")
	}
	fmt.Println()

	// Custom Commands section
	cyan.Println("Custom Commands:")
	if status.CustomCommandsEnabled {
		fmt.Println("  Status: Enabled")
		fmt.Printf("  Provider: %s\n", status.CustomCommandsProvider)
		if status.CommandsIndexedCount > 0 {
			fmt.Printf("  Indexed: %d commands", status.CommandsIndexedCount)
			if !status.LastIndexTime.IsZero() {
				fmt.Printf(" (%s ago)\n", formatDuration(status.LastIndexTime))
			} else {
				fmt.Println()
			}
		} else {
			fmt.Println("  Indexed: No commands indexed yet")
		}

		commandsDir, _ := customcmd.GetCommandsDir()
		fmt.Printf("  Location: %s\n", commandsDir)
	} else {
		fmt.Println("  Status: Disabled")
	}
	fmt.Println()

	// Config file location
	fmt.Printf("Configuration file: %s\n", status.ConfigPath)
	fmt.Println()

	fmt.Println("Press Enter to return to menu...")
	fmt.Scanln()
}

// configureCustomCommandsMenu shows custom commands configuration options
func configureCustomCommandsMenu(cfg *config.Config) error {
	ui.ShowSection("Custom Commands Settings")

	// Check current status
	isEnabled := cfg.CustomCommands != nil && cfg.CustomCommands.Enabled
	provider := ""
	if isEnabled {
		provider = string(cfg.CustomCommands.Provider)
	}

	// Show current status
	fmt.Println()
	if isEnabled {
		ui.ShowSuccess(fmt.Sprintf("Current Status: Enabled (%s)", provider))

		// Show options for enabled state
		options := []string{
			"Change embedding provider",
			"Disable custom commands",
			"Re-index commands",
			"Back to main menu",
		}

		selected, err := ui.ShowMenu("Actions:", options)
		if err != nil {
			return err
		}

		switch selected {
		case 0: // Change provider
			return changeEmbeddingProvider(cfg)
		case 1: // Disable
			return disableCustomCommands(cfg)
		case 2: // Re-index
			return reindexCommands(cfg)
		case 3: // Back
			return nil
		}
	} else {
		ui.ShowInfo("Current Status: Disabled")

		// Show options for disabled state
		options := []string{
			"Enable custom commands",
			"Back to main menu",
		}

		selected, err := ui.ShowMenu("Actions:", options)
		if err != nil {
			return err
		}

		switch selected {
		case 0: // Enable
			return enableCustomCommands(cfg)
		case 1: // Back
			return nil
		}
	}

	return nil
}

// enableCustomCommands enables custom commands feature
func enableCustomCommands(cfg *config.Config) error {
	ui.ShowInfo("\nCustom commands allow you to teach 'please' about proprietary/internal tools")
	ui.ShowInfo("by adding documentation to ~/.please/commands/\n")

	// Ask for provider
	provider, err := ui.PromptProvider()
	if err != nil {
		return err
	}

	var customCfg *config.CustomCommands

	switch provider {
	case "ollama":
		if err := customcmd.SetupOllama(); err != nil {
			return err
		}
		customCfg = config.NewDefaultCustomCommands(config.ProviderOllama)

	case "openai":
		apiKey, useEnv, err := customcmd.SetupOpenAI()
		if err != nil {
			return err
		}
		customCfg = config.NewDefaultCustomCommands(config.ProviderOpenAI)
		if !useEnv {
			customCfg.OpenAI.APIKey = apiKey
		}
		customCfg.OpenAI.UseEnvVar = useEnv

	case "none":
		if err := customcmd.SetupKeywordOnly(); err != nil {
			return err
		}
		customCfg = config.NewDefaultCustomCommands(config.ProviderNone)

	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	cfg.CustomCommands = customCfg

	// Create commands directory with templates
	if err := customcmd.EnsureCommandsDirWithTemplates(); err != nil {
		return err
	}

	// Save configuration
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	commandsDir, _ := customcmd.GetCommandsDir()
	ui.ShowSuccess(fmt.Sprintf("Custom commands enabled! Directory: %s", commandsDir))
	fmt.Println()
	ui.ShowInfo("Next steps:")
	ui.ShowInfo(fmt.Sprintf("  1. Add custom command docs to %s", commandsDir))
	ui.ShowInfo("  2. Run: please index")

	return nil
}

// disableCustomCommands disables custom commands feature
func disableCustomCommands(cfg *config.Config) error {
	confirmed, err := ui.PromptYesNo("Are you sure you want to disable custom commands?", false)
	if err != nil {
		return err
	}

	if !confirmed {
		ui.ShowInfo("Cancelled")
		return nil
	}

	cfg.CustomCommands.Enabled = false

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	ui.ShowSuccess("Custom commands disabled")
	return nil
}

// changeEmbeddingProvider changes the embedding provider
func changeEmbeddingProvider(cfg *config.Config) error {
	ui.ShowInfo(fmt.Sprintf("\nCurrent provider: %s\n", cfg.CustomCommands.Provider))

	provider, err := ui.PromptProvider()
	if err != nil {
		return err
	}

	// Don't change if same provider
	if provider == string(cfg.CustomCommands.Provider) {
		ui.ShowInfo("Provider unchanged")
		return nil
	}

	var customCfg *config.CustomCommands

	switch provider {
	case "ollama":
		if err := customcmd.SetupOllama(); err != nil {
			return err
		}
		customCfg = config.NewDefaultCustomCommands(config.ProviderOllama)

	case "openai":
		apiKey, useEnv, err := customcmd.SetupOpenAI()
		if err != nil {
			return err
		}
		customCfg = config.NewDefaultCustomCommands(config.ProviderOpenAI)
		if !useEnv {
			customCfg.OpenAI.APIKey = apiKey
		}
		customCfg.OpenAI.UseEnvVar = useEnv

	case "none":
		if err := customcmd.SetupKeywordOnly(); err != nil {
			return err
		}
		customCfg = config.NewDefaultCustomCommands(config.ProviderNone)

	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	cfg.CustomCommands = customCfg

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	ui.ShowSuccess(fmt.Sprintf("Provider changed to: %s", provider))
	ui.ShowInfo("Note: You may need to re-index commands for the new provider")

	return nil
}

// reindexCommands re-indexes custom command documentation
func reindexCommands(cfg *config.Config) error {
	ui.ShowInfo("Re-indexing custom commands...")

	manager, err := customcmd.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	// Configure embeddings if enabled
	if cfg.CustomCommands.Provider != config.ProviderNone {
		provider := string(cfg.CustomCommands.Provider)
		model := ""
		dims := 0

		if cfg.CustomCommands.Provider == config.ProviderOllama {
			model = cfg.CustomCommands.Ollama.Model
			dims = 384
		} else if cfg.CustomCommands.Provider == config.ProviderOpenAI {
			model = "text-embedding-3-small"
			dims = 1536
		}

		manager.SetEmbeddingConfig(provider, model, dims)
	}

	ctx := context.Background()
	if err := manager.Index(ctx, true); err != nil { // force=true
		return fmt.Errorf("failed to index commands: %w", err)
	}

	ui.ShowSuccess(fmt.Sprintf("Re-indexed %d commands", manager.Count()))
	return nil
}

// configureCustomCommandsInitial is the initial setup flow for custom commands
func configureCustomCommandsInitial(cfg *config.Config) error {
	ui.ShowSection("Custom Commands Setup")

	ui.ShowInfo("Custom commands allow you to teach 'please' about proprietary/internal tools")
	ui.ShowInfo("by adding documentation to ~/.please/commands/")
	fmt.Println()

	enabled, err := ui.PromptYesNo("Enable custom command support?", true)
	if err != nil {
		return err
	}

	if !enabled {
		ui.ShowInfo("Custom commands disabled")
		return nil
	}

	return enableCustomCommands(cfg)
}

func runConfigure(cmd *cobra.Command, args []string) error {
	ui.ShowSection("Please Configuration")

	// Analyze current configuration
	status, cfg, err := analyzeCurrentConfig()
	if err != nil {
		return fmt.Errorf("failed to analyze configuration: %w", err)
	}

	// If no configuration exists, run initial setup
	if !status.HasConfig {
		return runInitialSetup()
	}

	// Configuration exists - show menu
	for {
		// Display current status
		displayConfigStatus(status)

		// Show menu options
		options := []string{
			"Agent Setup (Claude CLI)",
			"Custom Commands Settings",
			"View Current Configuration",
			"Exit",
		}

		selected, err := ui.ShowMenu("What would you like to configure?", options)
		if err != nil {
			return err
		}

		switch selected {
		case 0: // Agent Setup
			if err := configureAgentMenu(cfg); err != nil {
				ui.ShowError(fmt.Sprintf("Agent configuration failed: %v", err))
			}
		case 1: // Custom Commands
			if err := configureCustomCommandsMenu(cfg); err != nil {
				ui.ShowError(fmt.Sprintf("Custom commands configuration failed: %v", err))
			}
		case 2: // View Configuration
			viewCurrentConfiguration(status, cfg)
		case 3: // Exit
			ui.ShowInfo("Configuration menu closed")
			return nil
		}

		// Re-analyze after changes
		status, cfg, err = analyzeCurrentConfig()
		if err != nil {
			return fmt.Errorf("failed to reload configuration: %w", err)
		}
	}
}

func runCommand(cmd *cobra.Command, args []string) error {
	// Combine all args into a single request first (for debug logging)
	request := strings.Join(args, " ")

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Main: starting with request: %q\n", request)
	}

	// Load configuration
	if debug {
		configPath, _ := config.GetConfigPath()
		fmt.Fprintf(os.Stderr, "[DEBUG] Config: loading from %s\n", configPath)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if cfg == nil {
		ui.ShowError("No configuration found. Please run 'please configure' first.")
		return nil
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Config: loaded successfully (agent=%s, custom_commands=%v)\n",
			cfg.Agent, cfg.CustomCommands != nil && cfg.CustomCommands.Enabled)
	}

	// Check if Claude CLI is installed
	if !agent.IsClaudeCLIInstalled() {
		ui.ShowError("Claude CLI not found!")
		ui.ShowInfo("Please install and authenticate with Claude CLI, then run 'please configure'")
		return nil
	}

	// Create agent
	var ag agent.Agent
	var claudeAg *agent.ClaudeAgent
	switch cfg.Agent {
	case config.AgentClaude:
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Agent: creating Claude agent\n")
		}
		claudeAg = agent.NewClaudeAgent()
		claudeAg.SetDebug(debug)
		ag = claudeAg
	default:
		return fmt.Errorf("unknown agent type: %s", cfg.Agent)
	}

	// Setup custom commands if enabled
	if cfg.CustomCommands != nil && cfg.CustomCommands.Enabled && claudeAg != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: setting up (provider=%s, strategy=%s)\n",
				cfg.CustomCommands.Provider, cfg.CustomCommands.Matching.Strategy)
		}
		cmdManager, err := setupCustomCommands(cfg)
		if err != nil {
			ui.ShowWarning(fmt.Sprintf("Custom commands setup failed: %v", err))
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: setup failed: %v\n", err)
			}
		} else if cmdManager != nil {
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: manager created with %d commands\n", cmdManager.Count())
			}
			// Set up the custom doc getter function
			claudeAg.SetCustomDocGetter(func(request string, maxDocs int) []agent.CustomCommandDoc {
				docs := cmdManager.GetRelevantDocsForAgent(request, maxDocs)
				if debug {
					fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: matched %d docs for request %q\n", len(docs), request)
					for _, doc := range docs {
						fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd:   - %s (%d examples)\n", doc.Command, len(doc.Examples))
					}
				}
				// Convert to agent types
				agentDocs := make([]agent.CustomCommandDoc, len(docs))
				for i, doc := range docs {
					agentDocs[i] = agent.CustomCommandDoc{
						Command:  doc.Command,
						Content:  doc.Content,
						Examples: make([]agent.CommandExample, len(doc.Examples)),
					}
					for j, ex := range doc.Examples {
						agentDocs[i].Examples[j] = agent.CommandExample{
							UserRequest: ex.UserRequest,
							Command:     ex.Command,
						}
					}
				}
				return agentDocs
			})
		}
	}

	// Load history
	if debug {
		histPath, _ := history.GetHistoryPath()
		fmt.Fprintf(os.Stderr, "[DEBUG] History: loading from %s\n", histPath)
	}
	hist, err := history.Load()
	if err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Track modifications for history
	modifications := []string{}

	ctx := context.Background()

	// Translate request to command
	ui.ShowInfo("Thinking...")
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Agent: translating request to command: %q\n", request)
	}
	currentCommand, err := ag.TranslateToCommand(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to translate command: %w", err)
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Agent: generated command: %q\n", currentCommand)
	}

	// Interactive loop for modification
	for {
		// Show command and get user action
		action, err := ui.ConfirmCommand(currentCommand)
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		switch action {
		case ui.ActionRun:
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] User: chose to run command\n")
			}
			// Execute the command
			if err := executor.ExecuteWithDebug(currentCommand, debug); err != nil {
				ui.ShowError(fmt.Sprintf("Command failed: %v", err))
			}

			// Save to history
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] History: saving entry (executed=true, modifications=%d)\n", len(modifications))
			}
			entry := history.NewEntry(request, currentCommand, true, modifications)
			hist.AddEntry(entry)
			if err := hist.Save(); err != nil {
				// Log error but don't fail
				fmt.Fprintf(os.Stderr, "Warning: failed to save history: %v\n", err)
			}

			return nil

		case ui.ActionExplain:
			// Get explanation from agent (pass original request for custom command context)
			ui.ShowInfo("Explaining...")
			explanation, err := ag.ExplainCommand(ctx, currentCommand, request)
			if err != nil {
				ui.ShowError(fmt.Sprintf("Failed to get explanation: %v", err))
			} else {
				// Format markdown for terminal display
				formattedExplanation := ui.FormatMarkdown(explanation)
				fmt.Println("\n" + formattedExplanation + "\n")
			}

			// Loop continues to show the command again

		case ui.ActionCopy:
			// Copy to clipboard
			if err := clipboard.WriteAll(currentCommand); err != nil {
				ui.ShowError(fmt.Sprintf("Failed to copy to clipboard: %v", err))
			} else {
				ui.ShowSuccess("Command copied to clipboard!")
			}

			// Loop continues to show the command again

		case ui.ActionCancel:
			ui.ShowInfo("Cancelled.")

			// Save to history (not executed)
			entry := history.NewEntry(request, currentCommand, false, modifications)
			hist.AddEntry(entry)
			if err := hist.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save history: %v\n", err)
			}

			return nil

		case ui.ActionModify:
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] User: chose to modify command\n")
			}
			// Get modification request
			modRequest, err := ui.PromptForModification()
			if err != nil {
				return fmt.Errorf("failed to get modification: %w", err)
			}

			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] User: modification request: %q\n", modRequest)
			}

			modifications = append(modifications, modRequest)

			// Refine command
			ui.ShowInfo("Refining...")
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Agent: refining command with modification: %q\n", modRequest)
			}
			currentCommand, err = ag.RefineCommand(ctx, currentCommand, modRequest)
			if err != nil {
				return fmt.Errorf("failed to refine command: %w", err)
			}
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Agent: refined command: %q\n", currentCommand)
			}

			// Loop continues to show the new command
		}
	}
}

// setupCustomCommands creates and initializes the custom command manager
func setupCustomCommands(cfg *config.Config) (*customcmd.Manager, error) {
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: creating manager\n")
	}

	manager, err := customcmd.NewManagerWithDebug(debug)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Check if there are any custom commands
	hasCommands, err := customcmd.HasCommands()
	if err != nil {
		return nil, fmt.Errorf("failed to check for commands: %w", err)
	}

	if !hasCommands {
		if debug {
			commandsDir, _ := customcmd.GetCommandsDir()
			fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: no commands found in %s\n", commandsDir)
		}
		// No commands yet, return empty manager
		return manager, nil
	}

	// Load/index commands (auto-index on first run or if stale)
	if !manager.IsIndexed() || manager.NeedsReindex() {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: needs indexing (indexed=%v, stale=%v)\n",
				manager.IsIndexed(), manager.NeedsReindex())
		}
		ui.ShowInfo("Indexing custom commands...")
		if err := manager.Load(); err != nil {
			return nil, fmt.Errorf("failed to index commands: %w", err)
		}
		ui.ShowSuccess(fmt.Sprintf("Indexed %d custom commands", manager.Count()))
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: indexed %d commands successfully\n", manager.Count())
		}
	} else if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] CustomCmd: using cached index (%d commands)\n", manager.Count())
	}

	return manager, nil
}

// runIndex indexes custom command documentation
func runIndex(cmd *cobra.Command, args []string) error {
	ui.ShowSection("Indexing Custom Commands")

	// Load config to check if custom commands are enabled
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if cfg == nil {
		ui.ShowError("No configuration found. Please run 'please configure' first.")
		return nil
	}

	if cfg.CustomCommands == nil || !cfg.CustomCommands.Enabled {
		ui.ShowError("Custom commands are not enabled")
		ui.ShowInfo("Run 'please configure' to enable custom commands")
		return nil
	}

	// Check if there are any command files
	hasCommands, err := customcmd.HasCommands()
	if err != nil {
		return fmt.Errorf("failed to check for commands: %w", err)
	}

	if !hasCommands {
		commandsDir, _ := customcmd.GetCommandsDir()
		ui.ShowWarning("No custom command files found")
		ui.ShowInfo(fmt.Sprintf("Add .md files to: %s", commandsDir))
		ui.ShowInfo("See README.md in that directory for format")
		return nil
	}

	// Create manager and index
	manager, err := customcmd.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	// Configure embeddings if enabled
	if cfg.CustomCommands.Provider != config.ProviderNone {
		provider := string(cfg.CustomCommands.Provider)
		model := ""
		dims := 0

		if cfg.CustomCommands.Provider == config.ProviderOllama {
			model = cfg.CustomCommands.Ollama.Model
			dims = 384 // nomic-embed-text
		} else if cfg.CustomCommands.Provider == config.ProviderOpenAI {
			model = "text-embedding-3-small"
			dims = 1536
		}

		manager.SetEmbeddingConfig(provider, model, dims)
	}

	if forceReindex {
		ui.ShowInfo("Force reindexing (--force flag)")
	}

	ui.ShowInfo("Indexing...")
	ctx := context.Background()
	if err := manager.Index(ctx, forceReindex); err != nil {
		return fmt.Errorf("failed to index commands: %w", err)
	}

	ui.ShowSuccess(fmt.Sprintf("Indexed %d custom commands", manager.Count()))

	// Show summary
	docs := manager.GetDocs()
	fmt.Println()
	ui.ShowInfo("Commands indexed:")
	for _, doc := range docs {
		fmt.Printf("  • %s (%d examples, %d keywords)\n",
			doc.Command, len(doc.Examples), len(doc.Keywords))
	}

	return nil
}

// runListCommands lists indexed custom commands
func runListCommands(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if cfg == nil {
		ui.ShowError("No configuration found. Please run 'please configure' first.")
		return nil
	}

	if cfg.CustomCommands == nil || !cfg.CustomCommands.Enabled {
		ui.ShowError("Custom commands are not enabled")
		ui.ShowInfo("Run 'please configure' to enable custom commands")
		return nil
	}

	// Create manager and load
	manager, err := customcmd.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	// Check if indexed
	hasCommands, err := customcmd.HasCommands()
	if err != nil {
		return fmt.Errorf("failed to check for commands: %w", err)
	}

	if !hasCommands {
		commandsDir, _ := customcmd.GetCommandsDir()
		ui.ShowWarning("No custom command files found")
		ui.ShowInfo(fmt.Sprintf("Add .md files to: %s", commandsDir))
		return nil
	}

	// Load commands
	if err := manager.Load(); err != nil {
		return fmt.Errorf("failed to load commands: %w", err)
	}

	docs := manager.GetDocs()
	if len(docs) == 0 {
		ui.ShowWarning("No commands indexed")
		ui.ShowInfo("Run 'please index' to index your commands")
		return nil
	}

	// Display commands
	ui.ShowSection("Custom Commands")
	fmt.Printf("Indexed %s ago\n\n", formatDuration(manager.GetIndexTime()))

	for _, doc := range docs {
		fmt.Printf("📦 %s\n", doc.Command)
		if len(doc.Aliases) > 0 {
			fmt.Printf("   Aliases: %s\n", strings.Join(doc.Aliases, ", "))
		}
		if len(doc.Keywords) > 0 {
			keywords := doc.Keywords
			if len(keywords) > 10 {
				keywords = keywords[:10]
			}
			fmt.Printf("   Keywords: %s\n", strings.Join(keywords, ", "))
		}
		fmt.Printf("   Examples: %d\n", len(doc.Examples))
		if !doc.UpdatedAt.IsZero() {
			fmt.Printf("   Updated: %s\n", doc.UpdatedAt.Format("2006-01-02"))
		}
		fmt.Println()
	}

	// Show provider info
	providerName := "keyword matching"
	switch cfg.CustomCommands.Provider {
	case config.ProviderOllama:
		providerName = fmt.Sprintf("Ollama (%s)", cfg.CustomCommands.Ollama.Model)
	case config.ProviderOpenAI:
		providerName = fmt.Sprintf("OpenAI (%s)", cfg.CustomCommands.OpenAI.Model)
	}

	fmt.Printf("Provider: %s\n", providerName)
	fmt.Printf("Strategy: %s\n", cfg.CustomCommands.Matching.Strategy)

	commandsDir, _ := customcmd.GetCommandsDir()
	fmt.Printf("\nCommands directory: %s\n", commandsDir)
	ui.ShowInfo("Run 'please index' to re-index after changes")

	return nil
}

// formatDuration formats a time.Time as "X ago"
func formatDuration(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
}

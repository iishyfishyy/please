package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"golang.org/x/term"
)

// Action represents the user's choice
type Action int

const (
	ActionRun Action = iota
	ActionExplain
	ActionModify
	ActionCopy
	ActionCancel
)

// ConfigureAgent prompts the user to select an agent
func ConfigureAgent() (string, error) {
	var agent string
	prompt := &survey.Select{
		Message: "Select an LLM agent:",
		Options: []string{"Claude Code"},
		Default: "Claude Code",
	}

	if err := survey.AskOne(prompt, &agent); err != nil {
		return "", err
	}

	return agent, nil
}

// ConfirmCommand shows the command and asks the user what to do
func ConfirmCommand(command string) (Action, error) {
	// Display the command with nice formatting
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("\nGenerated command:")
	fmt.Printf("  %s\n\n", command)

	// Display options with keyboard shortcuts
	fmt.Println("What would you like to do?")
	fmt.Println("  [r] Run it")
	fmt.Println("  [e] Explain")
	fmt.Println("  [m] Modify it")
	fmt.Println("  [c] Copy to clipboard")
	fmt.Println("  [q] Cancel")
	fmt.Print("\nPress a key: ")

	// Read a single keypress
	key, err := readKey()
	if err != nil {
		return ActionCancel, err
	}

	// Clear the line
	fmt.Println()

	// Map key to action
	switch key {
	case 'r', 'R':
		return ActionRun, nil
	case 'e', 'E':
		return ActionExplain, nil
	case 'm', 'M':
		return ActionModify, nil
	case 'c', 'C':
		return ActionCopy, nil
	case 'q', 'Q', '\x1b': // ESC key is \x1b
		return ActionCancel, nil
	default:
		// Invalid key, ask again
		ShowError("Invalid choice. Please try again.")
		return ConfirmCommand(command)
	}
}

// readKey reads a single keypress from the terminal
func readKey() (rune, error) {
	// Save the current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0, err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Read a single byte
	buf := make([]byte, 1)
	_, err = os.Stdin.Read(buf)
	if err != nil {
		return 0, err
	}

	return rune(buf[0]), nil
}

// PromptForModification asks the user how to modify the command
func PromptForModification() (string, error) {
	var modification string
	prompt := &survey.Input{
		Message: "How would you like to modify the command?",
	}

	if err := survey.AskOne(prompt, &modification, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return modification, nil
}

// ShowSuccess displays a success message
func ShowSuccess(message string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Printf("✓ %s\n", message)
}

// ShowError displays an error message
func ShowError(message string) {
	red := color.New(color.FgRed, color.Bold)
	red.Printf("✗ %s\n", message)
}

// ShowInfo displays an info message
func ShowInfo(message string) {
	blue := color.New(color.FgBlue)
	blue.Println(message)
}

// ShowWarning displays a warning message
func ShowWarning(message string) {
	yellow := color.New(color.FgYellow, color.Bold)
	yellow.Printf("⚠ %s\n", message)
}

// ShowSection displays a section header
func ShowSection(title string) {
	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println(title)
	fmt.Println(strings.Repeat("━", len(title)))
}

// PromptYesNo asks a yes/no question
func PromptYesNo(message string, defaultValue bool) (bool, error) {
	var result bool
	prompt := &survey.Confirm{
		Message: message,
		Default: defaultValue,
	}

	if err := survey.AskOne(prompt, &result); err != nil {
		return false, err
	}

	return result, nil
}

// PromptProvider prompts for embedding provider selection
func PromptProvider() (string, error) {
	var provider string
	prompt := &survey.Select{
		Message: "Choose embedding provider:",
		Options: []string{
			"Local (Ollama) - Private, runs on your machine",
			"OpenAI API - Cloud-based, most accurate",
			"None - Keyword matching only (faster, less accurate)",
		},
		Default: "Local (Ollama) - Private, runs on your machine",
	}

	if err := survey.AskOne(prompt, &provider); err != nil {
		return "", err
	}

	// Parse selection to provider name
	if contains(provider, "Ollama") {
		return "ollama", nil
	} else if contains(provider, "OpenAI") {
		return "openai", nil
	}
	return "none", nil
}

// PromptAPIKeyStorage prompts for how to store OpenAI API key
func PromptAPIKeyStorage() (bool, error) {
	var choice string
	prompt := &survey.Select{
		Message: "OpenAI API key storage:",
		Options: []string{
			"Environment variable (recommended)",
			"Config file (less secure)",
		},
		Default: "Environment variable (recommended)",
	}

	if err := survey.AskOne(prompt, &choice); err != nil {
		return false, err
	}

	return contains(choice, "Environment"), nil
}

// PromptPassword prompts for a password/API key (hidden input)
func PromptPassword(message string) (string, error) {
	var password string
	prompt := &survey.Password{
		Message: message,
	}

	if err := survey.AskOne(prompt, &password, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return password, nil
}

// PromptInput prompts for text input
func PromptInput(message string, defaultValue string) (string, error) {
	var input string
	prompt := &survey.Input{
		Message: message,
		Default: defaultValue,
	}

	if err := survey.AskOne(prompt, &input); err != nil {
		return "", err
	}

	return input, nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ShowMenu displays an interactive menu and returns the selected index
func ShowMenu(title string, options []string) (int, error) {
	var selected string
	prompt := &survey.Select{
		Message: title,
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return -1, err
	}

	// Find the index of the selected option
	for i, opt := range options {
		if opt == selected {
			return i, nil
		}
	}

	return -1, fmt.Errorf("selected option not found")
}

// ShowConfigStatus displays a summary of the current configuration
func ShowConfigStatus(status interface{}) {
	// This will be called with a ConfigStatus from main.go
	// We use interface{} to avoid circular dependency
	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("Current Settings:")
	cyan.Println("─────────────────")
}

// FormatMarkdown converts markdown text to terminal-friendly format
func FormatMarkdown(text string) string {
	var result strings.Builder
	lines := strings.Split(text, "\n")

	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)
	inCodeBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle code blocks
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue // Skip the ``` markers
		}

		if inCodeBlock {
			// Code blocks: indent slightly with gray color
			gray := color.New(color.FgHiBlack)
			result.WriteString("  ")
			result.WriteString(gray.Sprint(line))
			result.WriteString("\n")
			continue
		}

		// Handle headers (## Header or ### Header)
		if strings.HasPrefix(trimmed, "### ") {
			// H3: Yellow, less prominent
			headerText := strings.TrimPrefix(trimmed, "### ")
			result.WriteString("\n")
			result.WriteString(yellow.Sprint(headerText))
			result.WriteString("\n")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			// H2: Cyan bold with underline
			headerText := strings.TrimPrefix(trimmed, "## ")
			result.WriteString("\n")
			result.WriteString(cyan.Sprint(headerText))
			result.WriteString("\n")
			result.WriteString(cyan.Sprint(strings.Repeat("─", len(headerText))))
			result.WriteString("\n")
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			// H1: Cyan bold with double underline
			headerText := strings.TrimPrefix(trimmed, "# ")
			result.WriteString("\n")
			result.WriteString(cyan.Sprint(headerText))
			result.WriteString("\n")
			result.WriteString(cyan.Sprint(strings.Repeat("═", len(headerText))))
			result.WriteString("\n")
			continue
		}

		// Handle bold text (**text** or __text__)
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")

		// Handle italic (just remove markers, terminal doesn't support well)
		line = strings.ReplaceAll(line, "*", "")
		line = strings.ReplaceAll(line, "_", "")

		// Handle inline code (`code`)
		// Keep backticks for now, they're readable in terminal

		// Handle bullet points - keep them but ensure proper spacing
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			result.WriteString("  • ")
			result.WriteString(strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* "))
			result.WriteString("\n")
			continue
		}

		// Handle numbered lists
		if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' && trimmed[1] == '.' {
			result.WriteString("  ")
			result.WriteString(trimmed)
			result.WriteString("\n")
			continue
		}

		// Regular paragraphs
		if trimmed != "" {
			result.WriteString(line)
			result.WriteString("\n")
		} else if i < len(lines)-1 {
			// Preserve empty lines for spacing (but not trailing)
			result.WriteString("\n")
		}
	}

	return result.String()
}

package ui

import (
	"fmt"
	"os"

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
	fmt.Println("  [r/1] Run it")
	fmt.Println("  [e/2] Explain")
	fmt.Println("  [m/3] Modify it")
	fmt.Println("  [c/4] Copy to clipboard")
	fmt.Println("  [q/5] Cancel")
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
	case 'r', 'R', '1':
		return ActionRun, nil
	case 'e', 'E', '2':
		return ActionExplain, nil
	case 'm', 'M', '3':
		return ActionModify, nil
	case 'c', 'C', '4':
		return ActionCopy, nil
	case 'q', 'Q', '5', '\x1b': // ESC key is \x1b
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

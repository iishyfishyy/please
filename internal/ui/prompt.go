package ui

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

// Action represents the user's choice
type Action int

const (
	ActionRun Action = iota
	ActionCancel
	ActionModify
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

	var choice string
	prompt := &survey.Select{
		Message: "What would you like to do?",
		Options: []string{
			"Run it",
			"Modify it",
			"Cancel",
		},
	}

	if err := survey.AskOne(prompt, &choice); err != nil {
		return ActionCancel, err
	}

	switch choice {
	case "Run it":
		return ActionRun, nil
	case "Modify it":
		return ActionModify, nil
	default:
		return ActionCancel, nil
	}
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

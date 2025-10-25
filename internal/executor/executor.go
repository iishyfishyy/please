package executor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Execute runs a shell command and returns the output
func Execute(command string) error {
	var cmd *exec.Cmd

	// Determine shell based on OS
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd = exec.Command(shell, "-c", command)
	}

	// Set up command to use current stdin/stdout/stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

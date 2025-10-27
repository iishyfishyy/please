package executor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Execute runs a shell command and returns the output
func Execute(command string) error {
	return ExecuteWithDebug(command, false)
}

// ExecuteWithDebug runs a shell command with optional debug logging
func ExecuteWithDebug(command string, debug bool) error {
	var cmd *exec.Cmd
	var shell string
	var shellArgs []string

	// Determine shell based on OS
	if runtime.GOOS == "windows" {
		shell = "cmd"
		shellArgs = []string{"/C", command}
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Executor: using Windows cmd.exe\n")
		}
	} else {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		shellArgs = []string{"-c", command}
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Executor: using shell %s\n", shell)
		}
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Executor: executing command: %q\n", command)
	}

	cmd = exec.Command(shell, shellArgs...)

	// Set up command to use current stdin/stdout/stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		if debug {
			// Check if it's an exit error with a code
			if exitError, ok := err.(*exec.ExitError); ok {
				fmt.Fprintf(os.Stderr, "[DEBUG] Executor: command failed with exit code %d\n", exitError.ExitCode())
			} else {
				fmt.Fprintf(os.Stderr, "[DEBUG] Executor: command failed: %v\n", err)
			}
		}
		return fmt.Errorf("command failed: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Executor: command completed successfully\n")
	}

	return nil
}

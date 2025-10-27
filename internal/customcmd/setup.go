package customcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/iishyfishyy/please/internal/ui"
)

// SetupOllama handles the complete Ollama setup flow
func SetupOllama() error {
	ui.ShowSection("Ollama Setup")

	// Check if Ollama is installed
	if !IsOllamaInstalled() {
		ui.ShowInfo("Ollama not found")

		install, err := ui.PromptYesNo("Install Ollama now?", true)
		if err != nil {
			return err
		}

		if !install {
			return fmt.Errorf("ollama installation declined - required for local embeddings")
		}

		ui.ShowInfo("Installing Ollama...")
		if err := InstallOllama(); err != nil {
			return fmt.Errorf("failed to install Ollama: %w", err)
		}

		ui.ShowSuccess("Ollama installed successfully")
	} else {
		ui.ShowSuccess("Ollama is already installed")
	}

	// Check if Ollama is running
	if !IsOllamaRunning() {
		ui.ShowInfo("Starting Ollama service...")
		if err := StartOllama(); err != nil {
			return fmt.Errorf("failed to start Ollama: %w", err)
		}
		// Give it a moment to start
		time.Sleep(2 * time.Second)
	}

	// Pull the embedding model
	model := "nomic-embed-text"
	ui.ShowInfo(fmt.Sprintf("Downloading embedding model (%s, ~275MB)...", model))
	ui.ShowInfo("This may take a few minutes...")

	if err := PullOllamaModel(model); err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}

	ui.ShowSuccess("Model downloaded successfully")

	// Test Ollama
	ui.ShowInfo("Testing Ollama...")
	if err := TestOllama(model); err != nil {
		return fmt.Errorf("ollama test failed: %w", err)
	}

	ui.ShowSuccess("Ollama setup complete!")
	return nil
}

// SetupOpenAI handles OpenAI setup flow
func SetupOpenAI() (string, bool, error) {
	ui.ShowSection("OpenAI Setup")

	// Ask how to provide API key
	useEnv, err := ui.PromptAPIKeyStorage()
	if err != nil {
		return "", false, err
	}

	var apiKey string

	if useEnv {
		// Check if environment variable exists
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			ui.ShowWarning("OPENAI_API_KEY environment variable not set")
			ui.ShowInfo("")
			ui.ShowInfo("Please set it in your shell:")
			ui.ShowInfo("  export OPENAI_API_KEY=sk-...")
			ui.ShowInfo("")

			return "", true, fmt.Errorf("OPENAI_API_KEY not set")
		}
	} else {
		// Prompt for API key
		apiKey, err = ui.PromptPassword("Enter OpenAI API key:")
		if err != nil {
			return "", false, err
		}

		ui.ShowWarning("API key will be saved to ~/.please/config.json (0600 perms)")
	}

	// Test the API key
	ui.ShowInfo("Testing OpenAI connection...")
	if err := TestOpenAI(apiKey); err != nil {
		return "", useEnv, fmt.Errorf("openAI test failed: %w", err)
	}

	ui.ShowSuccess("OpenAI configured successfully!")
	ui.ShowInfo("Model: text-embedding-3-small")
	ui.ShowInfo("Cost: ~$0.02 per 1M tokens (very low for typical use)")

	return apiKey, useEnv, nil
}

// SetupKeywordOnly handles keyword-only setup
func SetupKeywordOnly() error {
	ui.ShowSection("Keyword Matching")

	ui.ShowInfo("Using keyword matching only")
	ui.ShowInfo("  - Faster (< 1ms)")
	ui.ShowInfo("  - No external dependencies")
	ui.ShowInfo("  - Works well for 5-20 commands")
	ui.ShowInfo("  - Can upgrade to embeddings later")

	return nil
}

// IsOllamaInstalled checks if Ollama is installed
func IsOllamaInstalled() bool {
	_, err := exec.LookPath("ollama")
	return err == nil
}

// IsOllamaRunning checks if Ollama service is running
func IsOllamaRunning() bool {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// InstallOllama installs Ollama based on the operating system
func InstallOllama() error {
	switch runtime.GOOS {
	case "darwin":
		// macOS - use Homebrew
		return exec.Command("brew", "install", "ollama").Run()

	case "linux":
		// Linux - use install script
		script := "curl -fsSL https://ollama.com/install.sh | sh"
		cmd := exec.Command("sh", "-c", script)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	default:
		return fmt.Errorf("automatic installation not supported on %s - please install manually from https://ollama.com", runtime.GOOS)
	}
}

// StartOllama starts the Ollama service
func StartOllama() error {
	switch runtime.GOOS {
	case "darwin":
		// macOS - start as background service
		return exec.Command("brew", "services", "start", "ollama").Run()

	case "linux":
		// Linux - start with systemd or directly
		if err := exec.Command("systemctl", "start", "ollama").Run(); err != nil {
			// Fallback: run in background
			cmd := exec.Command("ollama", "serve")
			if err := cmd.Start(); err != nil {
				return err
			}
			return nil
		}
		return nil

	default:
		return fmt.Errorf("automatic start not supported on %s", runtime.GOOS)
	}
}

// PullOllamaModel pulls an embedding model
func PullOllamaModel(model string) error {
	cmd := exec.Command("ollama", "pull", model)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// TestOllama tests if Ollama is working by generating a test embedding
func TestOllama(model string) error {
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": "test",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		"http://localhost:11434/api/embeddings",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return fmt.Errorf("empty embedding returned")
	}

	return nil
}

// TestOpenAI tests if the OpenAI API key is valid
func TestOpenAI(apiKey string) error {
	reqBody := map[string]interface{}{
		"model": "text-embedding-3-small",
		"input": "test",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		"https://api.openai.com/v1/embeddings",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	return nil
}

// EnsureCommandsDirWithTemplates creates commands directory and copies templates
func EnsureCommandsDirWithTemplates() error {
	// Create commands directory
	if err := EnsureCommandsDir(); err != nil {
		return err
	}

	commandsDir, err := GetCommandsDir()
	if err != nil {
		return err
	}

	// Copy README template
	readmePath := commandsDir + "/README.md"
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		if err := copyTemplateFile("commands_README.md", readmePath); err != nil {
			// Non-fatal, just warn
			ui.ShowWarning(fmt.Sprintf("Could not create README: %v", err))
		}
	}

	// Copy kubectl example
	kubectlPath := commandsDir + "/kubectl.md"
	if _, err := os.Stat(kubectlPath); os.IsNotExist(err) {
		if err := copyTemplateFile("kubectl_example.md", kubectlPath); err != nil {
			ui.ShowWarning(fmt.Sprintf("Could not create kubectl example: %v", err))
		} else {
			ui.ShowInfo(fmt.Sprintf("Created example: %s", kubectlPath))
		}
	}

	return nil
}

// copyTemplateFile copies a template file to destination
func copyTemplateFile(templateName, destPath string) error {
	// Try to find template file
	// First check if we're in development (templates/ directory exists)
	templatePath := "templates/" + templateName
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		// Try embedded or installed location
		// For now, we'll embed the content directly
		return createTemplateContent(templateName, destPath)
	}

	// Copy file
	input, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}

	return os.WriteFile(destPath, input, 0644)
}

// createTemplateContent creates template files with embedded content
func createTemplateContent(templateName, destPath string) error {
	var content string

	switch templateName {
	case "commands_README.md":
		content = `# Custom Commands for please

This directory contains custom command documentation that teaches ` + "`please`" + ` about your tools.

## Quick Start

1. Create a .md file for each command (e.g., ` + "`deploy-tool.md`" + `)
2. Add YAML frontmatter with command metadata
3. Add examples showing how to use the command
4. Run ` + "`please index`" + ` to index your commands
5. Use ` + "`please \"your natural language request\"`" + `

## File Format

` + "```" + `yaml
---
command: tool-name          # Required: primary command name
aliases: ["alt1", "alt2"]   # Optional: alternative names
keywords: ["key1", "key2"]  # Optional: search keywords
categories: ["cat1"]        # Optional: categories
priority: high              # Optional: high/medium/low
---

# Tool Description

Brief description of what this tool does.

## Examples

**User**: "natural language request"
**Command**: ` + "`" + `actual command to run` + "`" + `

**User**: "another request"
**Command**: ` + "`" + `another command` + "`" + `
` + "```" + `

## Tips

- Add 10-15 diverse examples per command
- Use good keywords (synonyms, abbreviations)
- Add aliases for common alternative names
- Set priority=high for frequently used tools
- Examples should cover common use cases

## Examples

See ` + "`kubectl.md`" + ` for a comprehensive example with 15+ patterns.

Create your own files like:
- ` + "`docker.md`" + ` - Docker commands
- ` + "`deploy-tool.md`" + ` - Your internal deployment tool
- ` + "`custom-script.md`" + ` - Your custom scripts

Run ` + "`please list-commands`" + ` to see all indexed commands.
`
	case "kubectl_example.md":
		content = `---
command: kubectl
aliases: ["k8s", "kube"]
keywords: ["kubernetes", "pods", "deployments", "services", "logs", "namespace", "containers"]
categories: ["devops", "kubernetes", "containers"]
priority: high
version: "1.28"
---

# kubectl - Kubernetes Command Line Tool

kubectl is the Kubernetes CLI for managing clusters and workloads.

## Common Patterns

- List resources: ` + "`kubectl get {type}`" + `
- Describe resource: ` + "`kubectl describe {type} {name}`" + `
- View logs: ` + "`kubectl logs {pod}`" + `
- Execute commands: ` + "`kubectl exec {pod} -- {command}`" + `
- Apply manifests: ` + "`kubectl apply -f {file}`" + `

## Examples

**User**: "show me all pods"
**Command**: ` + "`kubectl get pods -A`" + `

**User**: "list pods in production namespace"
**Command**: ` + "`kubectl get pods -n production`" + `

**User**: "get logs from nginx pod"
**Command**: ` + "`kubectl logs nginx`" + `

**User**: "describe the api deployment"
**Command**: ` + "`kubectl describe deployment api`" + `

**User**: "show all services"
**Command**: ` + "`kubectl get services -A`" + `

**User**: "list deployments in staging"
**Command**: ` + "`kubectl get deployments -n staging`" + `

**User**: "get pod logs for the last hour"
**Command**: ` + "`kubectl logs --since=1h`" + `

**User**: "show nodes"
**Command**: ` + "`kubectl get nodes`" + `

**User**: "exec into nginx pod"
**Command**: ` + "`kubectl exec -it nginx -- /bin/bash`" + `

**User**: "apply manifest from app.yaml"
**Command**: ` + "`kubectl apply -f app.yaml`" + `

**User**: "delete pod nginx"
**Command**: ` + "`kubectl delete pod nginx`" + `

**User**: "show resource usage"
**Command**: ` + "`kubectl top pods`" + `

**User**: "port forward to service on 8080"
**Command**: ` + "`kubectl port-forward service/api 8080:80`" + `

**User**: "scale deployment to 3 replicas"
**Command**: ` + "`kubectl scale deployment api --replicas=3`" + `

**User**: "check cluster info"
**Command**: ` + "`kubectl cluster-info`" + `
`
	default:
		return fmt.Errorf("unknown template: %s", templateName)
	}

	return os.WriteFile(destPath, []byte(content), 0644)
}

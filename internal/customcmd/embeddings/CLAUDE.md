# internal/customcmd/embeddings - Embedding Providers

## Overview

This package provides a unified interface for generating text embeddings from different providers. Embeddings are vector representations of text that enable semantic search and similarity matching.

**Supported Providers**:
- **Ollama**: Local embeddings via Ollama HTTP API (privacy-focused)
- **OpenAI**: Cloud embeddings via OpenAI API (accuracy-focused)

**Key Concepts**:
- **Embedding**: A dense vector representation of text (e.g., `[0.1, -0.3, 0.5, ...]`)
- **Dimensions**: The size of the embedding vector (e.g., 384 for Ollama, 1536 for OpenAI)
- **Semantic Similarity**: Similar meanings → similar vectors (measured by cosine similarity)

## Architecture

```
User Request: "deploy to staging"
         ↓
    Embedder.Embed()
         ↓
    ┌─────────────────┐
    │   Provider      │
    ├─────────────────┤
    │ Ollama (local)  │ → [0.12, -0.34, 0.56, ...] (384 dims)
    │ OpenAI (API)    │ → [0.08, -0.21, 0.43, ...] (1536 dims)
    └─────────────────┘
         ↓
    Vector ([]float32)
         ↓
    Stored in VectorStore
         ↓
    Used for semantic search
```

## Interface

All embedding providers implement this interface:

```go
type Embedder interface {
    // Embed generates an embedding for a single text
    Embed(ctx context.Context, text string) ([]float32, error)

    // EmbedBatch generates embeddings for multiple texts
    // More efficient than calling Embed multiple times
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

    // Dimensions returns the embedding dimension size
    Dimensions() int

    // Name returns a human-readable name for the embedder
    // Format: "{provider}/{model}"
    Name() string
}
```

**Design Principles**:
- **Context-aware**: All methods accept `context.Context` for cancellation/timeout
- **Error handling**: Return descriptive errors with provider context
- **Batching**: Support batch operations for efficiency
- **Immutable**: Embedders are read-only after creation

## Ollama Provider

### Overview

Ollama provides local embedding generation via HTTP API. This is privacy-focused and free, but requires Ollama to be installed and running.

**Specifications**:
- Model: `nomic-embed-text` (default, recommended)
- Dimensions: 384
- API Endpoint: `http://localhost:11434/api/embeddings`
- Latency: ~50-100ms per embedding
- Cost: Free (runs locally)
- Privacy: 100% local, no data sent externally

### Creating an Embedder

```go
import "github.com/iishyfishyy/please/internal/customcmd/embeddings"

// Default URL and model
embedder, err := embeddings.NewOllamaEmbedder("http://localhost:11434", "nomic-embed-text")
if err != nil {
    return fmt.Errorf("failed to create Ollama embedder: %w", err)
}

// Custom model (e.g., llama2 embeddings)
embedder, err := embeddings.NewOllamaEmbedder("http://localhost:11434", "llama2")
```

### Embedding Text

```go
ctx := context.Background()

// Single text
embedding, err := embedder.Embed(ctx, "deploy to staging")
if err != nil {
    return fmt.Errorf("failed to embed: %w", err)
}
// Returns []float32 with 384 elements

// Batch (more efficient)
texts := []string{
    "deploy to staging",
    "list all pods",
    "check service status",
}
embeddings, err := embedder.EmbedBatch(ctx, texts)
if err != nil {
    return fmt.Errorf("failed to embed batch: %w", err)
}
// Returns [][]float32, len(embeddings) == len(texts)
```

### API Request Format

The Ollama API expects this format:

```json
POST http://localhost:11434/api/embeddings
Content-Type: application/json

{
  "model": "nomic-embed-text",
  "prompt": "deploy to staging"
}
```

Response:
```json
{
  "embedding": [0.123, -0.456, 0.789, ...]
}
```

### Implementation Details

```go
func (o *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    // 1. Build request
    reqBody := map[string]interface{}{
        "model":  o.model,
        "prompt": text,
    }
    jsonData, _ := json.Marshal(reqBody)

    // 2. Create HTTP request
    url := fmt.Sprintf("%s/api/embeddings", o.baseURL)
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
    req.Header.Set("Content-Type", "application/json")

    // 3. Send request
    resp, err := o.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("Ollama request failed: %w", err)
    }
    defer resp.Body.Close()

    // 4. Check status
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
    }

    // 5. Parse response
    var result struct {
        Embedding []float32 `json:"embedding"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return result.Embedding, nil
}
```

### Error Handling

Common errors and solutions:

```go
// Connection refused
// → Ollama not running. Start with: ollama serve
return fmt.Errorf("failed to connect to Ollama at %s: %w. Is Ollama running?", baseURL, err)

// Model not found
// → Model not downloaded. Pull with: ollama pull nomic-embed-text
return fmt.Errorf("model %s not found. Pull it with: ollama pull %s", model, model)

// Timeout
// → Increase timeout in HTTP client
o.client.Timeout = 60 * time.Second
```

### Batch Processing

Ollama doesn't have native batch support, so we loop:

```go
func (o *OllamaEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
    embeddings := make([][]float32, len(texts))

    for i, text := range texts {
        embedding, err := o.Embed(ctx, text)
        if err != nil {
            return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
        }
        embeddings[i] = embedding
    }

    return embeddings, nil
}
```

**Future Enhancement**: Parallelize with goroutines for faster batch processing.

## OpenAI Provider

### Overview

OpenAI provides cloud-based embedding generation via REST API. This is more accurate and supports batch operations natively, but requires an API key and sends data to OpenAI.

**Specifications**:
- Model: `text-embedding-3-small` (default) or `text-embedding-3-large`
- Dimensions: 1536 (small) or 3072 (large)
- API Endpoint: `https://api.openai.com/v1/embeddings`
- Latency: ~100-200ms per embedding
- Cost: $0.02 per 1M tokens (~$0.0001 per command index)
- Privacy: Data sent to OpenAI

### Creating an Embedder

```go
import (
    "os"
    "github.com/iishyfishyy/please/internal/customcmd/embeddings"
)

// Default model (text-embedding-3-small)
apiKey := os.Getenv("OPENAI_API_KEY")
embedder, err := embeddings.NewOpenAIEmbedder(apiKey, "")
if err != nil {
    return fmt.Errorf("failed to create OpenAI embedder: %w", err)
}

// Large model (more accurate, higher cost)
embedder, err := embeddings.NewOpenAIEmbedder(apiKey, "text-embedding-3-large")
```

### Embedding Text

```go
ctx := context.Background()

// Single text
embedding, err := embedder.Embed(ctx, "deploy to staging")
if err != nil {
    return fmt.Errorf("failed to embed: %w", err)
}
// Returns []float32 with 1536 elements (for text-embedding-3-small)

// Batch (native batch support)
texts := []string{
    "deploy to staging",
    "list all pods",
    "check service status",
}
embeddings, err := embedder.EmbedBatch(ctx, texts)
// Much more efficient than calling Embed 3 times
```

### API Request Format

The OpenAI API expects this format:

**Single Embedding**:
```json
POST https://api.openai.com/v1/embeddings
Authorization: Bearer sk-...
Content-Type: application/json

{
  "model": "text-embedding-3-small",
  "input": "deploy to staging"
}
```

**Batch Embedding**:
```json
{
  "model": "text-embedding-3-small",
  "input": [
    "deploy to staging",
    "list all pods",
    "check service status"
  ]
}
```

Response:
```json
{
  "data": [
    {
      "embedding": [0.123, -0.456, ...],
      "index": 0
    },
    {
      "embedding": [0.234, -0.567, ...],
      "index": 1
    }
  ]
}
```

### Implementation Details

```go
func (o *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    // 1. Build request
    reqBody := map[string]interface{}{
        "model": o.model,
        "input": text,
    }
    jsonData, _ := json.Marshal(reqBody)

    // 2. Create HTTP request
    req, _ := http.NewRequestWithContext(ctx, "POST",
        "https://api.openai.com/v1/embeddings",
        bytes.NewBuffer(jsonData))
    req.Header.Set("Authorization", "Bearer "+o.apiKey)
    req.Header.Set("Content-Type", "application/json")

    // 3. Send request
    resp, err := o.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("OpenAI request failed: %w", err)
    }
    defer resp.Body.Close()

    // 4. Check status
    if resp.StatusCode != 200 {
        var errResp struct {
            Error struct {
                Message string `json:"message"`
            } `json:"error"`
        }
        json.NewDecoder(resp.Body).Decode(&errResp)
        return nil, fmt.Errorf("OpenAI API error (status %d): %s",
            resp.StatusCode, errResp.Error.Message)
    }

    // 5. Parse response
    var result struct {
        Data []struct {
            Embedding []float32 `json:"embedding"`
        } `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return result.Data[0].Embedding, nil
}
```

### Error Handling

Common errors and solutions:

```go
// Invalid API key (status 401)
return fmt.Errorf("invalid OpenAI API key. Check OPENAI_API_KEY env var")

// Rate limit (status 429)
return fmt.Errorf("OpenAI rate limit exceeded. Wait and retry")

// Insufficient credits (status 402)
return fmt.Errorf("OpenAI account has insufficient credits")

// Network error
return fmt.Errorf("failed to connect to OpenAI: %w. Check internet connection", err)
```

### Batch Processing

OpenAI has native batch support:

```go
func (o *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
    // Send all texts in one request
    reqBody := map[string]interface{}{
        "model": o.model,
        "input": texts,  // Array of strings
    }

    // ... (send request) ...

    // Parse response with indices
    embeddings := make([][]float32, len(texts))
    for _, item := range result.Data {
        if item.Index < len(embeddings) {
            embeddings[item.Index] = item.Embedding
        }
    }

    return embeddings, nil
}
```

**Advantage**: Much more efficient than multiple requests (reduced latency and cost).

## Adding a New Provider

To add support for a new embedding provider (e.g., Cohere, HuggingFace, local models):

### Step 1: Create Provider File

Create `internal/customcmd/embeddings/{provider}.go`:

```go
package embeddings

import (
    "context"
    "fmt"
    "net/http"
    "time"
)

type CohereEmbedder struct {
    apiKey string
    model  string
    client *http.Client
    dims   int
}

func NewCohereEmbedder(apiKey, model string) (*CohereEmbedder, error) {
    if apiKey == "" {
        return nil, fmt.Errorf("Cohere API key is required")
    }

    if model == "" {
        model = "embed-english-v3.0" // Default model
    }

    return &CohereEmbedder{
        apiKey: apiKey,
        model:  model,
        client: &http.Client{Timeout: 30 * time.Second},
        dims:   1024, // Cohere embed-english-v3.0
    }, nil
}
```

### Step 2: Implement Embedder Interface

```go
func (c *CohereEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    // 1. Build request (Cohere-specific format)
    reqBody := map[string]interface{}{
        "texts":      []string{text},
        "model":      c.model,
        "input_type": "search_document",
    }

    // 2. Send POST to https://api.cohere.ai/v1/embed

    // 3. Parse response

    // 4. Return embedding
}

func (c *CohereEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
    // Cohere supports native batching
    reqBody := map[string]interface{}{
        "texts":      texts,
        "model":      c.model,
        "input_type": "search_document",
    }

    // ... (send and parse) ...
}

func (c *CohereEmbedder) Dimensions() int {
    return c.dims
}

func (c *CohereEmbedder) Name() string {
    return fmt.Sprintf("cohere/%s", c.model)
}
```

### Step 3: Update Config

Add provider to `internal/config/config.go`:

```go
type EmbeddingProvider string

const (
    ProviderNone   EmbeddingProvider = "none"
    ProviderOllama EmbeddingProvider = "ollama"
    ProviderOpenAI EmbeddingProvider = "openai"
    ProviderCohere EmbeddingProvider = "cohere"  // New provider
)
```

### Step 4: Add Setup Logic

Update `internal/customcmd/setup.go`:

```go
func SetupCohere() (string, error) {
    // 1. Prompt for API key
    apiKey := promptForAPIKey("Enter your Cohere API key:")

    // 2. Test connection
    embedder, err := embeddings.NewCohereEmbedder(apiKey, "")
    if err != nil {
        return "", err
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err = embedder.Embed(ctx, "test")
    if err != nil {
        return "", fmt.Errorf("connection test failed: %w", err)
    }

    return apiKey, nil
}
```

### Step 5: Wire into Configure Command

Update `cmd/please/main.go` to offer Cohere as an option in the configure wizard.

### Step 6: Test

```bash
# Build
go build -o please ./cmd/please

# Configure
please configure
# Select Cohere provider
# Enter API key
# Test indexing

# Index
please index

# Test search
please "your test query"
```

## Testing

### Unit Tests

Mock HTTP responses for each provider:

```go
func TestOllamaEmbedder_Embed(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.URL.Path != "/api/embeddings" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }

        // Return mock embedding
        json.NewEncoder(w).Encode(map[string]interface{}{
            "embedding": make([]float32, 384),
        })
    }))
    defer server.Close()

    // Create embedder with mock URL
    embedder, err := NewOllamaEmbedder(server.URL, "nomic-embed-text")
    if err != nil {
        t.Fatalf("failed to create embedder: %v", err)
    }

    // Test Embed
    embedding, err := embedder.Embed(context.Background(), "test")
    if err != nil {
        t.Fatalf("embed failed: %v", err)
    }

    if len(embedding) != 384 {
        t.Errorf("expected 384 dimensions, got %d", len(embedding))
    }
}
```

### Integration Tests

Test with real APIs (requires API keys):

```go
func TestOpenAIEmbedder_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    embedder, err := NewOpenAIEmbedder(apiKey, "")
    if err != nil {
        t.Fatalf("failed to create embedder: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Test single embed
    embedding, err := embedder.Embed(ctx, "test")
    if err != nil {
        t.Fatalf("embed failed: %v", err)
    }

    if len(embedding) != 1536 {
        t.Errorf("expected 1536 dimensions, got %d", len(embedding))
    }

    // Test batch
    embeddings, err := embedder.EmbedBatch(ctx, []string{"test1", "test2"})
    if err != nil {
        t.Fatalf("batch embed failed: %v", err)
    }

    if len(embeddings) != 2 {
        t.Errorf("expected 2 embeddings, got %d", len(embeddings))
    }
}
```

Run tests:
```bash
# Unit tests only
go test ./internal/customcmd/embeddings/

# Include integration tests
OPENAI_API_KEY=sk-... go test -v ./internal/customcmd/embeddings/
```

## Best Practices

### For Users

1. **Ollama for Privacy**: Use Ollama if you handle sensitive data
2. **OpenAI for Accuracy**: Use OpenAI for best semantic matching
3. **API Key Security**: Store OpenAI key in env var, not config file
4. **Model Selection**: Use default models unless you have specific needs

### For Developers

1. **Context Timeout**: Always use context with timeout for external APIs
2. **Error Messages**: Include provider name and actionable guidance
3. **Retry Logic**: Consider retries with exponential backoff for transient errors
4. **Batching**: Use native batch APIs when available
5. **Testing**: Mock HTTP responses, don't call real APIs in unit tests
6. **Caching**: Consider caching embeddings to reduce API calls (future enhancement)

## Performance

### Latency Comparison

| Provider | Single Embed | Batch (10 texts) |
|----------|-------------|------------------|
| Ollama   | ~50-100ms   | ~500-1000ms      |
| OpenAI   | ~100-200ms  | ~150-300ms       |

**Key Insight**: OpenAI's native batching is much more efficient than Ollama's sequential processing.

### Cost Comparison

| Provider | Cost per 1M tokens | Cost per 50 commands |
|----------|-------------------|---------------------|
| Ollama   | Free              | Free                |
| OpenAI   | $0.02             | ~$0.001             |

**Key Insight**: OpenAI is extremely cheap for this use case (~$0.001 to index 50 commands).

### Memory Usage

| Provider | Embedding Size | 50 Commands |
|----------|---------------|-------------|
| Ollama   | 1.5 KB        | 75 KB       |
| OpenAI   | 6 KB          | 300 KB      |

**Key Insight**: Ollama uses 4x less memory due to smaller embedding dimensions.

## Troubleshooting

### Ollama Issues

**"Connection refused"**:
```bash
# Start Ollama
ollama serve

# Or check if service is running
ps aux | grep ollama
```

**"Model not found"**:
```bash
# List models
ollama list

# Pull model
ollama pull nomic-embed-text
```

**"Slow embedding generation"**:
- Ollama runs on CPU by default
- Consider GPU acceleration for faster embeddings
- Or use smaller models

### OpenAI Issues

**"Invalid API key"**:
```bash
# Check env var
echo $OPENAI_API_KEY

# Or check config
cat ~/.please/config.json | grep apiKey
```

**"Rate limit exceeded"**:
- OpenAI has rate limits (requests per minute)
- Wait and retry, or implement exponential backoff
- Consider upgrading to higher tier

**"High latency"**:
- OpenAI latency depends on network and API load
- Use batch operations to amortize overhead
- Consider caching embeddings (future enhancement)

## References

- [Ollama API Documentation](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [OpenAI Embeddings Guide](https://platform.openai.com/docs/guides/embeddings)
- [Nomic Embed Text](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5)
- [Cosine Similarity](https://en.wikipedia.org/wiki/Cosine_similarity)
- [Text Embeddings Explained](https://www.anthropic.com/index/embeddings)

# internal/customcmd - Custom Commands Package

## Overview

This package implements a **RAG (Retrieval Augmented Generation)** system that allows users to teach `please` about custom, proprietary, or internal tools by providing documentation in markdown files. The system uses hybrid matching (keyword + semantic) to find relevant documentation and inject it into the LLM's context.

**Key Components**:
- **Manager**: Coordinates loading, matching, and indexing
- **Loader**: Scans and loads .md files from `~/.please/commands/`
- **Parser**: Extracts YAML frontmatter and user→command examples
- **Matcher**: Keyword-based scoring algorithm
- **Semantic**: Hybrid semantic search with embeddings
- **Setup**: Automated setup for Ollama and OpenAI
- **Embeddings**: Provider interface and implementations
- **VectorStore**: In-memory vector storage with cosine similarity

## Architecture

```
User Request: "deploy to staging"
         ↓
    Manager.GetRelevantDocs()
         ↓
    ┌─────────────────┐
    │ HybridMatcher   │
    ├─────────────────┤
    │ 1. Keyword      │ ← Fast path (matcher.go)
    │    Matching     │   - Score: command/alias/keyword/example
    │                 │   - Priority multipliers
    ├─────────────────┤
    │ 2. Semantic     │ ← Smart path (semantic.go)
    │    Search       │   - Embeddings (Ollama/OpenAI)
    │                 │   - Cosine similarity
    └─────────────────┘
         ↓
    Top 3 CommandDocs
         ↓
    Injected into LLM prompt
         ↓
    Accurate command generation!
```

## File Structure

```
internal/customcmd/
├── customcmd.go           # Manager, CommandDoc, main API
├── loader.go              # Load .md files from disk
├── parser.go              # Parse YAML + examples
├── matcher.go             # Keyword-based matching
├── semantic.go            # Hybrid semantic search
├── setup.go               # Ollama/OpenAI setup automation
├── templates.go           # Embedded template files
├── embeddings/
│   ├── embedder.go        # Embedder interface
│   ├── ollama.go          # Ollama implementation
│   └── openai.go          # OpenAI implementation
└── vectorstore/
    ├── store.go           # Store interface
    └── memory.go          # In-memory implementation
```

## Core Data Structures

### CommandDoc

Represents a parsed custom command documentation file.

```go
type CommandDoc struct {
    Filename   string    // e.g., "kubectl.md"
    Command    string    // Primary command name
    Aliases    []string  // Alternative names
    Keywords   []string  // Keywords for matching
    Categories []string  // Organizational categories
    Priority   string    // "high", "medium", "low"
    Version    string    // Tool version
    Content    string    // Full markdown content
    Examples   []Example // Parsed User→Command examples
    UpdatedAt  time.Time // File modification time
}

type Example struct {
    UserRequest string // "deploy to staging"
    Command     string // "deploy-tool --env=staging"
}
```

### AgentCommandDoc

Simplified version sent to the agent (to avoid unnecessary dependencies).

```go
type AgentCommandDoc struct {
    Command  string
    Content  string
    Examples []AgentExample
}

type AgentExample struct {
    UserRequest string
    Command     string
}
```

## Manager API

The `Manager` is the main entry point for custom commands.

### Creating a Manager

```go
manager, err := customcmd.NewManager()
if err != nil {
    return fmt.Errorf("failed to create custom command manager: %w", err)
}
```

### Loading/Indexing Commands

```go
// Load all .md files from ~/.please/commands/
if err := manager.Load(); err != nil {
    return fmt.Errorf("failed to load custom commands: %w", err)
}

// Or use Index() which is the same as Load()
if err := manager.Index(ctx); err != nil {
    return fmt.Errorf("failed to index: %w", err)
}
```

### Finding Relevant Docs

```go
// Get up to 3 most relevant docs for a user request
docs := manager.GetRelevantDocs("deploy to staging", 3)

// For agent integration
agentDocs := manager.GetRelevantDocsForAgent("deploy to staging", 3)
```

### Checking Index Status

```go
// Is indexed?
if !manager.IsIndexed() {
    // Prompt user to run `please index`
}

// Needs reindexing?
if manager.NeedsReindex() {
    // Files have been modified since last index
    // Prompt user to reindex
}

// When was it indexed?
indexTime := manager.GetIndexTime()

// How many commands?
count := manager.Count()

// Get all docs (for listing)
allDocs := manager.GetDocs()
```

## Matching Strategies

### Keyword-Only Matching (matcher.go)

**Fast, no dependencies, works offline.**

Scoring algorithm:
```go
func (m *Matcher) scoreDoc(doc CommandDoc, request string) int {
    score := 0
    requestLower := strings.ToLower(request)
    requestWords := strings.Fields(requestLower)

    // 1. Command name match: 100 points
    if strings.Contains(requestLower, strings.ToLower(doc.Command)) {
        score += 100
    }

    // 2. Alias match: 80 points each
    for _, alias := range doc.Aliases {
        if strings.Contains(requestLower, strings.ToLower(alias)) {
            score += 80
        }
    }

    // 3. Keyword match: 10 points each
    for _, keyword := range doc.Keywords {
        if strings.Contains(requestLower, strings.ToLower(keyword)) {
            score += 10
        }
    }

    // 4. Example similarity: 15 points per word overlap
    for _, example := range doc.Examples {
        exampleWords := strings.Fields(strings.ToLower(example.UserRequest))
        overlap := countWordOverlap(requestWords, exampleWords)
        score += overlap * 15
    }

    // 5. Priority multiplier
    multiplier := 1.0
    switch doc.Priority {
    case "high":
        multiplier = 1.3
    case "medium":
        multiplier = 1.1
    }

    return int(float64(score) * multiplier)
}
```

**Best for**:
- Users who want keyword-only (no embeddings)
- Fast lookups with exact keyword matches
- Offline usage

### Hybrid Semantic Search (semantic.go)

**Accurate, requires Ollama or OpenAI.**

Strategy:
```go
func (h *HybridMatcher) FindRelevantDocs(ctx context.Context, request string, maxDocs int) ([]CommandDoc, error) {
    // 1. Try keyword matching first (fast path)
    keywordDocs := h.keywordMatcher.FindRelevantDocs(request, maxDocs)

    // 2. If good keyword matches (score > threshold), use them
    if len(keywordDocs) > 0 && scoreIsGood(keywordDocs[0]) {
        return keywordDocs, nil
    }

    // 3. Fall back to semantic search
    if h.semanticMatcher.indexed {
        semanticDocs, _, err := h.semanticMatcher.Search(ctx, request, maxDocs)
        if err == nil && len(semanticDocs) > 0 {
            return semanticDocs, nil
        }
    }

    // 4. Fall back to keyword matches even if scores are low
    return keywordDocs, nil
}
```

**Best for**:
- Users who want maximum accuracy
- Handling synonyms, paraphrasing, conceptual matches
- Example: "show pods" matches "list kubernetes containers"

## Embedding Providers

### Interface

```go
type Embedder interface {
    // Embed generates an embedding for a single text
    Embed(ctx context.Context, text string) ([]float32, error)

    // EmbedBatch generates embeddings for multiple texts
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

    // Dimensions returns the embedding dimension size
    Dimensions() int

    // Name returns the model name
    Name() string
}
```

### Ollama (Local)

```go
embedder, err := embeddings.NewOllamaEmbedder("http://localhost:11434", "nomic-embed-text")
if err != nil {
    return fmt.Errorf("failed to create Ollama embedder: %w", err)
}

// Generate embedding
embedding, err := embedder.Embed(ctx, "deploy to staging")
// Returns []float32 with 384 dimensions
```

**Specs**:
- Model: `nomic-embed-text`
- Dimensions: 384
- API: `POST http://localhost:11434/api/embeddings`
- Latency: ~50-100ms
- Cost: Free (local)

### OpenAI (API)

```go
apiKey := os.Getenv("OPENAI_API_KEY")
embedder, err := embeddings.NewOpenAIEmbedder(apiKey, "text-embedding-3-small")
if err != nil {
    return fmt.Errorf("failed to create OpenAI embedder: %w", err)
}

// Generate embedding
embedding, err := embedder.Embed(ctx, "deploy to staging")
// Returns []float32 with 1536 dimensions
```

**Specs**:
- Model: `text-embedding-3-small` (default) or `text-embedding-3-large`
- Dimensions: 1536 (small) or 3072 (large)
- API: `POST https://api.openai.com/v1/embeddings`
- Latency: ~100-200ms
- Cost: $0.02 per 1M tokens

### Adding a New Provider

1. Create `internal/customcmd/embeddings/{provider}.go`
2. Implement the `Embedder` interface
3. Add provider type to `config.EmbeddingProvider`
4. Update `setup.go` with provider-specific setup
5. Add connection testing

Example:
```go
type CohereEmbedder struct {
    apiKey string
    model  string
    client *http.Client
}

func NewCohereEmbedder(apiKey, model string) (*CohereEmbedder, error) {
    // Implementation...
}

func (c *CohereEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    // Call Cohere API...
}

// Implement other interface methods...
```

## Vector Store

### Interface

```go
type Store interface {
    // Add stores a vector with associated metadata
    Add(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error

    // Search finds the top K most similar vectors
    Search(ctx context.Context, query []float32, topK int) ([]SearchResult, error)

    // Delete removes a vector by ID
    Delete(ctx context.Context, id string) error

    // Clear removes all vectors
    Clear(ctx context.Context) error

    // Count returns the number of stored vectors
    Count() int
}
```

### In-Memory Store

Current implementation uses in-memory storage with cosine similarity.

**Cosine Similarity**:
```go
func cosineSimilarity(a, b []float32) float32 {
    var dotProduct, normA, normB float32

    for i := range a {
        dotProduct += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }

    if normA == 0 || normB == 0 {
        return 0
    }

    return dotProduct / (sqrt(normA) * sqrt(normB))
}
```

Returns a value between -1 and 1, where 1 means identical direction.

**Usage**:
```go
store := vectorstore.NewMemoryStore()

// Add vectors
store.Add(ctx, "kubectl", embedding1, map[string]interface{}{
    "command": "kubectl",
    "filename": "kubectl.md",
})

// Search
results, err := store.Search(ctx, queryEmbedding, 3)
for _, result := range results {
    fmt.Printf("ID: %s, Score: %.3f\n", result.ID, result.Score)
}
```

### Adding Persistent Storage

For future enhancement, implement a persistent store:

```go
type DiskStore struct {
    dbPath string
    db     *leveldb.DB
    // ...
}

func (d *DiskStore) Add(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
    // Serialize and write to disk
}

func (d *DiskStore) Search(ctx context.Context, query []float32, topK int) ([]SearchResult, error) {
    // Load vectors, compute similarity, return top K
}
```

## File Format

### YAML Frontmatter

```yaml
---
command: kubectl              # Required: primary command name
aliases: ["k8s", "kube"]     # Optional: alternative names
keywords: ["kubernetes", "pods"]  # Optional: matching keywords
categories: ["devops"]       # Optional: organizational categories
priority: high               # Optional: high/medium/low
version: "1.28"              # Optional: tool version
---
```

### Markdown Content

```markdown
# Kubernetes kubectl

kubectl is the Kubernetes command-line tool for managing clusters.

## Common Patterns

- List resources: `kubectl get {type}`
- Describe: `kubectl describe {type} {name}`

## Examples

**User**: "show me all pods"
**Command**: `kubectl get pods`

**User**: "get logs from nginx"
**Command**: `kubectl logs nginx`
```

### Parsing Logic

The parser extracts:
1. **YAML Frontmatter**: `command`, `aliases`, `keywords`, etc.
2. **Full Content**: Entire markdown (for semantic indexing)
3. **Examples**: Lines matching `**User**:` and `**Command**:` pattern

```go
func (p *Parser) Parse(filename string, content []byte) (*CommandDoc, error) {
    // 1. Split frontmatter and body
    frontmatter, body := splitFrontmatter(content)

    // 2. Parse YAML
    var meta Metadata
    yaml.Unmarshal(frontmatter, &meta)

    // 3. Extract examples
    examples := extractExamples(body)

    // 4. Build CommandDoc
    doc := &CommandDoc{
        Filename: filename,
        Command:  meta.Command,
        Aliases:  meta.Aliases,
        // ...
        Examples: examples,
        Content:  string(body),
    }

    return doc, nil
}
```

## Setup Automation

### Ollama Setup

```go
func SetupOllama() error {
    // 1. Check if installed
    if !isOllamaInstalled() {
        // 2. Offer to install
        if confirmed {
            if runtime.GOOS == "darwin" {
                exec.Command("brew", "install", "ollama").Run()
            } else {
                // Download and run install script
            }
        }
    }

    // 3. Start service
    startOllamaService()

    // 4. Pull model
    exec.Command("ollama", "pull", "nomic-embed-text").Run()

    // 5. Test connection
    if err := testOllamaConnection(); err != nil {
        return fmt.Errorf("connection test failed: %w", err)
    }

    return nil
}
```

### OpenAI Setup

```go
func SetupOpenAI() (string, bool, error) {
    // 1. Check env var first
    if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
        // Use env var
        return apiKey, true, nil
    }

    // 2. Prompt for API key
    apiKey := promptForAPIKey()

    // 3. Ask where to store
    useEnvVar := promptAPIKeyStorage()

    // 4. Test connection
    if err := testOpenAIConnection(apiKey); err != nil {
        return "", false, fmt.Errorf("connection test failed: %w", err)
    }

    return apiKey, useEnvVar, nil
}
```

## Integration with Agent

The custom command manager is integrated with the agent via a function getter pattern (to avoid circular dependencies).

**In cmd/please/main.go**:
```go
// Create manager
customCmdManager, err := customcmd.NewManager()
if err != nil {
    // Handle error
}

// Load commands
if err := customCmdManager.Load(); err != nil {
    // Handle error
}

// Create agent
claudeAgent := agent.NewClaudeAgent()

// Set custom doc getter
claudeAgent.SetCustomDocGetter(func(request string, maxDocs int) []agent.CustomCommandDoc {
    // Get docs from manager
    docs := customCmdManager.GetRelevantDocsForAgent(request, maxDocs)

    // Convert to agent type
    agentDocs := make([]agent.CustomCommandDoc, len(docs))
    for i, doc := range docs {
        agentDocs[i] = agent.CustomCommandDoc{
            Command:  doc.Command,
            Content:  doc.Content,
            Examples: convertExamples(doc.Examples),
        }
    }

    return agentDocs
})
```

**In internal/agent/claude.go**:
```go
func (c *ClaudeAgent) buildSystemPrompt(ctx context.Context, request string) string {
    prompt := "You are a CLI command generator..."

    // Add custom command context
    if c.customDocGetter != nil {
        docs := c.customDocGetter(request, 3)
        if len(docs) > 0 {
            prompt += "\n\n# Custom Commands\n\n"
            prompt += c.buildCustomCommandContext(docs)
        }
    }

    return prompt
}

func (c *ClaudeAgent) buildCustomCommandContext(docs []CustomCommandDoc) string {
    var builder strings.Builder

    for _, doc := range docs {
        builder.WriteString(fmt.Sprintf("## %s\n\n", doc.Command))

        // Add up to 5 examples (token budget)
        examples := doc.Examples
        if len(examples) > 5 {
            examples = examples[:5]
        }

        builder.WriteString("Examples:\n")
        for _, ex := range examples {
            builder.WriteString(fmt.Sprintf("- User: %s\n", ex.UserRequest))
            builder.WriteString(fmt.Sprintf("  Command: %s\n", ex.Command))
        }

        builder.WriteString("\n")
    }

    return builder.String()
}
```

## Token Budget Management

To keep prompts efficient, we limit:
- **Max 3 commands** matched per request (configurable via `maxDocsToRetrieve`)
- **Max 5 examples** per command sent to LLM
- **Max 10 common patterns** extracted from content

This prevents token bloat when users have many custom commands.

```go
const (
    MaxDocsPerRequest = 3
    MaxExamplesPerDoc = 5
    MaxPatternsPerDoc = 10
)
```

## Error Handling

### Graceful Degradation

The system should gracefully degrade if:
- Embedding service is unavailable → fall back to keyword-only
- No custom commands indexed → skip custom command context
- File parsing fails → log error, skip that file

```go
func (m *Manager) Load() error {
    loader := NewLoader()
    docs, err := loader.LoadAll(m.commandsDir)
    if err != nil {
        // Don't fail hard, just return empty docs
        log.Printf("Warning: failed to load custom commands: %v", err)
        return nil
    }

    m.docs = docs
    m.indexed = true
    return nil
}
```

### User-Friendly Messages

When errors occur, provide actionable guidance:

```go
// Bad
return fmt.Errorf("connection failed")

// Good
return fmt.Errorf("failed to connect to Ollama at %s. Is Ollama running? Try: ollama serve", baseURL)
```

## Testing

### Unit Tests (To Be Implemented)

```go
// matcher_test.go
func TestMatcher_ScoreDoc(t *testing.T) {
    tests := []struct {
        name     string
        doc      CommandDoc
        request  string
        minScore int
    }{
        {
            name: "exact command match",
            doc: CommandDoc{
                Command: "kubectl",
            },
            request:  "kubectl get pods",
            minScore: 100,
        },
        {
            name: "alias match",
            doc: CommandDoc{
                Command: "kubectl",
                Aliases: []string{"k8s"},
            },
            request:  "k8s get pods",
            minScore: 80,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            matcher := NewMatcher()
            score := matcher.scoreDoc(tt.doc, tt.request)
            if score < tt.minScore {
                t.Errorf("expected score >= %d, got %d", tt.minScore, score)
            }
        })
    }
}

// embeddings/ollama_test.go (mocked)
func TestOllamaEmbedder_Embed(t *testing.T) {
    // Mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return mock embedding
        json.NewEncoder(w).Encode(map[string]interface{}{
            "embedding": make([]float32, 384),
        })
    }))
    defer server.Close()

    embedder, _ := NewOllamaEmbedder(server.URL, "nomic-embed-text")

    embedding, err := embedder.Embed(context.Background(), "test")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }

    if len(embedding) != 384 {
        t.Errorf("expected 384 dimensions, got %d", len(embedding))
    }
}
```

## Performance Considerations

### Indexing Performance

- **Keyword-only**: ~1ms per file (just file I/O + parsing)
- **With embeddings**: ~50-200ms per file (depends on provider)

For 50 commands:
- Keyword: ~50ms total
- Ollama: ~2.5-5 seconds
- OpenAI: ~5-10 seconds

### Search Performance

- **Keyword matching**: <1ms (in-memory string operations)
- **Semantic search**: ~50-200ms (embedding generation + cosine similarity)
- **Hybrid**: ~1-200ms (fast keyword path, slow semantic fallback)

### Memory Usage

- **CommandDoc**: ~1-10KB per doc (depends on content size)
- **Vector embedding**: ~1.5KB (384 dims × 4 bytes) to ~6KB (1536 dims × 4 bytes)
- **For 50 commands**: ~50-500KB memory

## Future Enhancements

1. **Persistent Vector Store**: Save embeddings to disk (LevelDB, SQLite)
2. **Incremental Indexing**: Only reindex changed files
3. **Chunking**: Split long docs into chunks for better retrieval
4. **Reranking**: Use cross-encoder for final ranking
5. **Query Expansion**: Expand user query with synonyms
6. **Negative Examples**: "NOT this command" patterns
7. **Command Suggestions**: Suggest commands based on history
8. **Collaborative Filtering**: Learn from user corrections

## Best Practices

### For Users

1. **Comprehensive Examples**: Add 10-15 diverse examples per command
2. **Good Keywords**: Include synonyms, abbreviations, related terms
3. **Clear Aliases**: Add common alternative names
4. **Priority Levels**: Set `priority: high` for most-used tools
5. **Regular Updates**: Keep docs current as tools evolve

### For Developers

1. **Graceful Degradation**: Always have fallbacks
2. **Token Efficiency**: Limit context sent to LLM
3. **Fast Keyword Path**: Try cheap operations first
4. **User Feedback**: Provide actionable error messages
5. **Testing**: Mock external services (Ollama, OpenAI)
6. **Thread Safety**: Use mutexes for shared state

## Troubleshooting

### "No custom commands found"

```bash
# Check directory exists
ls ~/.please/commands/

# Check files are valid markdown
cat ~/.please/commands/kubectl.md

# Reindex
please index
```

### "Failed to connect to Ollama"

```bash
# Check if Ollama is running
curl http://localhost:11434/api/embeddings

# Start Ollama
ollama serve

# Check model is downloaded
ollama list
```

### "OpenAI API error"

```bash
# Check API key
echo $OPENAI_API_KEY

# Test manually
curl https://api.openai.com/v1/embeddings \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"text-embedding-3-small","input":"test"}'
```

### "Commands not matching"

1. Check keywords in frontmatter match your query
2. Add more examples to .md files
3. Try hybrid strategy if using keyword-only
4. Lower `scoreThreshold` in config to be more permissive

## References

- [YAML Frontmatter Spec](https://jekyllrb.com/docs/front-matter/)
- [Cosine Similarity](https://en.wikipedia.org/wiki/Cosine_similarity)
- [Ollama API](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [OpenAI Embeddings](https://platform.openai.com/docs/guides/embeddings)
- [RAG Overview](https://www.anthropic.com/index/retrieval-augmented-generation)

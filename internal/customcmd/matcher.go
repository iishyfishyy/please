package customcmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
)

// Matcher performs keyword-based matching of user requests to command docs
type Matcher struct {
	docs  []CommandDoc
	debug bool
}

// NewMatcher creates a new matcher
func NewMatcher() *Matcher {
	return NewMatcherWithDebug(false)
}

// NewMatcherWithDebug creates a new matcher with debug logging
func NewMatcherWithDebug(debug bool) *Matcher {
	return &Matcher{
		docs:  []CommandDoc{},
		debug: debug,
	}
}

// SetDocs sets the documents to match against
func (m *Matcher) SetDocs(docs []CommandDoc) {
	m.docs = docs
}

// ScoredDoc represents a document with a match score
type ScoredDoc struct {
	Doc   CommandDoc
	Score int
}

// FindRelevantDocs finds the most relevant documents for a request
func (m *Matcher) FindRelevantDocs(request string, maxDocs int) []CommandDoc {
	if len(m.docs) == 0 {
		return []CommandDoc{}
	}

	requestWords := tokenize(strings.ToLower(request))
	if len(requestWords) == 0 {
		return []CommandDoc{}
	}

	if m.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Matcher: searching %d docs for request words: %v\n", len(m.docs), requestWords)
	}

	var scored []ScoredDoc

	for _, doc := range m.docs {
		score := m.scoreDoc(doc, requestWords)
		if score > 0 {
			scored = append(scored, ScoredDoc{
				Doc:   doc,
				Score: score,
			})
			if m.debug && score > 50 {
				fmt.Fprintf(os.Stderr, "[DEBUG] Matcher:   %s scored %d\n", doc.Command, score)
			}
		}
	}

	// Sort by score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Return top N
	n := min(len(scored), maxDocs)
	result := make([]CommandDoc, n)
	for i := 0; i < n; i++ {
		result[i] = scored[i].Doc
	}

	if m.debug && n > 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG] Matcher: returning top %d docs (best score: %d)\n", n, scored[0].Score)
	}

	return result
}

// scoreDoc calculates a relevance score for a document
func (m *Matcher) scoreDoc(doc CommandDoc, requestWords []string) int {
	score := 0

	// Direct command name match (highest priority)
	commandName := strings.ToLower(doc.Command)
	if containsWord(requestWords, commandName) {
		score += 100
	}

	// Partial command name match (e.g., "kube" matches "kubectl")
	for _, word := range requestWords {
		if strings.Contains(commandName, word) || strings.Contains(word, commandName) {
			score += 50
			break
		}
	}

	// Alias match
	for _, alias := range doc.Aliases {
		aliasLower := strings.ToLower(alias)
		if containsWord(requestWords, aliasLower) {
			score += 80
		}
		// Partial alias match
		for _, word := range requestWords {
			if strings.Contains(aliasLower, word) || strings.Contains(word, aliasLower) {
				score += 40
				break
			}
		}
	}

	// Keyword match
	keywordMatches := 0
	for _, keyword := range doc.Keywords {
		keywordLower := strings.ToLower(keyword)
		if containsWord(requestWords, keywordLower) {
			keywordMatches++
			score += 10
		}
		// Partial keyword match
		for _, word := range requestWords {
			if strings.Contains(keywordLower, word) || strings.Contains(word, keywordLower) {
				score += 3
				break
			}
		}
	}

	// Bonus for multiple keyword matches
	if keywordMatches > 2 {
		score += keywordMatches * 5
	}

	// Category match
	for _, category := range doc.Categories {
		categoryLower := strings.ToLower(category)
		if containsWord(requestWords, categoryLower) {
			score += 5
		}
	}

	// Example match (check if request is similar to known examples)
	for _, example := range doc.Examples {
		exampleWords := tokenize(strings.ToLower(example.UserRequest))
		overlap := wordOverlap(requestWords, exampleWords)
		if overlap > 0 {
			score += overlap * 15 // High value for example matches
		}
	}

	// Priority boost
	switch strings.ToLower(doc.Priority) {
	case "high":
		score = int(float64(score) * 1.3)
	case "medium":
		score = int(float64(score) * 1.1)
	}

	return score
}

// tokenize splits text into words, filtering out common stop words
func tokenize(text string) []string {
	// Stop words to ignore
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "the": true, "in": true,
		"on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true,
		"was": true, "are": true, "were": true, "be": true, "been": true,
		"my": true, "me": true, "i": true, "you": true, "it": true,
	}

	var words []string
	var currentWord strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			currentWord.WriteRune(unicode.ToLower(r))
		} else {
			if currentWord.Len() > 0 {
				word := currentWord.String()
				if !stopWords[word] && len(word) > 1 {
					words = append(words, word)
				}
				currentWord.Reset()
			}
		}
	}

	// Don't forget the last word
	if currentWord.Len() > 0 {
		word := currentWord.String()
		if !stopWords[word] && len(word) > 1 {
			words = append(words, word)
		}
	}

	return words
}

// containsWord checks if a word is in the list
func containsWord(words []string, word string) bool {
	for _, w := range words {
		if w == word {
			return true
		}
	}
	return false
}

// wordOverlap counts how many words are in both lists
func wordOverlap(words1, words2 []string) int {
	count := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				count++
				break
			}
		}
	}
	return count
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

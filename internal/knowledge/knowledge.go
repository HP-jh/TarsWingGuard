package knowledge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// KnowledgeBase manages indexed markdown documents from an Obsidian vault.
type KnowledgeBase struct {
	mu        sync.RWMutex
	vaultPath string
	docs      map[string]*Document
	index     map[string][]string // word -> doc IDs
	voc       *VoiceCorrector
}

// Document represents a single markdown file.
type Document struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Path      string            `json:"path"`
	Content   string            `json:"content"`
	FrontMatter map[string]string `json:"frontmatter,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
}

// SearchResult from the knowledge base.
type SearchResult struct {
	Query   string       `json:"query"`
	Results []DocMatch   `json:"results"`
	Count   int          `json:"count"`
}

type DocMatch struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
	Snippet  string   `json:"snippet"`
}

// IndexRequest to (re)index the vault.
type IndexRequest struct {
	VaultPath string `json:"vault_path,omitempty"`
}

// IndexResult from indexing.
type IndexResult struct {
	Indexed int      `json:"indexed"`
	Errors  []string `json:"errors,omitempty"`
}

func New(vaultPath string) *KnowledgeBase {
	kb := &KnowledgeBase{
		vaultPath: vaultPath,
		docs:      make(map[string]*Document),
		index:     make(map[string][]string),
		voc:       NewVoiceCorrector(),
	}
	return kb
}

// SetVault updates the vault path.
func (kb *KnowledgeBase) SetVault(path string) {
	kb.mu.Lock()
	defer kb.mu.Unlock()
	kb.vaultPath = path
}

// Vault returns current vault path.
func (kb *KnowledgeBase) Vault() string {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	return kb.vaultPath
}

// Index walks the vault and indexes all .md files.
func (kb *KnowledgeBase) Index(req IndexRequest) IndexResult {
	path := kb.vaultPath
	if req.VaultPath != "" {
		path = req.VaultPath
	}

	kb.mu.Lock()
	defer kb.mu.Unlock()

	kb.docs = make(map[string]*Document)
	kb.index = make(map[string][]string)

	if path == "" {
		return IndexResult{Errors: []string{"vault path not set"}}
	}

	var errors []string
	count := 0
	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".md") {
			return nil
		}
		doc, err := kb.parseMarkdown(p, path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", p, err))
			return nil
		}
		kb.docs[doc.ID] = doc
		kb.addToIndex(doc)
		count++
		return nil
	})

	return IndexResult{Indexed: count, Errors: errors}
}

func (kb *KnowledgeBase) parseMarkdown(fullPath, vaultRoot string) (*Document, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	content := string(data)

	// Parse frontmatter (simple YAML-like key: value between ---)
	frontMatter := make(map[string]string)
	var body string
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			for _, line := range strings.Split(parts[1], "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if idx := strings.Index(line, ":"); idx > 0 {
					key := strings.TrimSpace(line[:idx])
					val := strings.TrimSpace(line[idx+1:])
					frontMatter[key] = val
				}
			}
			body = parts[2]
		} else {
			body = content
		}
	} else {
		body = content
	}

	// Extract title from first # heading
	title := filepath.Base(fullPath)
	title = strings.TrimSuffix(title, ".md")
	if m := regexp.MustCompile(`(?m)^#\s+(.+)$`).FindStringSubmatch(body); len(m) > 1 {
		title = strings.TrimSpace(m[1])
	}

	// Extract tags
	var tags []string
	for _, m := range regexp.MustCompile(`#(\w+)`).FindAllStringSubmatch(body, -1) {
		tags = append(tags, m[1])
	}
	if t, ok := frontMatter["tags"]; ok {
		for _, tag := range strings.Split(t, ",") {
			tags = append(tags, strings.TrimSpace(tag))
		}
	}

	rel, _ := filepath.Rel(vaultRoot, fullPath)
	return &Document{
		ID:          strings.TrimSuffix(rel, ".md"),
		Title:       title,
		Path:        rel,
		Content:     strings.TrimSpace(body),
		FrontMatter: frontMatter,
		Tags:        uniqueStrings(tags),
	}, nil
}

func (kb *KnowledgeBase) addToIndex(doc *Document) {
	text := doc.Title + " " + doc.Content
	for _, word := range tokenize(text) {
		kb.index[word] = append(kb.index[word], doc.ID)
	}
	for _, tag := range doc.Tags {
		kb.index["#"+tag] = append(kb.index["#"+tag], doc.ID)
	}
}

// Search finds documents matching the query.
func (kb *KnowledgeBase) Search(query string) SearchResult {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	// Apply voice correction to query
	corrected := kb.voc.Correct(query)
	words := tokenize(corrected)

	scores := make(map[string]float64)
	for _, word := range words {
		for _, id := range kb.index[word] {
			scores[id]++
		}
	}

	var results []DocMatch
	for id, score := range scores {
		doc := kb.docs[id]
		if doc == nil {
			continue
		}
		snippet := makeSnippet(doc.Content, corrected)
		results = append(results, DocMatch{
			Document: *doc,
			Score:    score,
			Snippet:  snippet,
		})
	}

	// Simple sort by score desc
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return SearchResult{
		Query:   corrected,
		Results: results,
		Count:   len(results),
	}
}

func (kb *KnowledgeBase) Status() map[string]any {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	return map[string]any{
		"module":     "knowledge",
		"version":    "0.1.0",
		"vault_path": kb.vaultPath,
		"docs":       len(kb.docs),
		"index_keys": len(kb.index),
	}
}

// Tools returns MCP tool definitions.
func (kb *KnowledgeBase) Tools() []ToolDef {
	return []ToolDef{
		{
			Name:        "knowledge_search",
			Description: "Search the knowledge base for documents matching a query",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Search query (supports voice-corrected input)"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "knowledge_index",
			Description: "(Re)index the Obsidian vault. Optional vault_path to override default.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"vault_path": map[string]any{"type": "string", "description": "Path to Obsidian vault (optional)"},
				},
			},
		},
		{
			Name:        "knowledge_status",
			Description: "Get knowledge base status",
			Schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func (kb *KnowledgeBase) Handle(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	switch name {
	case "knowledge_search":
		var req struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("invalid search args: %w", err)
		}
		result := kb.Search(req.Query)
		b, _ := json.Marshal(result)
		return b, nil
	case "knowledge_index":
		var req IndexRequest
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("invalid index args: %w", err)
		}
		result := kb.Index(req)
		b, _ := json.Marshal(result)
		return b, nil
	case "knowledge_status":
		b, _ := json.Marshal(kb.Status())
		return b, nil
	default:
		return nil, fmt.Errorf("unknown tool %q", name)
	}
}

type ToolDef struct {
	Name        string
	Description string
	Schema      map[string]any
}

func tokenize(text string) []string {
	var words []string
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		w := strings.ToLower(regexp.MustCompile(`[^\w\p{Han}]+`).ReplaceAllString(scanner.Text(), ""))
		if w != "" && len(w) > 1 {
			words = append(words, w)
		}
	}
	return words
}

func makeSnippet(content, query string) string {
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)
	idx := strings.Index(lowerContent, lowerQuery)
	if idx < 0 {
		if len(content) > 200 {
			return content[:200] + "..."
		}
		return content
	}
	start := idx - 60
	if start < 0 {
		start = 0
	}
	end := idx + len(lowerQuery) + 60
	if end > len(content) {
		end = len(content)
	}
	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}
	return snippet
}

func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

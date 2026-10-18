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

// SearchByTag finds documents that have all the specified tags.
func (kb *KnowledgeBase) SearchByTag(tags []string) SearchResult {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	if len(tags) == 0 {
		return SearchResult{Query: "tag:", Results: nil, Count: 0}
	}

	// Find intersection of all tag indices
	var candidates map[string]bool
	for i, tag := range tags {
		tag = strings.ToLower(tag)
		ids, ok := kb.index["#"+tag]
		if !ok {
			return SearchResult{Query: "tag:" + strings.Join(tags, ","), Results: nil, Count: 0}
		}
		if i == 0 {
			candidates = make(map[string]bool)
			for _, id := range ids {
				candidates[id] = true
			}
		} else {
			newSet := make(map[string]bool)
			for _, id := range ids {
				if candidates[id] {
					newSet[id] = true
				}
			}
			candidates = newSet
		}
	}

	var results []DocMatch
	for id := range candidates {
		doc := kb.docs[id]
		if doc == nil {
			continue
		}
		results = append(results, DocMatch{
			Document: *doc,
			Score:    float64(len(tags)),
			Snippet:  makeSnippet(doc.Content, ""),
		})
	}

	return SearchResult{
		Query:   "tag:" + strings.Join(tags, ","),
		Results: results,
		Count:   len(results),
	}
}

// FuzzySearch finds documents using edit-distance fuzzy matching.
func (kb *KnowledgeBase) FuzzySearch(query string, maxDist int) SearchResult {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	if maxDist <= 0 {
		maxDist = 2
	}

	corrected := kb.voc.Correct(query)
	queryWords := tokenize(corrected)

	scores := make(map[string]float64)
	for id, doc := range kb.docs {
		docWords := tokenize(doc.Title + " " + doc.Content)
		matchScore := 0.0
		for _, qw := range queryWords {
			for _, dw := range docWords {
				dist := levenshtein(qw, dw)
				if dist == 0 {
					matchScore += 3.0 // exact match
				} else if dist <= maxDist {
					matchScore += float64(maxDist-dist+1) * 0.5 // fuzzy match
				}
			}
		}
		if matchScore > 0 {
			scores[id] = matchScore
		}
	}

	var results []DocMatch
	for id, score := range scores {
		doc := kb.docs[id]
		snippet := makeSnippet(doc.Content, corrected)
		results = append(results, DocMatch{
			Document: *doc,
			Score:    score,
			Snippet:  snippet,
		})
	}

	// Sort by score desc
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return SearchResult{
		Query:   "fuzzy:" + corrected,
		Results: results,
		Count:   len(results),
	}
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func (kb *KnowledgeBase) Status() map[string]any {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	return map[string]any{
		"module":     "knowledge",
		"version":    "0.3.0",
		"vault_path": kb.vaultPath,
		"docs":       len(kb.docs),
		"index_keys": len(kb.index),
		"features":   []string{"search", "voice_correction", "tag_filter", "fuzzy_search"},
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
		{
			Name:        "knowledge_search_by_tag",
			Description: "Search documents by tags (all tags must match)",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "List of tags to filter by"},
				},
				"required": []string{"tags"},
			},
		},
		{
			Name:        "knowledge_fuzzy_search",
			Description: "Fuzzy search with typo tolerance using Levenshtein distance",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":    map[string]any{"type": "string", "description": "Search query (typo-tolerant)"},
					"max_dist": map[string]any{"type": "integer", "description": "Maximum edit distance (default 2)"},
				},
				"required": []string{"query"},
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
	case "knowledge_search_by_tag":
		var req struct {
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("invalid search_by_tag args: %w", err)
		}
		result := kb.SearchByTag(req.Tags)
		b, _ := json.Marshal(result)
		return b, nil
	case "knowledge_fuzzy_search":
		var req struct {
			Query   string `json:"query"`
			MaxDist int    `json:"max_dist"`
		}
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("invalid fuzzy_search args: %w", err)
		}
		result := kb.FuzzySearch(req.Query, req.MaxDist)
		b, _ := json.Marshal(result)
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

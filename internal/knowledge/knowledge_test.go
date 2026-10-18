package knowledge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestKnowledgeBaseIndexAndSearch(t *testing.T) {
	tmp := t.TempDir()
	// Create a test markdown file
	md := filepath.Join(tmp, "test.md")
	content := `---
tags: security, waf
---
# Test Document

This is a test document about web application firewall.
It contains important security concepts.
`
	if err := os.WriteFile(md, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	kb := New(tmp)
	result := kb.Index(IndexRequest{})
	if result.Indexed != 1 {
		t.Fatalf("indexed = %d, want 1", result.Indexed)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("index errors: %v", result.Errors)
	}

	// Search
	sr := kb.Search("security")
	if sr.Count != 1 {
		t.Fatalf("search count = %d, want 1", sr.Count)
	}
	if sr.Results[0].Document.Title != "Test Document" {
		t.Errorf("title = %q", sr.Results[0].Document.Title)
	}
}

func TestKnowledgeBaseTagSearch(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "a.md"), []byte("---\ntags: go\n---\n# Go Tips\nSome Go tips.\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "b.md"), []byte("# Rust Tips\nSome Rust tips.\n"), 0644)

	kb := New(tmp)
	kb.Index(IndexRequest{})

	sr := kb.Search("go")
	if sr.Count != 1 {
		t.Errorf("tag search count = %d, want 1", sr.Count)
	}
}

func TestKnowledgeBaseVoiceCorrection(t *testing.T) {
	kb := New("")
	sr := kb.Search("ni hao")
	// Should not panic; correction applied
	_ = sr
}

func TestKnowledgeBaseStatus(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "x.md"), []byte("# X\ncontent\n"), 0644)
	kb := New(tmp)
	kb.Index(IndexRequest{})
	st := kb.Status()
	if st["docs"] != 1 {
		t.Errorf("docs = %v, want 1", st["docs"])
	}
}

func TestKnowledgeHandleSearch(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "s.md"), []byte("# Search Test\nFind me here.\n"), 0644)
	kb := New(tmp)
	kb.Index(IndexRequest{})

	args, _ := json.Marshal(map[string]string{"query": "Find"})
	out, err := kb.Handle(context.Background(), "knowledge_search", args)
	if err != nil {
		t.Fatal(err)
	}
	var result SearchResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 {
		t.Errorf("count = %d, want 1", result.Count)
	}
}

func TestKnowledgeHandleIndex(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "i.md"), []byte("# Index\nText.\n"), 0644)
	kb := New(tmp)

	args, _ := json.Marshal(IndexRequest{})
	out, err := kb.Handle(context.Background(), "knowledge_index", args)
	if err != nil {
		t.Fatal(err)
	}
	var result IndexResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	if result.Indexed != 1 {
		t.Errorf("indexed = %d, want 1", result.Indexed)
	}
}

func TestKnowledgeHandleUnknown(t *testing.T) {
	kb := New("")
	_, err := kb.Handle(context.Background(), "unknown", []byte("{}"))
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestKnowledgeSearchByTag(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "a.md"), []byte("---\ntags: security, go\n---\n# SecGo\nSecGo content.\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "b.md"), []byte("---\ntags: go\n---\n# GoOnly\nGoOnly content.\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "c.md"), []byte("# NoTags\nNo tags here.\n"), 0644)

	kb := New(tmp)
	kb.Index(IndexRequest{})

	sr := kb.SearchByTag([]string{"security", "go"})
	if sr.Count != 1 {
		t.Errorf("tag search count = %d, want 1", sr.Count)
	}
	if sr.Count > 0 && sr.Results[0].Document.Title != "SecGo" {
		t.Errorf("title = %q, want SecGo", sr.Results[0].Document.Title)
	}

	// Single tag should match both a.md and b.md
	sr2 := kb.SearchByTag([]string{"go"})
	if sr2.Count != 2 {
		t.Errorf("single tag search count = %d, want 2", sr2.Count)
	}

	// Nonexistent tag
	sr3 := kb.SearchByTag([]string{"nonexistent"})
	if sr3.Count != 0 {
		t.Errorf("nonexistent tag count = %d, want 0", sr3.Count)
	}
}

func TestKnowledgeFuzzySearch(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "f.md"), []byte("# Golang Programming\nThis is about Go programming language.\n"), 0644)

	kb := New(tmp)
	kb.Index(IndexRequest{})

	// Typo: "golng" instead of "golang" — edit distance 1
	sr := kb.FuzzySearch("golng", 2)
	if sr.Count != 1 {
		t.Errorf("fuzzy search count = %d, want 1", sr.Count)
	}
}

func TestKnowledgeHandleSearchByTag(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "t.md"), []byte("---\ntags: test\n---\n# Tagged\nTagged content.\n"), 0644)
	kb := New(tmp)
	kb.Index(IndexRequest{})

	args, _ := json.Marshal(map[string][]string{"tags": {"test"}})
	out, err := kb.Handle(context.Background(), "knowledge_search_by_tag", args)
	if err != nil {
		t.Fatal(err)
	}
	var result SearchResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 {
		t.Errorf("count = %d, want 1", result.Count)
	}
}

func TestKnowledgeHandleFuzzySearch(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "fz.md"), []byte("# Fuzzy Demo\nDemonstration of fuzzy search.\n"), 0644)
	kb := New(tmp)
	kb.Index(IndexRequest{})

	args, _ := json.Marshal(map[string]any{"query": "demostration", "max_dist": 2})
	out, err := kb.Handle(context.Background(), "knowledge_fuzzy_search", args)
	if err != nil {
		t.Fatal(err)
	}
	var result SearchResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 {
		t.Errorf("count = %d, want 1", result.Count)
	}
}

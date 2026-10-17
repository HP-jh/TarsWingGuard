package guard

import (
	"context"
	"encoding/json"
	"testing"
)

func TestGuardScanClean(t *testing.T) {
	g := New(false)
	r := g.Scan(ScanRequest{Payload: "hello world", Path: "/api/users", Method: "GET"})
	if !r.Clean {
		t.Errorf("expected clean, got threats=%v", r.Threats)
	}
	if r.Score != 0 {
		t.Errorf("score = %d, want 0", r.Score)
	}
}

func TestGuardScanXSS(t *testing.T) {
	g := New(false)
	r := g.Scan(ScanRequest{Payload: "<script>alert(1)</script>"})
	if r.Clean {
		t.Error("expected threat detected")
	}
	if len(r.Threats) == 0 {
		t.Fatal("expected threats")
	}
	if r.Threats[0].Type != "xss" {
		t.Errorf("threat type = %q", r.Threats[0].Type)
	}
	if r.Score == 0 {
		t.Error("expected non-zero score")
	}
}

func TestGuardScanSQLi(t *testing.T) {
	g := New(false)
	r := g.Scan(ScanRequest{Payload: "' OR '1'='1"})
	found := false
	for _, th := range r.Threats {
		if th.Type == "sql-injection" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected sql-injection threat, got %v", r.Threats)
	}
}

func TestGuardScanPathTraversal(t *testing.T) {
	g := New(false)
	r := g.Scan(ScanRequest{Payload: "../../../etc/passwd"})
	found := false
	for _, th := range r.Threats {
		if th.Type == "path-traversal" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected path-traversal threat, got %v", r.Threats)
	}
}

func TestGuardScanStrictMode(t *testing.T) {
	g := New(true)
	r := g.Scan(ScanRequest{Payload: "%3cscript%3e"})
	foundEncoded := false
	for _, th := range r.Threats {
		if th.Type == "xss-encoded" {
			foundEncoded = true
		}
	}
	if !foundEncoded {
		t.Errorf("expected xss-encoded in strict mode, got %v", r.Threats)
	}
}

func TestGuardScoreCap(t *testing.T) {
	g := New(true)
	// Multiple threats should cap at 100
	r := g.Scan(ScanRequest{Payload: "<script>' OR '1'='1' ../../../etc/passwd ${echo}"})
	if r.Score > 100 {
		t.Errorf("score = %d, should cap at 100", r.Score)
	}
}

func TestGuardHandleScan(t *testing.T) {
	g := New(false)
	args, _ := json.Marshal(ScanRequest{Payload: "<script>"})
	out, err := g.Handle(context.Background(), "wing_guard_scan", args)
	if err != nil {
		t.Fatal(err)
	}
	var result ScanResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	if result.Clean {
		t.Error("expected dirty result")
	}
}

func TestGuardHandleStatus(t *testing.T) {
	g := New(true)
	out, err := g.Handle(context.Background(), "wing_guard_status", []byte("{}"))
	if err != nil {
		t.Fatal(err)
	}
	var status map[string]any
	if err := json.Unmarshal(out, &status); err != nil {
		t.Fatal(err)
	}
	if status["module"] != "wing-guard" {
		t.Errorf("module = %v", status["module"])
	}
}

func TestGuardHandleUnknown(t *testing.T) {
	g := New(false)
	_, err := g.Handle(context.Background(), "unknown", []byte("{}"))
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

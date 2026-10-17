package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func buildBinary(t *testing.T, bin string) string {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, bin)
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, "./cmd/"+strings.TrimSuffix(bin, ".exe"))
	cmd.Dir = ".."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", bin, err, out)
	}
	return out
}

func TestWingctlList(t *testing.T) {
	wingctl := buildBinary(t, "wingctl")
	cmd := exec.Command(wingctl, "-catalog", "../wing.yaml", "-list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wingctl list: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "wing-guard") {
		t.Errorf("output missing wing-guard:\n%s", out)
	}
}

func TestWingctlResolveTier(t *testing.T) {
	wingctl := buildBinary(t, "wingctl")
	for _, tier := range []string{"basic", "pro", "ultimate"} {
		cmd := exec.Command(wingctl, "-catalog", "../wing.yaml", "-resolve", tier)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("wingctl resolve %s: %v\n%s", tier, err, out)
		}
		if !strings.Contains(string(out), "Resolved") {
			t.Errorf("resolve %s output unexpected:\n%s", tier, out)
		}
	}
}

func TestWingctlDryRun(t *testing.T) {
	wingctl := buildBinary(t, "wingctl")
	cmd := exec.Command(wingctl, "-catalog", "../wing.yaml", "-dry-run")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wingctl dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Dry-run OK") {
		t.Errorf("dry-run output unexpected:\n%s", out)
	}
}

func TestWingctlTSGConfig(t *testing.T) {
	wingctl := buildBinary(t, "wingctl")
	tmp := t.TempDir()
	// Copy wing.yaml into temp dir so ResolveCatalogPath finds it
	wingData, err := os.ReadFile("../wing.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "wing.yaml"), wingData, 0644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(tmp, "config.json")
	cfg := `{"wing":{"mode":"tier","tier":"pro","catalog":"./wing.yaml"}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(wingctl, "-catalog", filepath.Join(tmp, "wing.yaml"), "-tsg-config", cfgPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wingctl tsg-config: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "羽翼防护") && !strings.Contains(string(out), "wing-guard") {
		t.Errorf("tsg-config output missing expected modules:\n%s", out)
	}
}

func TestWingGuardStdio(t *testing.T) {
	wingGuard := buildBinary(t, "wing-guard")
	inBuf := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n")
	inBuf.WriteString(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")
	inBuf.WriteString(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"wing_guard_scan","arguments":{"payload":"<script>"}}}` + "\n")

	cmd := exec.Command(wingGuard, "-transport", "stdio")
	cmd.Stdin = inBuf
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = os.Stderr

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	// Wait for output or timeout
	select {
	case <-done:
		// expected: stdin EOF causes exit
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("wing-guard timeout")
	}

	out := outBuf.String()
	if !strings.Contains(out, "wing_guard_scan") {
		t.Errorf("output missing tool name:\n%s", out)
	}
	// Check that at least one JSON-RPC response has clean=false
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var foundResult bool
	for _, line := range lines {
		var resp map[string]any
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}
		if result, ok := resp["result"].(map[string]any); ok {
			if clean, ok := result["clean"].(bool); ok && !clean {
				foundResult = true
				break
			}
		}
	}
	if !foundResult {
		t.Errorf("expected scan result with clean=false in output:\n%s", out)
	}
}

func TestWingctlVersion(t *testing.T) {
	wingctl := buildBinary(t, "wingctl")
	cmd := exec.Command(wingctl, "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wingctl version: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "0.1.0") {
		t.Errorf("version output missing 0.1.0:\n%s", out)
	}
}

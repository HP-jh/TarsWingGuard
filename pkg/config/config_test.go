package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTSGConfig(t *testing.T) {
	json := `{"wing":{"mode":"tier","tier":"pro","catalog":"custom.yaml"}}`
	tmp := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(tmp, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadTSGConfig(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != "tier" {
		t.Errorf("mode = %q", cfg.Mode)
	}
	if cfg.Tier != "pro" {
		t.Errorf("tier = %q", cfg.Tier)
	}
}

func TestLoadTSGConfigPathTraversal(t *testing.T) {
	_, err := LoadTSGConfig("/etc/shadow")
	if err == nil {
		t.Fatal("expected path traversal denial")
	}
}

func TestResolveCatalogPath(t *testing.T) {
	cfg := &WingConfig{Catalog: "modules/wing.yaml"}
	path := ResolveCatalogPath(cfg, "/opt/tsg/config.json")
	if path != "/opt/tsg/modules/wing.yaml" {
		t.Errorf("path = %q", path)
	}
}

func TestResolveCatalogPathDefault(t *testing.T) {
	cfg := &WingConfig{}
	path := ResolveCatalogPath(cfg, "/opt/tsg/config.json")
	if path != "/opt/tsg/wing.yaml" {
		t.Errorf("default path = %q", path)
	}
}

func TestWingConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     WingConfig
		wantErr bool
	}{
		{"tier-ok", WingConfig{Mode: "tier", Tier: "pro"}, false},
		{"tier-missing", WingConfig{Mode: "tier"}, true},
		{"unlock-ok", WingConfig{Mode: "unlock", Unlock: "羽翼防护"}, false},
		{"unlock-empty", WingConfig{Mode: "unlock", Unlock: "  "}, true},
		{"modules-ok", WingConfig{Mode: "modules", Modules: []string{"a"}}, false},
		{"modules-empty", WingConfig{Mode: "modules"}, true},
		{"mode-missing", WingConfig{}, true},
		{"invalid-mode", WingConfig{Mode: "fancy"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestToTierConfig(t *testing.T) {
	w := WingConfig{Mode: "tier", Tier: "ultimate", Modules: []string{"a"}}
	c := w.ToTierConfig()
	if c.Mode != "tier" || c.Tier != "ultimate" {
		t.Errorf("tier config = %+v", c)
	}
}

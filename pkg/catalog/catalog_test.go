package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseValidCatalog(t *testing.T) {
	yaml := `
version: "0.1.0"
schema: wing/v1
core:
  name: TarsWingCore
  description: Core defense out-of-box
  version: "0.1.0"
modules:
  - id: wing-guard
    name: "羽翼防护"
    description: Advanced WAF + zero-trust
    version: "0.1.0"
    tier: pro
    transport: stdio
    command: wing-guard
`
	tmp := filepath.Join(t.TempDir(), "wing.yaml")
	if err := os.WriteFile(tmp, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	cat, err := Parse(tmp)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if cat.Version != "0.1.0" {
		t.Errorf("version = %q, want 0.1.0", cat.Version)
	}
	if len(cat.Modules) != 1 {
		t.Fatalf("modules count = %d, want 1", len(cat.Modules))
	}
	if cat.Modules[0].ID != "wing-guard" {
		t.Errorf("module id = %q", cat.Modules[0].ID)
	}
}

func TestParsePathTraversalDenied(t *testing.T) {
	_, err := Parse("/etc/passwd")
	if err == nil {
		t.Fatal("expected error for absolute path outside cwd")
	}
}

func TestValidateDuplicateID(t *testing.T) {
	cat := &Catalog{Version: "0.1.0", Modules: []Module{
		{ID: "a", Name: "A", Version: "1", Tier: TierBasic},
		{ID: "a", Name: "B", Version: "1", Tier: TierBasic},
	}}
	if err := cat.Validate(); err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestValidateSelfDependency(t *testing.T) {
	cat := &Catalog{Version: "0.1.0", Modules: []Module{
		{ID: "a", Name: "A", Version: "1", Tier: TierBasic, Deps: []string{"a"}},
	}}
	if err := cat.Validate(); err == nil {
		t.Fatal("expected self-dependency error")
	}
}

func TestValidateMissingDependency(t *testing.T) {
	cat := &Catalog{Version: "0.1.0", Modules: []Module{
		{ID: "a", Name: "A", Version: "1", Tier: TierBasic, Deps: []string{"b"}},
	}}
	if err := cat.Validate(); err == nil {
		t.Fatal("expected missing dependency error")
	}
}

func TestResolveTier(t *testing.T) {
	cat := &Catalog{Version: "0.1.0", Modules: []Module{
		{ID: "basic-mod", Name: "Basic", Version: "1", Tier: TierBasic},
		{ID: "pro-mod", Name: "Pro", Version: "1", Tier: TierPro},
		{ID: "ult-mod", Name: "Ultimate", Version: "1", Tier: TierUltimate},
		{ID: "disabled-mod", Name: "Disabled", Version: "1", Tier: TierPro, Disabled: true},
	}}
	basic := cat.ResolveTier(TierBasic)
	if len(basic) != 1 || basic[0].ID != "basic-mod" {
		t.Errorf("basic tier = %v", basic)
	}
	pro := cat.ResolveTier(TierPro)
	if len(pro) != 2 {
		t.Errorf("pro tier count = %d, want 2", len(pro))
	}
	ult := cat.ResolveTier(TierUltimate)
	if len(ult) != 3 {
		t.Errorf("ultimate tier count = %d, want 3", len(ult))
	}
}

func TestResolveModulesWithDeps(t *testing.T) {
	cat := &Catalog{Version: "0.1.0", Modules: []Module{
		{ID: "base", Name: "Base", Version: "1", Tier: TierBasic},
		{ID: "mid", Name: "Mid", Version: "1", Tier: TierPro, Deps: []string{"base"}},
		{ID: "top", Name: "Top", Version: "1", Tier: TierUltimate, Deps: []string{"mid"}},
	}}
	mods, err := cat.ResolveModules([]string{"top"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 3 {
		t.Fatalf("resolved count = %d, want 3", len(mods))
	}
	// deps should come first (DFS order)
	if mods[0].ID != "base" || mods[1].ID != "mid" || mods[2].ID != "top" {
		t.Errorf("order = %v", mods)
	}
}

func TestInvalidTier(t *testing.T) {
	if Tier("invalid").Valid() {
		t.Error("invalid tier should not be valid")
	}
}

func TestInvalidTransport(t *testing.T) {
	if Transport("ws").Valid() {
		t.Error("ws transport should not be valid")
	}
}

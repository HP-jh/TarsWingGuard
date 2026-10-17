package tier

import (
	"testing"

	"github.com/HP-jh/TarsWingGuard/pkg/catalog"
)

func makeTestCatalog() *catalog.Catalog {
	return &catalog.Catalog{Version: "0.1.0", Modules: []catalog.Module{
		{ID: "core-waf", Name: "Core WAF", Version: "1", Tier: catalog.TierBasic},
		{ID: "wing-guard", Name: "羽翼防护", Version: "1", Tier: catalog.TierPro},
		{ID: "knowledge", Name: "知识大脑", Version: "1", Tier: catalog.TierPro},
		{ID: "extreme", Name: "极致档", Version: "1", Tier: catalog.TierUltimate},
	}}
}

func TestResolveTierBasic(t *testing.T) {
	r := NewResolver(makeTestCatalog())
	mods, err := r.Resolve(Config{Mode: ModeTier, Tier: "basic"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].ID != "core-waf" {
		t.Errorf("basic = %v", mods)
	}
}

func TestResolveTierPro(t *testing.T) {
	r := NewResolver(makeTestCatalog())
	mods, err := r.Resolve(Config{Mode: ModeTier, Tier: "pro"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 3 {
		t.Errorf("pro count = %d, want 3", len(mods))
	}
}

func TestResolveTierUltimate(t *testing.T) {
	r := NewResolver(makeTestCatalog())
	mods, err := r.Resolve(Config{Mode: ModeTier, Tier: "ultimate"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 4 {
		t.Errorf("ultimate count = %d, want 4", len(mods))
	}
}

func TestResolveUnlock(t *testing.T) {
	r := NewResolver(makeTestCatalog())
	mods, err := r.Resolve(Config{Mode: ModeUnlock, Unlock: "羽翼防护"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 3 {
		t.Errorf("unlock count = %d, want 3", len(mods))
	}
}

func TestResolveUnlockEmpty(t *testing.T) {
	r := NewResolver(makeTestCatalog())
	_, err := r.Resolve(Config{Mode: ModeUnlock, Unlock: "  "})
	if err == nil {
		t.Fatal("expected error for empty unlock")
	}
}

func TestResolveModules(t *testing.T) {
	r := NewResolver(makeTestCatalog())
	mods, err := r.Resolve(Config{Mode: ModeModules, Modules: []string{"wing-guard", "knowledge"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 2 {
		t.Errorf("modules count = %d, want 2", len(mods))
	}
}

func TestResolveInvalidTier(t *testing.T) {
	r := NewResolver(makeTestCatalog())
	_, err := r.Resolve(Config{Mode: ModeTier, Tier: "gold"})
	if err == nil {
		t.Fatal("expected error for invalid tier")
	}
}

func TestTierSummary(t *testing.T) {
	summary := TierSummary(makeTestCatalog())
	if summary[catalog.TierBasic].ModuleCount != 1 {
		t.Errorf("basic count = %d", summary[catalog.TierBasic].ModuleCount)
	}
	if summary[catalog.TierPro].ModuleCount != 3 {
		t.Errorf("pro count = %d", summary[catalog.TierPro].ModuleCount)
	}
}

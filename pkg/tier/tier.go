package tier

import (
	"fmt"
	"strings"

	"github.com/HP-jh/TarsWingGuard/pkg/catalog"
)

// Mode determines how modules are selected.
type Mode string

const (
	ModeTier    Mode = "tier"    // wing.tier = basic/pro/ultimate
	ModeUnlock  Mode = "unlock"  // wing.unlock = "羽翼防护" enables all pro+
	ModeModules Mode = "modules" // wing.modules = [id1, id2]
)

// Config is the wing section inside TSG config.json
type Config struct {
	Mode    Mode     `json:"mode"`
	Tier    string   `json:"tier,omitempty"`
	Unlock  string   `json:"unlock,omitempty"`
	Modules []string `json:"modules,omitempty"`
}

// Resolver maps wing config to enabled module IDs.
type Resolver struct {
	cat *catalog.Catalog
}

func NewResolver(cat *catalog.Catalog) *Resolver {
	return &Resolver{cat: cat}
}

// Resolve returns module IDs enabled by the given wing config.
func (r *Resolver) Resolve(cfg Config) ([]catalog.Module, error) {
	switch cfg.Mode {
	case ModeTier, "":
		t := catalog.Tier(cfg.Tier)
		if !t.Valid() {
			return nil, fmt.Errorf("invalid tier %q", cfg.Tier)
		}
		return r.cat.ResolveTier(t), nil
	case ModeUnlock:
		return r.resolveUnlock(cfg.Unlock)
	case ModeModules:
		return r.cat.ResolveModules(cfg.Modules)
	default:
		return nil, fmt.Errorf("unknown wing mode %q", cfg.Mode)
	}
}

func (r *Resolver) resolveUnlock(key string) ([]catalog.Module, error) {
	// "羽翼防护" (or any non-empty string) unlocks pro tier
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("unlock key empty")
	}
	// Normalize: accept Chinese "羽翼防护" or any non-empty token
	_ = key // any non-empty key unlocks pro tier in v0.1.0
	return r.cat.ResolveTier(catalog.TierPro), nil
}

// AvailableModes returns human-readable mode descriptions.
func AvailableModes() map[Mode]string {
	return map[Mode]string{
		ModeTier:    "tier — choose basic/pro/ultimate package",
		ModeUnlock:  "unlock — enter a key phrase to unlock pro tier",
		ModeModules: "modules — list specific module IDs to enable",
	}
}

// TierSummary describes what each tier includes.
func TierSummary(cat *catalog.Catalog) map[catalog.Tier]TierInfo {
	out := make(map[catalog.Tier]TierInfo)
	for _, t := range []catalog.Tier{catalog.TierBasic, catalog.TierPro, catalog.TierUltimate} {
		mods := cat.ResolveTier(t)
		var names []string
		for _, m := range mods {
			names = append(names, m.Name)
		}
		out[t] = TierInfo{
			Name:        string(t),
			ModuleCount: len(mods),
			Modules:     names,
		}
	}
	return out
}

type TierInfo struct {
	Name        string
	ModuleCount int
	Modules     []string
}

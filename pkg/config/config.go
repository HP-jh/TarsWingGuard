package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HP-jh/TarsWingGuard/pkg/catalog"
	"github.com/HP-jh/TarsWingGuard/pkg/tier"
)

// WingConfig is the wing section that lives inside TSG's config.json
type WingConfig struct {
	Mode    string   `json:"mode"`
	Tier    string   `json:"tier,omitempty"`
	Unlock  string   `json:"unlock,omitempty"`
	Modules []string `json:"modules,omitempty"`
	Catalog string   `json:"catalog,omitempty"` // path to wing.yaml
}

// TSGConfig is a minimal subset of TSG config.json that includes the wing section.
type TSGConfig struct {
	Wing WingConfig `json:"wing"`
}

// LoadTSGConfig reads TSG config.json and extracts wing config.
func LoadTSGConfig(path string) (*WingConfig, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("config path resolve: %w", err)
	}
	if strings.Contains(abs, "..") {
		return nil, fmt.Errorf("config path traversal denied: %s", path)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("config read: %w", err)
	}
	var cfg TSGConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config parse: %w", err)
	}
	return &cfg.Wing, nil
}

// ResolveCatalogPath returns the absolute path to the wing catalog.
func ResolveCatalogPath(wingCfg *WingConfig, tsgConfigPath string) string {
	if wingCfg.Catalog != "" {
		if filepath.IsAbs(wingCfg.Catalog) {
			return wingCfg.Catalog
		}
		base := filepath.Dir(tsgConfigPath)
		return filepath.Join(base, wingCfg.Catalog)
	}
	// default: same dir as config.json
	base := filepath.Dir(tsgConfigPath)
	return filepath.Join(base, "wing.yaml")
}

// ToTierConfig converts WingConfig to tier.Config.
func (w *WingConfig) ToTierConfig() tier.Config {
	return tier.Config{
		Mode:    tier.Mode(w.Mode),
		Tier:    w.Tier,
		Unlock:  w.Unlock,
		Modules: w.Modules,
	}
}

// Validate checks wing config fields.
func (w *WingConfig) Validate() error {
	if w.Mode == "" {
		return fmt.Errorf("wing.mode required")
	}
	switch tier.Mode(w.Mode) {
	case tier.ModeTier:
		if !catalog.Tier(w.Tier).Valid() {
			return fmt.Errorf("wing.tier must be basic/pro/ultimate")
		}
	case tier.ModeUnlock:
		if strings.TrimSpace(w.Unlock) == "" {
			return fmt.Errorf("wing.unlock required when mode=unlock")
		}
	case tier.ModeModules:
		if len(w.Modules) == 0 {
			return fmt.Errorf("wing.modules required when mode=modules")
		}
	default:
		return fmt.Errorf("wing.mode must be tier/unlock/modules")
	}
	return nil
}

package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Tier levels
type Tier string

const (
	TierBasic    Tier = "basic"
	TierPro      Tier = "pro"
	TierUltimate Tier = "ultimate"
)

func (t Tier) Valid() bool {
	switch t {
	case TierBasic, TierPro, TierUltimate:
		return true
	}
	return false
}

// Transport types for MCP servers
type Transport string

const (
	TransportStdio      Transport = "stdio"
	TransportSSE        Transport = "sse"
	TransportStreamable Transport = "streamable"
)

func (t Transport) Valid() bool {
	switch t {
	case TransportStdio, TransportSSE, TransportStreamable:
		return true
	}
	return false
}

// Module represents a single MCP-capable wing module
type Module struct {
	ID          string    `yaml:"id"`
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Version     string    `yaml:"version"`
	Tier        Tier      `yaml:"tier"`
	Transport   Transport `yaml:"transport"`
	Command     string    `yaml:"command"`       // executable path or command
	Args        []string  `yaml:"args,omitempty"`
	Env         []EnvVar  `yaml:"env,omitempty"`
	Deps        []string  `yaml:"deps,omitempty"` // module IDs this depends on
	RepoURL     string    `yaml:"repo_url,omitempty"`
	DocURL      string    `yaml:"doc_url,omitempty"`
	Disabled    bool      `yaml:"disabled,omitempty"`
}

type EnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Catalog is the top-level wing module manifest
type Catalog struct {
	Version   string    `yaml:"version"`
	Schema    string    `yaml:"schema"`
	Core      CoreDesc  `yaml:"core"`
	Modules   []Module  `yaml:"modules"`
}

type CoreDesc struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// Parse reads a catalog from a YAML file, validating paths and tiers.
func Parse(path string) (*Catalog, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("catalog path resolve: %w", err)
	}
	// prevent path traversal: must be within cwd or explicit absolute
	if strings.Contains(abs, "..") {
		return nil, fmt.Errorf("catalog path traversal denied: %s", path)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("catalog read: %w", err)
	}
	var cat Catalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("catalog parse: %w", err)
	}
	if cat.Schema == "" {
		cat.Schema = "wing/v1"
	}
	if err := cat.Validate(); err != nil {
		return nil, fmt.Errorf("catalog validate: %w", err)
	}
	return &cat, nil
}

// Validate checks catalog integrity.
func (c *Catalog) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("catalog version required")
	}
	seen := make(map[string]bool)
	for i, m := range c.Modules {
		if m.ID == "" {
			return fmt.Errorf("module[%d]: id required", i)
		}
		if seen[m.ID] {
			return fmt.Errorf("module[%d]: duplicate id %q", i, m.ID)
		}
		seen[m.ID] = true
		if m.Name == "" {
			return fmt.Errorf("module %q: name required", m.ID)
		}
		if m.Version == "" {
			return fmt.Errorf("module %q: version required", m.ID)
		}
		if !m.Tier.Valid() {
			return fmt.Errorf("module %q: invalid tier %q", m.ID, m.Tier)
		}
		if m.Transport != "" && !m.Transport.Valid() {
			return fmt.Errorf("module %q: invalid transport %q", m.ID, m.Transport)
		}
		for _, dep := range m.Deps {
			if dep == m.ID {
				return fmt.Errorf("module %q: self-dependency", m.ID)
			}
		}
	}
	// check deps exist
	for i, m := range c.Modules {
		for _, dep := range m.Deps {
			if !seen[dep] {
				return fmt.Errorf("module[%d] %q: dependency %q not found", i, m.ID, dep)
			}
		}
	}
	return nil
}

// ResolveTier returns modules enabled for a given tier.
func (c *Catalog) ResolveTier(t Tier) []Module {
	var out []Module
	for _, m := range c.Modules {
		if m.Disabled {
			continue
		}
		switch t {
		case TierUltimate:
			out = append(out, m)
		case TierPro:
			if m.Tier == TierBasic || m.Tier == TierPro {
				out = append(out, m)
			}
		case TierBasic:
			if m.Tier == TierBasic {
				out = append(out, m)
			}
		}
	}
	return out
}

// ResolveModules returns modules by explicit IDs, including transitive deps.
func (c *Catalog) ResolveModules(ids []string) ([]Module, error) {
	idx := make(map[string]Module)
	for _, m := range c.Modules {
		idx[m.ID] = m
	}
	var out []Module
	visited := make(map[string]bool)
	var dfs func(string) error
	dfs = func(id string) error {
		if visited[id] {
			return nil
		}
		m, ok := idx[id]
		if !ok {
			return fmt.Errorf("module %q not found", id)
		}
		visited[id] = true
		for _, dep := range m.Deps {
			if err := dfs(dep); err != nil {
				return err
			}
		}
		out = append(out, m)
		return nil
	}
	for _, id := range ids {
		if err := dfs(id); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ModuleByID looks up a single module.
func (c *Catalog) ModuleByID(id string) (Module, bool) {
	for _, m := range c.Modules {
		if m.ID == id {
			return m, true
		}
	}
	return Module{}, false
}

package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Guard implements the wing-guard MCP module: advanced WAF + zero-trust signals.
type Guard struct {
	strictMode bool
}

func New(strict bool) *Guard {
	return &Guard{strictMode: strict}
}

// ScanRequest is the input to wing-guard scan tool.
type ScanRequest struct {
	Payload string `json:"payload"`
	Path    string `json:"path,omitempty"`
	Method  string `json:"method,omitempty"`
}

// ScanResult is the output.
type ScanResult struct {
	Threats []Threat `json:"threats"`
	Clean   bool     `json:"clean"`
	Score   int      `json:"score"` // 0-100, higher = more risky
}

type Threat struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
	Pos     int    `json:"position"`
}

// Scan performs a lightweight threat scan on the payload.
func (g *Guard) Scan(req ScanRequest) ScanResult {
	var threats []Threat
	score := 0
	payload := req.Payload

	// Basic rule checks
	if strings.Contains(payload, "<script") || strings.Contains(payload, "javascript:") {
		threats = append(threats, Threat{Type: "xss", Pattern: "script tag", Pos: strings.Index(payload, "<")})
		score += 40
	}
	if strings.Contains(payload, "' OR '") || strings.Contains(payload, "'; DROP") {
		threats = append(threats, Threat{Type: "sql-injection", Pattern: "sql pattern", Pos: strings.Index(payload, "'")})
		score += 50
	}
	if strings.Contains(payload, "../") || strings.Contains(payload, "..\\") {
		threats = append(threats, Threat{Type: "path-traversal", Pattern: "dot-dot-slash", Pos: strings.Index(payload, "..")})
		score += 35
	}
	if strings.Contains(payload, "${") || strings.Contains(payload, "`") {
		threats = append(threats, Threat{Type: "command-injection", Pattern: "template/exec", Pos: strings.Index(payload, "$")})
		score += 45
	}

	if g.strictMode {
		// stricter: check for encoded variants
		lower := strings.ToLower(payload)
		if strings.Contains(lower, "%3c") || strings.Contains(lower, "%3e") {
			threats = append(threats, Threat{Type: "xss-encoded", Pattern: "url-encoded tag", Pos: 0})
			score += 20
		}
		if strings.Contains(lower, "%00") {
			threats = append(threats, Threat{Type: "null-byte", Pattern: "null injection", Pos: strings.Index(lower, "%00")})
			score += 25
		}
	}

	return ScanResult{
		Threats: threats,
		Clean:   len(threats) == 0,
		Score:   min(score, 100),
	}
}

// Tools returns MCP tool definitions for wing-guard.
func (g *Guard) Tools() []ToolDef {
	return []ToolDef{
		{
			Name:        "wing_guard_scan",
			Description: "Scan a request payload for security threats (XSS, SQLi, path traversal, command injection)",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"payload": map[string]any{"type": "string", "description": "The request payload to scan"},
					"path":    map[string]any{"type": "string", "description": "Request path (optional)"},
					"method":  map[string]any{"type": "string", "description": "HTTP method (optional)"},
				},
				"required": []string{"payload"},
			},
		},
		{
			Name:        "wing_guard_status",
			Description: "Get wing-guard module status and configuration",
			Schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func (g *Guard) Handle(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	switch name {
	case "wing_guard_scan":
		var req ScanRequest
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("invalid scan args: %w", err)
		}
		result := g.Scan(req)
		b, _ := json.Marshal(result)
		return b, nil
	case "wing_guard_status":
		b, _ := json.Marshal(map[string]any{
			"module":      "wing-guard",
			"version":     "0.1.0",
			"strict_mode": g.strictMode,
			"rules":       6,
		})
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

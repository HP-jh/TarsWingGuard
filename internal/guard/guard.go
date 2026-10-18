package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
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
	lower := strings.ToLower(payload)

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

	// NoSQL injection detection
	if strings.Contains(lower, "$ne") || strings.Contains(lower, "$gt") ||
		strings.Contains(lower, "$where") || strings.Contains(lower, "__proto__") ||
		strings.Contains(lower, "$regex") {
		threats = append(threats, Threat{Type: "nosql-injection", Pattern: "nosql operator", Pos: strings.Index(lower, "$")})
		score += 40
	}

	// LDAP injection detection
	if strings.Contains(payload, "*") && (strings.Contains(payload, "(") || strings.Contains(payload, "|") || strings.Contains(payload, "&")) {
		threats = append(threats, Threat{Type: "ldap-injection", Pattern: "ldap wildcard", Pos: strings.Index(payload, "*")})
		score += 35
	}

	// XXE / XML external entity detection
	if strings.Contains(lower, "<!entity") || strings.Contains(lower, "system") || strings.Contains(lower, "file://") {
		threats = append(threats, Threat{Type: "xxe", Pattern: "xml external entity", Pos: strings.Index(lower, "<")})
		score += 45
	}

	// SSRF: detect internal IP addresses in URLs within payload
	if ipThreats, ipScore := g.checkSSRF(payload); len(ipThreats) > 0 {
		threats = append(threats, ipThreats...)
		score += ipScore
	}

	// Path-context awareness: sensitive paths get extra scrutiny
	if req.Path != "" {
		if pathThreats, pathScore := g.checkSensitivePath(req.Path, req.Method); len(pathThreats) > 0 {
			threats = append(threats, pathThreats...)
			score += pathScore
		}
	}

	if g.strictMode {
		// stricter: check for encoded variants
		if strings.Contains(lower, "%3c") || strings.Contains(lower, "%3e") {
			threats = append(threats, Threat{Type: "xss-encoded", Pattern: "url-encoded tag", Pos: 0})
			score += 20
		}
		if strings.Contains(lower, "%00") {
			threats = append(threats, Threat{Type: "null-byte", Pattern: "null injection", Pos: strings.Index(lower, "%00")})
			score += 25
		}
		// Strict mode: double encoding detection
		if strings.Contains(lower, "%253c") || strings.Contains(lower, "%253e") {
			threats = append(threats, Threat{Type: "xss-double-encoded", Pattern: "double-encoded tag", Pos: 0})
			score += 30
		}
	}

	return ScanResult{
		Threats: threats,
		Clean:   len(threats) == 0,
		Score:   min(score, 100),
	}
}

// checkSSRF detects Server-Side Request Forgery attempts by finding
// internal/private IP addresses embedded in URLs within the payload.
func (g *Guard) checkSSRF(payload string) ([]Threat, int) {
	var threats []Threat
	score := 0

	// Extract potential URLs from payload
	words := strings.FieldsFunc(payload, func(r rune) bool {
		return r == ' ' || r == '"' || r == '\'' || r == ';' || r == ',' || r == '\n' || r == '\t'
	})

	for _, w := range words {
		u, err := url.Parse(w)
		if err != nil {
			continue
		}
		host := u.Hostname()
		if host == "" {
			continue
		}
		ip := net.ParseIP(host)
		if ip == nil {
			// Try resolving as IP directly if no scheme
			ip = net.ParseIP(host)
		}
		if ip != nil && isPrivateIP(ip) {
			threats = append(threats, Threat{Type: "ssrf", Pattern: "internal ip: " + host, Pos: strings.Index(payload, w)})
			score += 50
		} else if isPrivateHost(host) {
			threats = append(threats, Threat{Type: "ssrf", Pattern: "internal host: " + host, Pos: strings.Index(payload, w)})
			score += 45
		}
	}
	return threats, score
}

// isPrivateIP returns true for RFC1918 / loopback / link-local addresses.
func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// isPrivateHost returns true for well-known internal hostnames.
func isPrivateHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "localhost.localdomain", "ip6-localhost", "ip6-loopback":
		return true
	}
	return false
}

// checkSensitivePath detects requests targeting administrative or sensitive endpoints.
func (g *Guard) checkSensitivePath(path, method string) ([]Threat, int) {
	var threats []Threat
	score := 0
	lowerPath := strings.ToLower(path)

	sensitive := []struct {
		prefix string
		score  int
	}{
		{"/admin", 15},
		{"/config", 15},
		{"/.env", 20},
		{"/.git", 25},
		{"/api/internal", 20},
		{"/debug", 20},
		{"/actuator", 20},
		{"/wp-admin", 15},
		{"/phpmyadmin", 20},
	}

	for _, s := range sensitive {
		if strings.HasPrefix(lowerPath, s.prefix) {
			threats = append(threats, Threat{Type: "sensitive-path", Pattern: s.prefix, Pos: 0})
			score += s.score
		}
	}

	// Non-GET/POST to sensitive paths is extra suspicious
	if method != "" && method != "GET" && method != "POST" {
		if strings.HasPrefix(lowerPath, "/admin") || strings.HasPrefix(lowerPath, "/api/internal") {
			threats = append(threats, Threat{Type: "sensitive-method", Pattern: method + " " + lowerPath, Pos: 0})
			score += 10
		}
	}

	return threats, score
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
			"version":     "0.3.0",
			"strict_mode": g.strictMode,
			"rules":       12,
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

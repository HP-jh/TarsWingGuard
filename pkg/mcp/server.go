package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Server is a minimal MCP server implementing stdio/SSE/streamable transports.
type Server struct {
	name        string
	version     string
	tools       map[string]Tool
	handlers    map[string]ToolHandler
	mu          sync.RWMutex
	transport   Transport
	initialized bool
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type ToolHandler func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)

type Transport interface {
	Serve(s *Server) error
	Shutdown(ctx context.Context) error
}

func NewServer(name, version string) *Server {
	return &Server{
		name:     name,
		version:  version,
		tools:    make(map[string]Tool),
		handlers: make(map[string]ToolHandler),
	}
}

func (s *Server) RegisterTool(t Tool, h ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[t.Name] = t
	s.handlers[t.Name] = h
}

func (s *Server) SetTransport(t Transport) {
	s.transport = t
}

func (s *Server) Serve() error {
	if s.transport == nil {
		return fmt.Errorf("no transport configured")
	}
	return s.transport.Serve(s)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.transport == nil {
		return nil
	}
	return s.transport.Shutdown(ctx)
}

func (s *Server) handleInitialize() json.RawMessage {
	return mustJSON(map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
	})
}

func (s *Server) handleToolsList() json.RawMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []Tool
	for _, t := range s.tools {
		list = append(list, t)
	}
	return mustJSON(map[string]any{"tools": list})
}

func (s *Server) handleToolCall(name string, args json.RawMessage) (json.RawMessage, error) {
	s.mu.RLock()
	h, ok := s.handlers[name]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return h(context.Background(), args)
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// JSONRPCMessage is a minimal JSON-RPC envelope.
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) ProcessMessage(raw []byte) ([]byte, error) {
	var req JSONRPCMessage
	if err := json.Unmarshal(raw, &req); err != nil {
		return json.Marshal(JSONRPCMessage{
			JSONRPC: "2.0",
			Error:   &JSONRPCError{Code: -32700, Message: "parse error"},
		})
	}
	resp := JSONRPCMessage{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = s.handleInitialize()
		s.initialized = true
	case "tools/list":
		if !s.initialized {
			resp.Error = &JSONRPCError{Code: -32002, Message: "not initialized"}
			break
		}
		resp.Result = s.handleToolsList()
	case "tools/call":
		if !s.initialized {
			resp.Error = &JSONRPCError{Code: -32002, Message: "not initialized"}
			break
		}
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp.Error = &JSONRPCError{Code: -32602, Message: "invalid params"}
			break
		}
		result, err := s.handleToolCall(p.Name, p.Arguments)
		if err != nil {
			resp.Error = &JSONRPCError{Code: -32603, Message: err.Error()}
			break
		}
		resp.Result = result
	default:
		resp.Error = &JSONRPCError{Code: -32601, Message: "method not found"}
	}
	return json.Marshal(resp)
}

// ReadWriteTransport implements stdio transport.
type ReadWriteTransport struct {
	In  io.Reader
	Out io.Writer
}

func (t *ReadWriteTransport) Serve(s *Server) error {
	dec := json.NewDecoder(t.In)
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		out, err := s.ProcessMessage(raw)
		if err != nil {
			continue
		}
		if _, err := t.Out.Write(out); err != nil {
			return err
		}
		if _, err := t.Out.Write([]byte("\n")); err != nil {
			return err
		}
	}
}

func (t *ReadWriteTransport) Shutdown(ctx context.Context) error {
	return nil
}

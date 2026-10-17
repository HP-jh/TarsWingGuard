package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HP-jh/TarsWingGuard/internal/knowledge"
	"github.com/HP-jh/TarsWingGuard/internal/version"
	"github.com/HP-jh/TarsWingGuard/pkg/mcp"
)

func main() {
	var (
		transport = flag.String("transport", "stdio", "Transport: stdio|sse|streamable")
		addr      = flag.String("addr", ":8081", "SSE/Streamable listen address")
		vault     = flag.String("vault", "", "Path to Obsidian vault (default: env OBSIDIAN_VAULT)")
		ver       = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *ver {
		fmt.Printf("knowledge %s (commit %s)\n", version.Version, version.Commit)
		os.Exit(0)
	}

	vpath := *vault
	if vpath == "" {
		vpath = os.Getenv("OBSIDIAN_VAULT")
	}

	kb := knowledge.New(vpath)
	// Auto-index if vault path is set
	if vpath != "" {
		result := kb.Index(knowledge.IndexRequest{})
		fmt.Fprintf(os.Stderr, "Indexed %d document(s) from %s\n", result.Indexed, vpath)
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "Index error: %s\n", e)
		}
	}

	server := mcp.NewServer("knowledge", version.Version)

	for _, t := range kb.Tools() {
		tool := mcp.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: mustJSON(t.Schema),
		}
		server.RegisterTool(tool, func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			return kb.Handle(ctx, t.Name, args)
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nShutting down knowledge...")
		server.Shutdown(ctx)
		cancel()
	}()

	var tr mcp.Transport
	switch *transport {
	case "stdio":
		tr = &mcp.ReadWriteTransport{In: os.Stdin, Out: os.Stdout}
	case "sse":
		fmt.Fprintf(os.Stderr, "SSE transport on %s (not fully implemented in v0.1.0, falling back to stdio)\n", *addr)
		tr = &mcp.ReadWriteTransport{In: os.Stdin, Out: os.Stdout}
	case "streamable":
		fmt.Fprintf(os.Stderr, "Streamable transport on %s (not fully implemented in v0.1.0, falling back to stdio)\n", *addr)
		tr = &mcp.ReadWriteTransport{In: os.Stdin, Out: os.Stdout}
	default:
		fmt.Fprintf(os.Stderr, "Unknown transport %q, using stdio\n", *transport)
		tr = &mcp.ReadWriteTransport{In: os.Stdin, Out: os.Stdout}
	}

	server.SetTransport(tr)
	if err := server.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

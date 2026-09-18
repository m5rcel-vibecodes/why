package main

import (
	"fmt"
	"os"

	"github.com/m5rcel-vibecodes/why/internal/config"
	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/matcher"
	"github.com/m5rcel-vibecodes/why/internal/mcp"
)

var (
	version = "1.2.0"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "why-mcp config warning: %v\n", err)
	}

	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "why-mcp fatal: failed to load embedded kb: %v\n", err)
		os.Exit(1)
	}

	if cfg.RulesDir != "" {
		_ = knowledgeBase.LoadCustomDir(cfg.RulesDir)
	}

	ruleMatcher := matcher.New(knowledgeBase)
	server := mcp.NewServer(knowledgeBase, ruleMatcher, version)

	if err := server.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "why-mcp server error: %v\n", err)
		os.Exit(1)
	}
}

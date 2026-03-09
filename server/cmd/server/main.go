package main

import (
	"fmt"
	"os"

	"github.com/bis-code/claude-toolkit/server/internal/toolkit"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := toolkit.NewServer()

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "claude-toolkit-server error: %v\n", err)
		os.Exit(1)
	}
}

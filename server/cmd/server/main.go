package main

import (
	"fmt"
	"os"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/toolkit"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	store, err := db.NewStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	s := toolkit.NewServer(toolkit.WithStore(store))

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "claude-toolkit-server error: %v\n", err)
		os.Exit(1)
	}
}

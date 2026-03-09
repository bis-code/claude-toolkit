package main

import (
	"fmt"
	"net"
	"os"

	"github.com/bis-code/claude-toolkit/server/internal/dashboard"
	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/patrol"
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

	// Start dashboard on a fixed port. If already taken (another session),
	// skip — the existing dashboard serves the same SQLite DB.
	const dashAddr = "127.0.0.1:19280"
	detector := patrol.NewDetector(patrol.DefaultThresholds())
	dash := dashboard.NewServer(store, detector)
	ln, err := net.Listen("tcp", dashAddr)
	if err == nil {
		go func() {
			if serveErr := dash.Serve(ln); serveErr != nil {
				fmt.Fprintf(os.Stderr, "dashboard error: %v\n", serveErr)
			}
		}()
	}
	// If port taken, dashAddr still points to the existing dashboard.
	s := toolkit.NewServer(toolkit.WithStore(store), toolkit.WithDashboardAddr(dashAddr))

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "claude-toolkit-server error: %v\n", err)
		os.Exit(1)
	}
}

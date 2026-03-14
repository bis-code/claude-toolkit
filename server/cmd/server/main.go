package main

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bis-code/claude-toolkit/server/internal/dashboard"
	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/patrol"
	"github.com/bis-code/claude-toolkit/server/internal/toolkit"
	"github.com/mark3labs/mcp-go/server"
)

const dashAddr = "127.0.0.1:19280"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "serve":
			runServer()
		case "init":
			runInit()
		case "update":
			runUpdate()
		case "status":
			runStatus()
		case "health":
			runHealth()
		case "dashboard":
			runDashboard()
		default:
			// Default: MCP server mode (backward compatible with older Claude configs).
			runServer()
		}
	} else {
		runServer()
	}
}

// runServer is the original MCP stdio server mode.
func runServer() {
	store, err := db.NewStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Start dashboard on a fixed port. If already taken (another session),
	// skip -- the existing dashboard serves the same SQLite DB.
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

// runInit initializes the toolkit in a target directory.
//
// Usage: claude-toolkit-server init [directory]
//
// This is currently a stub. The full implementation will migrate install.sh
// logic to Go once the shell-based installer is stabilised.
func runInit() {
	dir := resolveTargetDir()

	fmt.Printf("Initializing toolkit in %s...\n", dir)
	fmt.Println()
	fmt.Println("Planned steps:")
	fmt.Println("  1. Detect tech stack (Go, Node, Python, ...)")
	fmt.Println("  2. Copy rules from templates/rules/common/")
	fmt.Println("  3. Copy language-specific rules")
	fmt.Println("  4. Install git hooks")
	fmt.Println("  5. Write .claude-toolkit.json config")
	fmt.Println("  6. Write .mcp.json with server reference")
	fmt.Println()
	fmt.Println("TODO: full implementation pending install.sh migration to Go.")
}

// runUpdate updates the toolkit in a target directory.
//
// Usage: claude-toolkit-server update [directory]
//
// Stub -- will pull latest rule templates and apply non-destructive merges.
func runUpdate() {
	dir := resolveTargetDir()

	fmt.Printf("Updating toolkit in %s...\n", dir)
	fmt.Println()
	fmt.Println("Planned steps:")
	fmt.Println("  1. Read current .claude-toolkit.json version")
	fmt.Println("  2. Fetch latest rule templates")
	fmt.Println("  3. Merge rules (skip user-modified files)")
	fmt.Println("  4. Bump version in .claude-toolkit.json")
	fmt.Println()
	fmt.Println("TODO: full implementation pending install.sh migration to Go.")
}

// runStatus reports on managed projects and session counts.
//
// Usage: claude-toolkit-server status
func runStatus() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot determine home directory: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(home, ".claude-toolkit", "store.db")

	fmt.Println("claude-toolkit status")
	fmt.Println(strings.Repeat("-", 40))

	if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
		fmt.Printf("Database: not found (%s)\n", dbPath)
		fmt.Println("         Run 'claude-toolkit-server' once to initialise.")
	} else {
		sessionCount, countErr := querySessionCount(dbPath)
		if countErr != nil {
			fmt.Printf("Database: %s (error reading: %v)\n", dbPath, countErr)
		} else {
			fmt.Printf("Database: %s\n", dbPath)
			fmt.Printf("Sessions: %d recorded\n", sessionCount)
		}
	}

	fmt.Println()
	fmt.Println("Managed projects (directories containing .claude-toolkit.json):")

	projects := findManagedProjects(home)
	if len(projects) == 0 {
		fmt.Println("  none found under ~/ (searched two levels deep)")
	} else {
		for _, p := range projects {
			rel, relErr := filepath.Rel(home, p)
			if relErr != nil {
				rel = p
			}
			fmt.Printf("  ~/%s\n", rel)
		}
	}
}

// runHealth checks that the server dependencies are accessible.
//
// Usage: claude-toolkit-server health
func runHealth() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot determine home directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("claude-toolkit health check")
	fmt.Println(strings.Repeat("-", 40))

	allOK := true

	// Check 1: server binary
	self, _ := os.Executable()
	checkItem("Server binary", self != "", self)

	// Check 2: SQLite database accessible
	dbPath := filepath.Join(home, ".claude-toolkit", "store.db")
	dbExists := false
	if _, statErr := os.Stat(dbPath); statErr == nil {
		dbExists = true
	}
	checkItem("SQLite database", dbExists, dbPath)
	if !dbExists {
		allOK = false
	}

	// Check 3: dashboard port availability
	ln, listenErr := net.Listen("tcp", dashAddr)
	portFree := listenErr == nil
	if portFree {
		ln.Close()
		checkItem("Dashboard port ("+dashAddr+")", true, "available")
	} else {
		// Port taken means dashboard is already running -- that is also healthy.
		checkItem("Dashboard port ("+dashAddr+")", true, "in use (dashboard already running)")
	}

	// Check 4: Node.js available (required for hook scripts)
	nodePath, nodeErr := exec.LookPath("node")
	nodeOK := nodeErr == nil
	if nodeOK {
		checkItem("Node.js", true, nodePath)
	} else {
		checkItem("Node.js", false, "not found -- git hooks may not function")
		allOK = false
	}

	fmt.Println(strings.Repeat("-", 40))
	if allOK {
		fmt.Println("Status: healthy")
	} else {
		fmt.Println("Status: degraded (see items above)")
		os.Exit(1)
	}
}

// runDashboard starts the web dashboard and blocks until interrupted.
//
// Usage: claude-toolkit-server dashboard
func runDashboard() {
	store, err := db.NewStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	detector := patrol.NewDetector(patrol.DefaultThresholds())
	dash := dashboard.NewServer(store, detector)

	ln, err := net.Listen("tcp", dashAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot bind %s: %v\n", dashAddr, err)
		fmt.Fprintf(os.Stderr, "A dashboard may already be running at http://%s\n", dashAddr)
		os.Exit(1)
	}

	fmt.Printf("Dashboard running at http://%s\n", dashAddr)
	fmt.Println("Press Ctrl+C to stop.")

	go func() {
		if serveErr := dash.Serve(ln); serveErr != nil {
			fmt.Fprintf(os.Stderr, "dashboard error: %v\n", serveErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down dashboard.")
}

// -- Helpers ------------------------------------------------------------------

// resolveTargetDir returns os.Args[2] when present, otherwise the cwd.
func resolveTargetDir() string {
	if len(os.Args) > 2 {
		return os.Args[2]
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	return cwd
}

// querySessionCount opens the SQLite DB read-only and returns the session count.
func querySessionCount(dbPath string) (int, error) {
	d, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return 0, err
	}
	defer d.Close()

	var count int
	if err := d.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// findManagedProjects scans two directory levels below root for .claude-toolkit.json.
// The depth limit keeps the scan fast for an interactive command.
func findManagedProjects(root string) []string {
	var found []string

	topEntries, err := os.ReadDir(root)
	if err != nil {
		return found
	}

	for _, top := range topEntries {
		if !top.IsDir() || strings.HasPrefix(top.Name(), ".") {
			continue
		}
		topPath := filepath.Join(root, top.Name())

		if hasToolkitConfig(topPath) {
			found = append(found, topPath)
			continue
		}

		subEntries, readErr := os.ReadDir(topPath)
		if readErr != nil {
			continue
		}
		for _, sub := range subEntries {
			if !sub.IsDir() || strings.HasPrefix(sub.Name(), ".") {
				continue
			}
			subPath := filepath.Join(topPath, sub.Name())
			if hasToolkitConfig(subPath) {
				found = append(found, subPath)
			}
		}
	}

	return found
}

// hasToolkitConfig returns true when dir contains .claude-toolkit.json.
func hasToolkitConfig(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".claude-toolkit.json"))
	return err == nil
}

// checkItem prints a single health check result line.
func checkItem(label string, ok bool, detail string) {
	status := "OK  "
	if !ok {
		status = "FAIL"
	}
	if detail != "" {
		fmt.Printf("  [%s] %-30s %s\n", status, label, detail)
	} else {
		fmt.Printf("  [%s] %s\n", status, label)
	}
}

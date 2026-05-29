# muster Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **IMPORTANT driver:** This plan is designed to be driven by `/ralph-loop:ralph-loop`, one ralph-loop session per phase. Inside each ralph-loop iteration, use `superpowers:test-driven-development`, `superpowers:subagent-driven-development`, and `superpowers:verification-before-completion` per the phase prompts in §Phase Launch Prompts. **Never run a single ralph-loop across multiple phases** — it violates the "no mega-loops" rule (see `~/.claude/projects/…/memory/feedback_no_loops.md`).

**Goal:** Build muster v0 — a headless-first multi-agent harness for Claude Code — as a single Go binary that spawns Claude workers and coordinates them through JSONL mailbox files on disk, ships 13 superpowers-style skills, and includes first-class meeting support.

**Architecture:** Go binary embedding the Claude Agent SDK (subprocess `claude -p` initially, in-process if TS/Python SDK proves cleaner). In-process MCP server exposes mailbox/blackboard/meeting tools. Coordination primitive is JSONL mailbox files + fsnotify + self-claiming workers. Flat coordinator/worker topology (subagents cannot spawn subagents, by platform constraint). Target: ~1200 LOC Go across well-bounded packages.

**Tech Stack:**
- Go 1.22+ (toolchain)
- `github.com/mark3labs/mcp-go` — MCP server (stdio + Streamable HTTP + in-process)
- `github.com/etcd-io/bbolt` — NOT USED IN V0 (deferred to v2 upgrade)
- `github.com/fsnotify/fsnotify` — filesystem watcher
- `github.com/oklog/ulid/v2` — run/message IDs
- `github.com/santhosh-tekuri/jsonschema/v6` — JSON Schema validation
- Claude Agent SDK (via `claude -p` subprocess in v0)
- Standard library for everything else (HTTP/SSE, CLI flags via `flag`)

**Design reference:** `docs/superpowers/specs/2026-04-07-muster-design.md` (committed in `6855260`).

**Target repo:** `github.com/bis-code/muster` — **a new, separate repo** at `~/som/personal-projects/muster/`. This plan lives in claude-toolkit as the handoff document; the actual code does not.

---

## File Structure Map

muster is organized by domain, not by technical layer. Each package is small (<400 LOC) and has one responsibility.

```
muster/                                    # new repo
├── cmd/
│   └── muster/
│       └── main.go                        # CLI entry, flag parsing, subcommand dispatch
├── internal/
│   ├── spec/                              # Spec file parsing (markdown + YAML frontmatter)
│   │   ├── spec.go                        # Types: CrewSpec, Role, MailboxEdge, BlackboardKey
│   │   ├── parser.go                      # Parse .md → CrewSpec
│   │   └── parser_test.go
│   ├── manifest/                          # Run manifest JSON
│   │   ├── manifest.go                    # Types + read/write
│   │   └── manifest_test.go
│   ├── mailbox/                           # Append-only JSONL mailboxes
│   │   ├── envelope.go                    # Envelope type + schema constants
│   │   ├── store.go                       # Append, read, cursor
│   │   ├── wait.go                        # mailbox_wait impl via fsnotify
│   │   ├── watcher.go                     # fsnotify + 100ms macOS backstop
│   │   ├── store_test.go
│   │   ├── wait_test.go
│   │   └── watcher_test.go
│   ├── blackboard/                        # Shared artifact store
│   │   ├── blackboard.go                  # Put, Get, List
│   │   └── blackboard_test.go
│   ├── tasks/                             # Self-claiming task queue
│   │   ├── queue.go                       # Claim via atomic rename
│   │   └── queue_test.go
│   ├── meetings/                          # Round-based meeting loop
│   │   ├── types.go                       # MeetingSpec, Turn, State, Decision
│   │   ├── orchestrator.go                # Round loop, speaker selection, termination
│   │   ├── tools.go                       # MCP tool handlers
│   │   ├── turn_token.go                  # One-shot capability
│   │   └── orchestrator_test.go
│   ├── orchestrator/                      # Run lifecycle, spawn
│   │   ├── run.go                         # CreateRun, lifecycle state transitions
│   │   ├── spawn.go                       # Spawn workers via Agent SDK (iface + impl)
│   │   ├── preflight.go                   # Git clean, contracts green, no-active-run checks
│   │   └── run_test.go
│   ├── identity/                          # canUseTool + per-worker tokens
│   │   ├── token.go                       # Mint, verify, rotate
│   │   ├── hook.go                        # SubagentStart hook payload writer
│   │   └── token_test.go
│   ├── mcp/                               # MCP server wiring (mark3labs/mcp-go)
│   │   ├── server.go                      # createSdkMcpServer equivalent
│   │   ├── tools_mailbox.go               # mailbox_* tool registration
│   │   ├── tools_blackboard.go
│   │   ├── tools_tasks.go
│   │   ├── tools_meetings.go
│   │   └── tools_roster.go
│   ├── sse/                               # Observability HTTP
│   │   ├── server.go                      # GET /events stream
│   │   └── server_test.go
│   └── cli/                               # Subcommand implementations (one file per verb)
│       ├── run.go
│       ├── status.go
│       ├── tail.go
│       ├── inspect.go
│       ├── finish.go
│       ├── meeting.go
│       ├── init.go
│       └── version.go
├── skills/                                # 13 SKILL.md files (copied from design spec)
│   ├── using-muster/SKILL.md
│   ├── brainstorming-crew/SKILL.md
│   ├── ...
│   └── writing-muster-skill/SKILL.md
├── testdata/
│   ├── hello.md                           # minimal spec used in Phase 1
│   └── two-worker.md                      # two-worker spec used in Phase 2
├── docs/
│   ├── architecture.md                    # link to claude-toolkit's design spec commit
│   └── skill-tests/                       # pressure-test outputs
├── .goreleaser.yaml                       # Phase 4
├── .gitignore
├── go.mod
├── go.sum
├── README.md
└── LICENSE                                # MIT
```

---

## Phase Launch Prompts (for `/ralph-loop:ralph-loop`)

Each phase is **one ralph-loop session**. Before launching, commit your current state and open a fresh session. Copy the phase prompt below verbatim.

### Phase 1 prompt

```
/ralph-loop:ralph-loop "Implement Phase 1 of muster per
docs/superpowers/plans/2026-04-07-muster-implementation.md §Phase 1.

Scope (Phase 1 only):
- Create new Go repo at ~/som/personal-projects/muster
- Packages: internal/{spec,manifest,orchestrator}, cmd/muster
- Spec parser (markdown + YAML frontmatter → CrewSpec)
- Manifest read/write
- `muster run <spec.md>` creates .muster/runs/<ulid>/ with manifest.json
- `muster run` spawns ONE coordinator worker via a mocked spawn interface
- `muster version` prints version
- No mailbox/blackboard/meeting primitives yet — those are Phase 2/3

Process:
1. superpowers:test-driven-development for every file. Red-green-refactor.
   No production code without a failing test first.
2. superpowers:subagent-driven-development for parallel file groups
   (spec package vs manifest package vs orchestrator package are independent).
3. After each subagent returns, run `go build ./... && go test ./...` and
   fix failures before the next iteration.
4. superpowers:verification-before-completion before emitting the promise.

Definition of done for Phase 1 (all must pass in the same iteration):
- `go build ./...` exits 0
- `go test ./...` exits 0
- `go vet ./...` exits 0
- `./muster version` prints a version string
- `./muster run testdata/hello.md` creates .muster/runs/<ulid>/manifest.json
  with status 'active' and roster containing one coordinator
- `.muster/runs/latest` symlink points at the new run

When ALL gates pass in a single iteration, output exactly:
<promise>PHASE1_DONE</promise>

Do not emit the promise early. If stuck, note blockers in
.muster/blockers.md and keep iterating. Work from the task list in
§Phase 1 in order."
--max-iterations 30 --completion-promise "PHASE1_DONE"
```

### Phase 2 prompt

```
/ralph-loop "Implement Phase 2 of muster per
docs/superpowers/plans/2026-04-07-muster-implementation.md §Phase 2.

Scope (Phase 2 only):
- internal/mailbox (envelope, store, wait, watcher with fsnotify + 100ms backstop)
- internal/blackboard
- internal/tasks (task queue via atomic rename)
- internal/identity (per-worker token + canUseTool callback)
- internal/mcp (mark3labs/mcp-go server registration, tools_mailbox, tools_blackboard, tools_tasks, tools_roster)
- Real worker spawn via Claude Agent SDK (subprocess `claude -p` in v0)
- Multi-worker spawn from spec
- SubagentStart hook script that injects agent_id and token via additionalContext

Process: TDD + subagent-driven + verification-before-completion as Phase 1.

Definition of done for Phase 2:
- All Phase 1 gates still pass
- Integration test: 3-worker hello-world crew where workers exchange
  messages via mailboxes, finishes in <60s
- Contract tests pass for all envelope kinds
- Identity spoofing test passes (worker cannot set from= to another agent's id)
- `mailbox_wait` wakes within 500ms of a message append on both Linux and macOS

When ALL gates pass: <promise>PHASE2_DONE</promise>"
--max-iterations 30 --completion-promise "PHASE2_DONE"
```

### Phase 3 prompt

```
/ralph-loop "Implement Phase 3 of muster per
docs/superpowers/plans/2026-04-07-muster-implementation.md §Phase 3.

Scope (Phase 3 only):
- CLI subcommands: muster status / tail / inspect / finish [--discard]
- internal/sse observability endpoint (GET /events)
- internal/meetings full package:
  - MeetingOpen / Close / Adjourn / Speak / Transcript / State
  - Round-based coordinator loop (Go state machine)
  - round_robin / moderator / priority strategies
  - Turn tokens (one-shot capability)
  - Failure handling (timeout, retry once, skip)
  - decision.json schema validation via santhosh-tekuri/jsonschema
- internal/mcp/tools_meetings tool registration
- `muster meeting open/tail/close` CLI subcommand group

Process: TDD + subagent-driven + verification-before-completion.

Definition of done for Phase 3:
- All previous gates pass
- End-to-end test: spawn a 3-worker crew that opens a meeting, reaches a
  decision.json that validates against a decision_shape, and finishes via
  `muster finish`
- SSE endpoint streams all envelopes and meeting events for a full run
  without loss (verified by counting)
- Meeting with a deliberate TurnTimeout causes the orchestrator to skip
  the stalled speaker and advance

When ALL gates pass: <promise>PHASE3_DONE</promise>"
--max-iterations 30 --completion-promise "PHASE3_DONE"
```

### Phase 4 prompt

```
/ralph-loop "Implement Phase 4 of muster per
docs/superpowers/plans/2026-04-07-muster-implementation.md §Phase 4.

Scope (Phase 4 only):
- Copy all 13 SKILL.md files from
  ../claude-toolkit/docs/superpowers/specs/2026-04-07-muster-skills/
  to ./skills/<slug>/SKILL.md
- For every rigid skill, run muster:writing-muster-skill pressure test:
  dispatch a fresh subagent with the skill + a pressure-test prompt,
  capture verdict to docs/skill-tests/<slug>.md, iterate until HONORED
- README.md with quickstart, CLI reference, link to design spec
- `muster init` subcommand scaffolds .muster/ in a new project
- .goreleaser.yaml cross-platform binaries
- Final polish: `go vet ./... && golangci-lint run` clean

Process: TDD + subagent-driven + verification-before-completion.

Definition of done for Phase 4:
- All previous gates pass
- Every rigid skill has a HONORED pressure-test file at docs/skill-tests/<slug>.md
- `muster init && muster run testdata/hello.md` works on a clean clone
- `go vet ./... && golangci-lint run` clean
- goreleaser --snapshot --clean builds binaries for darwin/amd64,
  darwin/arm64, linux/amd64, linux/arm64

When ALL gates pass: <promise>PHASE4_DONE</promise>"
--max-iterations 30 --completion-promise "PHASE4_DONE"
```

---

## Phase 1 — Skeleton + `muster run`

**Deliverables:** a `muster` binary that parses a spec file, creates a run directory with a manifest, and spawns one coordinator worker via a mocked spawn interface. Real spawning comes in Phase 2.

### Task 1.1: Initialize the muster repository

**Files:**
- Create: `~/som/personal-projects/muster/` (new git repo)
- Create: `~/som/personal-projects/muster/go.mod`
- Create: `~/som/personal-projects/muster/LICENSE`
- Create: `~/som/personal-projects/muster/README.md`
- Create: `~/som/personal-projects/muster/.gitignore`

- [ ] **Step 1: Create the directory and initialize git**

Run:
```bash
mkdir -p ~/som/personal-projects/muster
cd ~/som/personal-projects/muster
git init
git config user.name "Ioan-Sorin Baicoianu"
git config user.email "baicoianuioansorin@gmail.com"
```

- [ ] **Step 2: Initialize the Go module**

Run:
```bash
go mod init github.com/bis-code/muster
```

Expected: `go.mod` created with `module github.com/bis-code/muster` and a `go` directive.

- [ ] **Step 3: Write the MIT LICENSE file**

Create `LICENSE`:
```
MIT License

Copyright (c) 2026 Ioan-Sorin Baicoianu

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 4: Write a one-paragraph README stub**

Create `README.md`:
```markdown
# muster

*call up the agents, ship the work.*

muster is a headless-first multi-agent harness for Claude Code. A Go
orchestrator spawns Claude subagents as workers, and they coordinate
through JSONL mailbox files on disk.

**Status:** v0 in development. See design spec at
`<claude-toolkit>/docs/superpowers/specs/2026-04-07-muster-design.md`.
```

- [ ] **Step 5: Write the `.gitignore`**

Create `.gitignore`:
```
# Binaries
/muster
/bin/
dist/

# Test artifacts
coverage.out
*.test

# Run state
.muster/runs/
.muster/archive/

# IDE
.vscode/
.idea/

# OS
.DS_Store
```

- [ ] **Step 6: Initial commit**

Run:
```bash
cd ~/som/personal-projects/muster
git add go.mod LICENSE README.md .gitignore
git commit -m "chore: initialize muster repository"
```

### Task 1.2: Define spec types and write the first failing parser test

**Files:**
- Create: `internal/spec/spec.go`
- Create: `internal/spec/parser.go`
- Create: `internal/spec/parser_test.go`
- Create: `testdata/hello.md`

- [ ] **Step 1: Write a minimal hello spec fixture**

Create `testdata/hello.md`:
```markdown
---
goal: "Say hello from a muster crew."
termination: "coordinator writes blackboard key 'greeting'"
---

# Hello Crew

## Roles
| Role | Count | Responsibility |
|------|------:|----------------|
| coordinator | 1 | Write the greeting to the blackboard |

## Mailbox Edges
(none in Phase 1)

## Blackboard Keys
| Key | Owner | Readers | Schema |
|-----|-------|---------|--------|
| `greeting` | coordinator | external | none |

## Acceptance Criteria
- [ ] Blackboard key `greeting` is set to "hello from muster"
```

- [ ] **Step 2: Create the spec type file (empty stub)**

Create `internal/spec/spec.go`:
```go
// Package spec parses muster crew spec files (markdown + YAML frontmatter).
package spec

// CrewSpec is the in-memory representation of a .muster/specs/<slug>.md file.
// Populated incrementally across Tasks 1.2–1.4.
type CrewSpec struct {
	Goal        string
	Termination string
	Roles       []Role
}

// Role describes one worker type in the crew.
type Role struct {
	Name           string
	Count          int
	Responsibility string
}
```

- [ ] **Step 3: Write the failing parser test**

Create `internal/spec/parser_test.go`:
```go
package spec

import (
	"path/filepath"
	"testing"
)

func TestParse_HelloSpec(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "hello.md")
	got, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%q) returned error: %v", path, err)
	}
	if got.Goal != "Say hello from a muster crew." {
		t.Errorf("Goal = %q, want %q", got.Goal, "Say hello from a muster crew.")
	}
	if len(got.Roles) != 1 {
		t.Fatalf("len(Roles) = %d, want 1", len(got.Roles))
	}
	if got.Roles[0].Name != "coordinator" {
		t.Errorf("Roles[0].Name = %q, want %q", got.Roles[0].Name, "coordinator")
	}
	if got.Roles[0].Count != 1 {
		t.Errorf("Roles[0].Count = %d, want 1", got.Roles[0].Count)
	}
}
```

- [ ] **Step 4: Run the test and confirm it fails**

Run:
```bash
go test ./internal/spec/...
```

Expected: compile error — `ParseFile` is not defined yet. This is the RED step.

- [ ] **Step 5: Commit the failing test**

Run:
```bash
git add internal/spec/spec.go internal/spec/parser_test.go testdata/hello.md
git commit -m "test(spec): add failing test for hello spec parse"
```

### Task 1.3: Implement `ParseFile` to make Task 1.2's test pass

**Files:**
- Create: `internal/spec/parser.go`

- [ ] **Step 1: Add a YAML frontmatter dependency**

Run:
```bash
go get gopkg.in/yaml.v3
```

- [ ] **Step 2: Write the minimal parser implementation**

Create `internal/spec/parser.go`:
```go
package spec

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontmatterDelim separates the YAML frontmatter from the markdown body.
const frontmatterDelim = "---"

// ParseFile reads a muster crew spec file from disk and returns a CrewSpec.
func ParseFile(path string) (*CrewSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec file: %w", err)
	}
	return Parse(data)
}

// Parse parses a muster crew spec from a byte slice.
func Parse(data []byte) (*CrewSpec, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	var header struct {
		Goal        string `yaml:"goal"`
		Termination string `yaml:"termination"`
	}
	if err := yaml.Unmarshal(fm, &header); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	roles, err := parseRolesTable(body)
	if err != nil {
		return nil, fmt.Errorf("parse roles table: %w", err)
	}

	return &CrewSpec{
		Goal:        header.Goal,
		Termination: header.Termination,
		Roles:       roles,
	}, nil
}

func splitFrontmatter(data []byte) (frontmatter, body []byte, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1<<16), 1<<20)

	var (
		inFM       bool
		fmLines    [][]byte
		bodyLines  [][]byte
		seenOpen   bool
	)

	for scanner.Scan() {
		line := scanner.Bytes()
		dup := append([]byte(nil), line...)

		if !seenOpen && bytes.Equal(bytes.TrimSpace(dup), []byte(frontmatterDelim)) {
			seenOpen = true
			inFM = true
			continue
		}
		if inFM && bytes.Equal(bytes.TrimSpace(dup), []byte(frontmatterDelim)) {
			inFM = false
			continue
		}
		if inFM {
			fmLines = append(fmLines, dup)
		} else if seenOpen {
			bodyLines = append(bodyLines, dup)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	if !seenOpen {
		return nil, nil, fmt.Errorf("spec file missing YAML frontmatter")
	}
	return bytes.Join(fmLines, []byte("\n")), bytes.Join(bodyLines, []byte("\n")), nil
}

// rolesHeader matches the H2 heading that precedes the roles table.
var rolesHeader = regexp.MustCompile(`(?m)^##\s+Roles\s*$`)

// roleRow matches a pipe-delimited row like:
//   | coordinator | 1 | Write the greeting to the blackboard |
var roleRow = regexp.MustCompile(`^\|\s*([a-zA-Z][\w-]*)\s*\|\s*(\d+)\s*\|\s*(.+?)\s*\|\s*$`)

func parseRolesTable(body []byte) ([]Role, error) {
	loc := rolesHeader.FindIndex(body)
	if loc == nil {
		return nil, nil
	}
	rest := body[loc[1]:]
	lines := strings.Split(string(rest), "\n")

	var roles []Role
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			if len(roles) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "##") {
			break
		}
		if strings.HasPrefix(line, "|---") || strings.Contains(line, "| Role ") || strings.Contains(line, "|------") {
			continue
		}
		m := roleRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		count, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("invalid role count %q: %w", m[2], err)
		}
		roles = append(roles, Role{
			Name:           m[1],
			Count:          count,
			Responsibility: m[3],
		})
	}
	return roles, nil
}
```

- [ ] **Step 3: Run the test and confirm it passes**

Run:
```bash
go test ./internal/spec/...
```

Expected: `PASS`.

- [ ] **Step 4: Run `go vet` and `go build`**

Run:
```bash
go vet ./...
go build ./...
```

Expected: no output (success).

- [ ] **Step 5: Commit the implementation**

Run:
```bash
git add internal/spec/parser.go go.mod go.sum
git commit -m "feat(spec): implement ParseFile for minimal hello spec"
```

### Task 1.4: Extend spec types to cover mailbox edges, blackboard, and acceptance criteria

**Files:**
- Modify: `internal/spec/spec.go`
- Modify: `internal/spec/parser.go`
- Modify: `internal/spec/parser_test.go`
- Modify: `testdata/hello.md` (no change needed; but add `testdata/two-worker.md`)
- Create: `testdata/two-worker.md`

- [ ] **Step 1: Write the expanded two-worker fixture**

Create `testdata/two-worker.md`:
```markdown
---
goal: "Producer-consumer crew writing results to the blackboard."
termination: "coordinator writes blackboard key 'result'"
---

# Two Worker Crew

## Roles
| Role | Count | Responsibility |
|------|------:|----------------|
| coordinator | 1 | Orchestrate producer and consumer, write result |
| producer | 1 | Emit task.assign messages |
| consumer | 1 | Read task.assign, emit task.result |

## Mailbox Edges
| From → To | Message Type | Max Depth | Notes |
|-----------|--------------|----------:|-------|
| producer → consumer | task.assign | 50 | fan-out |
| consumer → coordinator | task.result | 50 | fan-in |

## Blackboard Keys
| Key | Owner | Readers | Schema |
|-----|-------|---------|--------|
| `result` | coordinator | external | none |

## Acceptance Criteria
- [ ] Blackboard key `result` is set
- [ ] `consumer → coordinator` mailbox drained at end
```

- [ ] **Step 2: Extend `CrewSpec` with new fields**

Edit `internal/spec/spec.go` — replace the file with:
```go
// Package spec parses muster crew spec files (markdown + YAML frontmatter).
package spec

// CrewSpec is the in-memory representation of a .muster/specs/<slug>.md file.
type CrewSpec struct {
	Goal           string
	Termination    string
	Roles          []Role
	Mailboxes      []MailboxEdge
	BlackboardKeys []BlackboardKey
	Acceptance     []string
}

// Role describes one worker type in the crew.
type Role struct {
	Name           string
	Count          int
	Responsibility string
}

// MailboxEdge describes a directed message channel between roles.
type MailboxEdge struct {
	From        string
	To          string
	MessageKind string
	MaxDepth    int
	Notes       string
}

// BlackboardKey describes a shared artifact slot.
type BlackboardKey struct {
	Key     string
	Owner   string
	Readers []string
	Schema  string
}
```

- [ ] **Step 3: Write the failing test for the two-worker fixture**

Add to `internal/spec/parser_test.go`:
```go
func TestParse_TwoWorkerSpec(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "two-worker.md")
	got, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%q) returned error: %v", path, err)
	}
	if len(got.Roles) != 3 {
		t.Errorf("len(Roles) = %d, want 3", len(got.Roles))
	}
	if len(got.Mailboxes) != 2 {
		t.Fatalf("len(Mailboxes) = %d, want 2", len(got.Mailboxes))
	}
	if got.Mailboxes[0].From != "producer" || got.Mailboxes[0].To != "consumer" {
		t.Errorf("Mailboxes[0] = %+v, want producer→consumer", got.Mailboxes[0])
	}
	if got.Mailboxes[0].MaxDepth != 50 {
		t.Errorf("Mailboxes[0].MaxDepth = %d, want 50", got.Mailboxes[0].MaxDepth)
	}
	if len(got.BlackboardKeys) != 1 || got.BlackboardKeys[0].Key != "result" {
		t.Errorf("BlackboardKeys = %+v, want one 'result' key", got.BlackboardKeys)
	}
	if len(got.Acceptance) != 2 {
		t.Errorf("len(Acceptance) = %d, want 2", len(got.Acceptance))
	}
}
```

- [ ] **Step 4: Run the test and confirm it fails**

Run:
```bash
go test ./internal/spec/...
```

Expected: FAIL — new fields are zero-valued because the parser doesn't fill them yet.

- [ ] **Step 5: Extend the parser to fill the new fields**

Add to `internal/spec/parser.go`:
```go
// Append inside Parse, just before the `return &CrewSpec{...}` line:
//   mailboxes, err := parseMailboxTable(body)
//   if err != nil { return nil, fmt.Errorf("parse mailbox table: %w", err) }
//   blackboard, err := parseBlackboardTable(body)
//   if err != nil { return nil, fmt.Errorf("parse blackboard table: %w", err) }
//   acceptance := parseAcceptanceList(body)
//
// And populate the returned CrewSpec with those fields.

var (
	mailboxHeader    = regexp.MustCompile(`(?m)^##\s+Mailbox Edges\s*$`)
	blackboardHeader = regexp.MustCompile(`(?m)^##\s+Blackboard Keys\s*$`)
	acceptanceHeader = regexp.MustCompile(`(?m)^##\s+Acceptance Criteria\s*$`)

	mailboxRow = regexp.MustCompile(
		`^\|\s*([a-zA-Z][\w-]*)\s*→\s*([a-zA-Z][\w-]*)\s*\|\s*([a-zA-Z][\w.]*)\s*\|\s*(\d+)\s*\|\s*(.*?)\s*\|\s*$`,
	)
	blackboardRow = regexp.MustCompile(
		"^\\|\\s*`([a-zA-Z][\\w.-]*)`\\s*\\|\\s*([a-zA-Z][\\w-]*)\\s*\\|\\s*(.+?)\\s*\\|\\s*(.+?)\\s*\\|\\s*$",
	)
	acceptanceRow = regexp.MustCompile(`^-\s*\[[ xX]\]\s+(.+)$`)
)

func sectionLines(body []byte, header *regexp.Regexp) []string {
	loc := header.FindIndex(body)
	if loc == nil {
		return nil
	}
	rest := body[loc[1]:]
	out := []string{}
	for _, raw := range strings.Split(string(rest), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "##") {
			break
		}
		out = append(out, line)
	}
	return out
}

func parseMailboxTable(body []byte) ([]MailboxEdge, error) {
	var edges []MailboxEdge
	for _, line := range sectionLines(body, mailboxHeader) {
		if strings.HasPrefix(line, "|---") || strings.Contains(line, "| From") {
			continue
		}
		m := mailboxRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		depth, err := strconv.Atoi(m[4])
		if err != nil {
			return nil, fmt.Errorf("invalid max depth %q: %w", m[4], err)
		}
		edges = append(edges, MailboxEdge{
			From:        m[1],
			To:          m[2],
			MessageKind: m[3],
			MaxDepth:    depth,
			Notes:       m[5],
		})
	}
	return edges, nil
}

func parseBlackboardTable(body []byte) ([]BlackboardKey, error) {
	var keys []BlackboardKey
	for _, line := range sectionLines(body, blackboardHeader) {
		if strings.HasPrefix(line, "|---") || strings.Contains(line, "| Key") {
			continue
		}
		m := blackboardRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		readers := strings.Split(m[3], ",")
		for i := range readers {
			readers[i] = strings.TrimSpace(readers[i])
		}
		keys = append(keys, BlackboardKey{
			Key:     m[1],
			Owner:   m[2],
			Readers: readers,
			Schema:  m[4],
		})
	}
	return keys, nil
}

func parseAcceptanceList(body []byte) []string {
	var out []string
	for _, line := range sectionLines(body, acceptanceHeader) {
		m := acceptanceRow.FindStringSubmatch(line)
		if m != nil {
			out = append(out, m[1])
		}
	}
	return out
}
```

Also edit `Parse` to call the three new helpers and populate the returned `CrewSpec` fields.

- [ ] **Step 6: Run tests — both fixtures must pass**

Run:
```bash
go test ./internal/spec/... -v
```

Expected: `TestParse_HelloSpec` and `TestParse_TwoWorkerSpec` both PASS.

- [ ] **Step 7: Commit**

Run:
```bash
git add internal/spec/ testdata/two-worker.md
git commit -m "feat(spec): parse mailbox, blackboard, and acceptance sections"
```

### Task 1.5: Manifest type + JSON round-trip test

**Files:**
- Create: `internal/manifest/manifest.go`
- Create: `internal/manifest/manifest_test.go`

- [ ] **Step 1: Write the failing manifest round-trip test**

Create `internal/manifest/manifest_test.go`:
```go
package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteRead_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "manifest.json")

	m := &Manifest{
		RunID:     "run_01HX000000000000000000",
		SpecPath:  ".muster/specs/hello/hello.md",
		Status:    StatusActive,
		StartedAt: time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC),
		Roster: []Agent{
			{
				AgentID:   "coordinator-01",
				Role:      "coordinator",
				Worktree:  "/tmp/wt-coord",
				PID:       0,
				Status:    AgentAlive,
				StartedAt: time.Date(2026, 4, 7, 12, 0, 1, 0, time.UTC),
			},
		},
		Termination: TerminationSpec{
			Condition: "coordinator writes blackboard key 'greeting'",
		},
	}

	if err := Write(path, m); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.RunID != m.RunID {
		t.Errorf("RunID = %q, want %q", got.RunID, m.RunID)
	}
	if got.Status != StatusActive {
		t.Errorf("Status = %q, want %q", got.Status, StatusActive)
	}
	if len(got.Roster) != 1 || got.Roster[0].AgentID != "coordinator-01" {
		t.Errorf("Roster = %+v", got.Roster)
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run:
```bash
go test ./internal/manifest/...
```

Expected: compile error.

- [ ] **Step 3: Implement the manifest package**

Create `internal/manifest/manifest.go`:
```go
// Package manifest defines the JSON structure written to
// .muster/runs/<run-id>/manifest.json.
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Status is the high-level state of a run.
type Status string

const (
	StatusActive    Status = "active"
	StatusFinished  Status = "finished"
	StatusFailed    Status = "failed"
	StatusDiscarded Status = "discarded"
)

// AgentStatus is the per-agent liveness indicator.
type AgentStatus string

const (
	AgentAlive AgentStatus = "alive"
	AgentDone  AgentStatus = "done"
	AgentDead  AgentStatus = "dead"
)

// Manifest is the single source of truth for a muster run.
type Manifest struct {
	RunID       string          `json:"run_id"`
	SpecPath    string          `json:"spec_path"`
	Status      Status          `json:"status"`
	StartedAt   time.Time       `json:"started_at"`
	FinishedAt  *time.Time      `json:"finished_at,omitempty"`
	Roster      []Agent         `json:"roster"`
	Meetings    []string        `json:"meetings,omitempty"`
	Termination TerminationSpec `json:"termination"`
}

// Agent is one worker in the run's roster.
type Agent struct {
	AgentID   string      `json:"agent_id"`
	Role      string      `json:"role"`
	Worktree  string      `json:"worktree"`
	PID       int         `json:"pid"`
	Status    AgentStatus `json:"status"`
	StartedAt time.Time   `json:"started_at"`
	ExitCode  *int        `json:"exit_code,omitempty"`
}

// TerminationSpec describes when the orchestrator should consider the run done.
type TerminationSpec struct {
	Condition string `json:"condition"`
}

// Write serializes a manifest to the given path as pretty JSON.
func Write(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// Read parses a manifest from the given path.
func Read(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	return &m, nil
}
```

- [ ] **Step 4: Run the test**

Run:
```bash
go test ./internal/manifest/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:
```bash
git add internal/manifest/
git commit -m "feat(manifest): add JSON round-trip for run manifest"
```

### Task 1.6: Orchestrator `CreateRun` — run dir + manifest + latest symlink

**Files:**
- Create: `internal/orchestrator/run.go`
- Create: `internal/orchestrator/run_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/orchestrator/run_test.go`:
```go
package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bis-code/muster/internal/manifest"
	"github.com/bis-code/muster/internal/spec"
)

func TestCreateRun_WritesManifestAndSymlink(t *testing.T) {
	tmp := t.TempDir()
	s := &spec.CrewSpec{
		Goal:        "test run",
		Termination: "none",
		Roles: []spec.Role{
			{Name: "coordinator", Count: 1, Responsibility: "test"},
		},
	}

	run, err := CreateRun(tmp, "testdata/fake.md", s)
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if run.ID == "" {
		t.Error("run.ID is empty")
	}

	mfPath := filepath.Join(tmp, "runs", run.ID, "manifest.json")
	m, err := manifest.Read(mfPath)
	if err != nil {
		t.Fatalf("read manifest at %q: %v", mfPath, err)
	}
	if m.Status != manifest.StatusActive {
		t.Errorf("Status = %q, want %q", m.Status, manifest.StatusActive)
	}
	if len(m.Roster) != 1 || m.Roster[0].Role != "coordinator" {
		t.Errorf("Roster = %+v, want one coordinator", m.Roster)
	}

	latest := filepath.Join(tmp, "runs", "latest")
	target, err := os.Readlink(latest)
	if err != nil {
		t.Fatalf("readlink latest: %v", err)
	}
	if target != run.ID {
		t.Errorf("latest -> %q, want %q", target, run.ID)
	}
}
```

- [ ] **Step 2: Run and confirm failure**

Run:
```bash
go test ./internal/orchestrator/...
```

Expected: compile error.

- [ ] **Step 3: Implement `CreateRun`**

Create `internal/orchestrator/run.go`:
```go
// Package orchestrator owns the run lifecycle.
package orchestrator

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/bis-code/muster/internal/manifest"
	"github.com/bis-code/muster/internal/spec"
)

// Run is a live reference to a run directory on disk.
type Run struct {
	ID       string
	Root     string
	MustDir  string
	Manifest *manifest.Manifest
}

// CreateRun materializes a new run under <mustDirRoot>/runs/<run-id>/.
// It writes the manifest, builds the roster from the spec, and points
// <mustDirRoot>/runs/latest at the new run via a symlink.
func CreateRun(mustDirRoot, specPath string, s *spec.CrewSpec) (*Run, error) {
	now := time.Now().UTC()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(now.UnixNano())), 0)
	id := "run_" + ulid.MustNew(ulid.Timestamp(now), entropy).String()

	runDir := filepath.Join(mustDirRoot, "runs", id)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("create run dir: %w", err)
	}
	for _, sub := range []string{"mailboxes", "blackboard", "tasks", "transcripts", "meetings", "debug"} {
		if err := os.MkdirAll(filepath.Join(runDir, sub), 0o755); err != nil {
			return nil, fmt.Errorf("create %s dir: %w", sub, err)
		}
	}

	roster := buildRoster(s, now)
	m := &manifest.Manifest{
		RunID:     id,
		SpecPath:  specPath,
		Status:    manifest.StatusActive,
		StartedAt: now,
		Roster:    roster,
		Termination: manifest.TerminationSpec{
			Condition: s.Termination,
		},
	}
	if err := manifest.Write(filepath.Join(runDir, "manifest.json"), m); err != nil {
		return nil, err
	}

	if err := updateLatestSymlink(mustDirRoot, id); err != nil {
		return nil, fmt.Errorf("update latest symlink: %w", err)
	}

	return &Run{
		ID:       id,
		Root:     runDir,
		MustDir:  mustDirRoot,
		Manifest: m,
	}, nil
}

func buildRoster(s *spec.CrewSpec, startedAt time.Time) []manifest.Agent {
	var out []manifest.Agent
	for _, role := range s.Roles {
		for i := 1; i <= role.Count; i++ {
			out = append(out, manifest.Agent{
				AgentID:   fmt.Sprintf("%s-%02d", role.Name, i),
				Role:      role.Name,
				Status:    manifest.AgentAlive,
				StartedAt: startedAt,
			})
		}
	}
	return out
}

// updateLatestSymlink atomically re-points <root>/runs/latest at <id>.
func updateLatestSymlink(mustDirRoot, id string) error {
	runsDir := filepath.Join(mustDirRoot, "runs")
	linkPath := filepath.Join(runsDir, "latest")
	tmpPath := filepath.Join(runsDir, ".latest.tmp")

	_ = os.Remove(tmpPath)
	if err := os.Symlink(id, tmpPath); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, linkPath); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Add dependencies**

Run:
```bash
go get github.com/oklog/ulid/v2
```

- [ ] **Step 5: Run the test**

Run:
```bash
go test ./internal/orchestrator/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:
```bash
git add internal/orchestrator/ go.mod go.sum
git commit -m "feat(orchestrator): CreateRun writes manifest and latest symlink"
```

### Task 1.7: Spawn interface + mock implementation + coordinator spawn

**Files:**
- Create: `internal/orchestrator/spawn.go`
- Modify: `internal/orchestrator/run_test.go` (add spawn test)

- [ ] **Step 1: Write the failing spawn test**

Append to `internal/orchestrator/run_test.go`:
```go
func TestSpawnCoordinator_Mock(t *testing.T) {
	tmp := t.TempDir()
	s := &spec.CrewSpec{
		Goal: "spawn test",
		Roles: []spec.Role{
			{Name: "coordinator", Count: 1, Responsibility: "test"},
		},
	}
	run, err := CreateRun(tmp, "testdata/fake.md", s)
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	mock := &MockSpawner{Transcripts: map[string]string{
		"coordinator-01": "hello from coordinator-01\n",
	}}

	if err := run.SpawnAll(mock); err != nil {
		t.Fatalf("SpawnAll: %v", err)
	}
	if mock.Spawned != 1 {
		t.Errorf("Spawned = %d, want 1", mock.Spawned)
	}

	tPath := filepath.Join(run.Root, "transcripts", "coordinator-01.jsonl")
	data, err := os.ReadFile(tPath)
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	if string(data) != "hello from coordinator-01\n" {
		t.Errorf("transcript = %q", string(data))
	}
}
```

- [ ] **Step 2: Run and confirm failure**

Run:
```bash
go test ./internal/orchestrator/...
```

Expected: compile error.

- [ ] **Step 3: Implement the spawn interface and mock**

Create `internal/orchestrator/spawn.go`:
```go
package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bis-code/muster/internal/manifest"
)

// Spawner launches workers for a run.
// Phase 1 ships only MockSpawner; Phase 2 adds ClaudeSDKSpawner.
type Spawner interface {
	// Spawn launches one worker for the given agent and returns once the worker
	// exits. In v0 the transcript is captured as a side-effect on disk.
	Spawn(run *Run, agent manifest.Agent) error
}

// MockSpawner is a deterministic fake used in Phase 1 tests and the
// end-to-end smoke test before the real Claude SDK is wired in Phase 2.
type MockSpawner struct {
	Transcripts map[string]string // agent_id -> raw transcript to write
	Spawned     int               // counter for assertions
}

// Spawn writes the pre-canned transcript for the agent and returns.
func (m *MockSpawner) Spawn(run *Run, agent manifest.Agent) error {
	m.Spawned++
	out, ok := m.Transcripts[agent.AgentID]
	if !ok {
		out = fmt.Sprintf("mock transcript for %s\n", agent.AgentID)
	}
	path := filepath.Join(run.Root, "transcripts", agent.AgentID+".jsonl")
	return os.WriteFile(path, []byte(out), 0o644)
}

// SpawnAll spawns every agent in the run's roster sequentially.
// Phase 2 will add parallelism and error handling.
func (r *Run) SpawnAll(sp Spawner) error {
	for _, agent := range r.Manifest.Roster {
		if err := sp.Spawn(r, agent); err != nil {
			return fmt.Errorf("spawn %s: %w", agent.AgentID, err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the test**

Run:
```bash
go test ./internal/orchestrator/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:
```bash
git add internal/orchestrator/
git commit -m "feat(orchestrator): add Spawner iface and MockSpawner for phase 1"
```

### Task 1.8: `muster run` CLI subcommand — end to end

**Files:**
- Create: `internal/cli/run.go`
- Create: `internal/cli/version.go`
- Create: `cmd/muster/main.go`
- Create: `internal/cli/run_integration_test.go`

- [ ] **Step 1: Write an integration test that drives the binary**

Create `internal/cli/run_integration_test.go`:
```go
package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMusterRun_EndToEnd builds the binary and runs it against testdata/hello.md.
// It asserts that a run directory with a manifest is produced.
func TestMusterRun_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skip in short mode")
	}
	repo := findRepoRoot(t)
	bin := filepath.Join(t.TempDir(), "muster")

	build := exec.Command("go", "build", "-o", bin, "./cmd/muster")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	workspace := t.TempDir()
	cmd := exec.Command(bin, "run", filepath.Join(repo, "testdata", "hello.md"))
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), "MUSTER_SPAWNER=mock")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("muster run failed: %v\n%s", err, out)
	}

	latest, err := os.Readlink(filepath.Join(workspace, ".muster", "runs", "latest"))
	if err != nil {
		t.Fatalf("latest symlink missing: %v", err)
	}
	mf := filepath.Join(workspace, ".muster", "runs", latest, "manifest.json")
	if _, err := os.Stat(mf); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	t.Fatal("could not find repo root")
	return ""
}

// sanity check that go.mod is findable from cli package
var _ = strings.TrimSpace
```

- [ ] **Step 2: Run the test and confirm failure**

Run:
```bash
go test ./internal/cli/...
```

Expected: build error — `cmd/muster` doesn't exist.

- [ ] **Step 3: Implement the version subcommand**

Create `internal/cli/version.go`:
```go
package cli

import (
	"fmt"
	"io"
)

// Version is set at build time via -ldflags. Defaults to "dev".
var Version = "dev"

// RunVersion writes the version string and returns exit code 0.
func RunVersion(w io.Writer) int {
	fmt.Fprintln(w, "muster", Version)
	return 0
}
```

- [ ] **Step 4: Implement the run subcommand**

Create `internal/cli/run.go`:
```go
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bis-code/muster/internal/orchestrator"
	"github.com/bis-code/muster/internal/spec"
)

// RunRunCmd implements `muster run <spec.md>`.
//
// v0 semantics:
//   - Parse the spec file.
//   - Create a run directory under ./.muster/runs/<ulid>/.
//   - Spawn every agent in the roster via the selected spawner.
//
// The spawner is selected via MUSTER_SPAWNER env:
//   - "mock" (default in tests): MockSpawner from the orchestrator package
//   - "claude-sdk": real Claude Agent SDK — NOT IMPLEMENTED IN PHASE 1
func RunRunCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: muster run <spec.md>")
		return 2
	}
	specPath := args[0]

	crew, err := spec.ParseFile(specPath)
	if err != nil {
		fmt.Fprintf(stderr, "parse spec: %v\n", err)
		return 1
	}

	mustDir, err := ensureMustDir()
	if err != nil {
		fmt.Fprintf(stderr, "ensure .muster: %v\n", err)
		return 1
	}

	absSpec, _ := filepath.Abs(specPath)
	run, err := orchestrator.CreateRun(mustDir, absSpec, crew)
	if err != nil {
		fmt.Fprintf(stderr, "create run: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "created run:", run.ID)

	sp, err := selectSpawner()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := run.SpawnAll(sp); err != nil {
		fmt.Fprintf(stderr, "spawn: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "spawn complete; manifest:", filepath.Join(run.Root, "manifest.json"))
	return 0
}

func ensureMustDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := filepath.Join(wd, ".muster")
	if err := os.MkdirAll(filepath.Join(root, "runs"), 0o755); err != nil {
		return "", err
	}
	return root, nil
}

func selectSpawner() (orchestrator.Spawner, error) {
	switch os.Getenv("MUSTER_SPAWNER") {
	case "", "mock":
		return &orchestrator.MockSpawner{Transcripts: map[string]string{}}, nil
	case "claude-sdk":
		return nil, errors.New("claude-sdk spawner is not implemented in phase 1")
	default:
		return nil, fmt.Errorf("unknown MUSTER_SPAWNER: %q", os.Getenv("MUSTER_SPAWNER"))
	}
}
```

- [ ] **Step 5: Implement the CLI entrypoint**

Create `cmd/muster/main.go`:
```go
// Command muster is the multi-agent harness CLI.
package main

import (
	"fmt"
	"os"

	"github.com/bis-code/muster/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		os.Exit(cli.RunRunCmd(os.Args[2:], os.Stdout, os.Stderr))
	case "version", "--version", "-v":
		os.Exit(cli.RunVersion(os.Stdout))
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %q\n", os.Args[1])
		printUsage(os.Stderr)
		os.Exit(2)
	}
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, `muster — call up the agents, ship the work

Usage:
  muster <command> [args]

Commands:
  run <spec.md>     Spawn a crew from a spec file
  version           Print the muster version
  help              Show this help

Phase 1 supports only 'run' and 'version'. Phase 2 adds mailbox
primitives; Phase 3 adds status/tail/inspect/finish + meetings.`)
}
```

- [ ] **Step 6: Run all tests**

Run:
```bash
go build ./...
go test ./... -v
```

Expected: all PASS.

- [ ] **Step 7: Verify the binary manually**

Run:
```bash
go build -o /tmp/muster ./cmd/muster
/tmp/muster version
/tmp/muster run testdata/hello.md
ls -la .muster/runs/
cat .muster/runs/latest/manifest.json | head
```

Expected: version prints, run creates `.muster/runs/<id>/`, manifest is valid JSON.

- [ ] **Step 8: Clean up any `.muster/` tree created during manual testing**

Run:
```bash
rm -rf .muster
```

(`.muster/runs/` is gitignored but clean up anyway to avoid confusion in CI.)

- [ ] **Step 9: Commit**

Run:
```bash
git add cmd/ internal/cli/
git commit -m "feat(cli): add 'muster run' and 'muster version' subcommands"
```

### Task 1.9: Phase 1 verification gate

**Files:**
- None created; this is a pure verification task.

- [ ] **Step 1: Run the full build + test suite one last time**

Run:
```bash
go vet ./...
go build ./...
go test ./... -v
```

Expected: all green. Copy the output into your response.

- [ ] **Step 2: Execute the end-to-end smoke test manually**

Run:
```bash
rm -rf .muster
go build -o /tmp/muster ./cmd/muster
/tmp/muster version
/tmp/muster run testdata/hello.md
test -L .muster/runs/latest || { echo "no latest symlink"; exit 1; }
test -f .muster/runs/latest/manifest.json || { echo "no manifest"; exit 1; }
jq -e '.status == "active"' .muster/runs/latest/manifest.json || { echo "wrong status"; exit 1; }
jq -e '.roster[0].role == "coordinator"' .muster/runs/latest/manifest.json || { echo "wrong roster"; exit 1; }
test -f .muster/runs/latest/transcripts/coordinator-01.jsonl || { echo "no transcript"; exit 1; }
echo "PHASE 1 GATE: PASS"
```

Expected: `PHASE 1 GATE: PASS`.

- [ ] **Step 3: Clean the .muster/ tree and commit the gate proof in a note**

Run:
```bash
rm -rf .muster
```

- [ ] **Step 4: Emit the completion promise**

Output exactly (when invoked from the ralph-loop session):

```
<promise>PHASE1_DONE</promise>
```

---

## Phase 2 — Mailbox primitives + worker spawn + identity

**Deliverables:** the full mailbox MCP, real Claude Agent SDK spawning, in-process MCP server, per-worker identity via SubagentStart hook, and an end-to-end integration test with a 3-worker crew exchanging messages through JSONL files.

**Detailed task breakdown is produced during Phase 2's ralph-loop session** by a brainstorming + planning pass at the start of the session (per the user's "razor-sharp issue per session" rule). The session will read this phase header + DoD and produce its own internal task list.

**Task outline (to be expanded at Phase 2 start):**

1. **Envelope type + schema constants** (`internal/mailbox/envelope.go`) — exact fields from design §5.1
2. **Store: append + read + cursor** (`internal/mailbox/store.go`)
3. **Watcher: fsnotify + 100ms macOS backstop** (`internal/mailbox/watcher.go`)
4. **`mailbox_wait` blocking primitive** (`internal/mailbox/wait.go`)
5. **Blackboard package** (`internal/blackboard/blackboard.go`)
6. **Task queue + atomic claim** (`internal/tasks/queue.go`)
7. **Identity: token mint + verify** (`internal/identity/token.go`)
8. **SubagentStart hook payload writer** (`internal/identity/hook.go`)
9. **MCP server wiring with mark3labs/mcp-go** (`internal/mcp/server.go`)
10. **MCP tool registrations** (5 files: tools_mailbox, tools_blackboard, tools_tasks, tools_roster, with canUseTool stamping `from`)
11. **Claude Agent SDK spawner** (`internal/orchestrator/spawn.go` — new `ClaudeSDKSpawner` alongside `MockSpawner`)
12. **Multi-worker spawn via new spawner**
13. **Integration test: three-worker hello with real message exchange** (drops mock, uses SDK)
14. **Identity spoofing negative test**
15. **fsnotify latency benchmark** (Linux + macOS)

**Dependencies to add:**
```
github.com/mark3labs/mcp-go
github.com/fsnotify/fsnotify
github.com/santhosh-tekuri/jsonschema/v6
```

**Phase 2 DoD gates (from design spec §9.2):**
- All Phase 1 gates still pass
- Integration test: 3-worker hello-world crew where workers exchange messages via mailboxes, finishes in <60s
- Contract tests pass for all envelope kinds
- Identity spoofing test passes (worker cannot set `from=` to another agent's id)
- `mailbox_wait` wakes within 500ms of a message append on both Linux and macOS

**Promise:** `<promise>PHASE2_DONE</promise>`

---

## Phase 3 — CLI polish + meetings primitive

**Deliverables:** `muster status/tail/inspect/finish` CLI subcommands, the SSE observability endpoint, and the full meetings package including the round-based coordinator loop.

**Task outline (to be expanded at Phase 3 start):**

1. **`internal/sse/server.go`** — HTTP handler, `GET /events` SSE stream, buffered fan-out
2. **Integration of SSE with mailbox watcher** — every envelope + manifest transition becomes an event
3. **`muster status` subcommand** (`internal/cli/status.go`) — reads all `.muster/runs/*/manifest.json`
4. **`muster tail` subcommand** (`internal/cli/tail.go`) — tails a run's transcripts + mailboxes
5. **`muster inspect` subcommand** (`internal/cli/inspect.go`) — dumps a single agent's transcript + inbox
6. **`muster finish` subcommand** (`internal/cli/finish.go`) — transitions manifest status; supports `--discard`
7. **Meetings types** (`internal/meetings/types.go`) — MeetingSpec, Turn, State, Decision per design §7
8. **Turn token** (`internal/meetings/turn_token.go`) — one-shot capability
9. **Meetings orchestrator** (`internal/meetings/orchestrator.go`) — round loop, speaker selection, termination precedence
10. **Meeting MCP tools** (`internal/mcp/tools_meetings.go`) — MeetingOpen/Close/Adjourn/Speak/Transcript/State
11. **`muster meeting` CLI subcommand group** (`internal/cli/meeting.go`)
12. **Meeting integration test** — 4 participants × 5 rounds, reaches decision.json, schema-validated
13. **Meeting timeout test** — deliberate TurnTimeout causes skip
14. **Meeting adjourn test** — external MeetingAdjourn cleanly closes
15. **SSE loss test** — full run is reconstructable from SSE stream alone

**Phase 3 DoD gates (from design spec §9.2):**
- All previous gates pass
- End-to-end test: spawn a 3-worker crew that opens a meeting, reaches a decision, and finishes via `muster finish`
- SSE endpoint streams events for a full run without loss
- Meeting timeout test passes (stalled speaker skipped, meeting advances)

**Promise:** `<promise>PHASE3_DONE</promise>`

---

## Phase 4 — Skill library + docs + release

**Deliverables:** the 13 SKILL.md files shipped with the binary, pressure-test verification files for every rigid skill, `muster init` subcommand, goreleaser config, and a README that makes the project launchable on a clean clone.

**Task outline (to be expanded at Phase 4 start):**

1. **Copy skill files** — from `../claude-toolkit/docs/superpowers/specs/2026-04-07-muster-skills/*.md` into `skills/<slug>/SKILL.md`
2. **Skill manifest** (`skills/skills.json`) — index of installed skills with name + description
3. **Per-skill pressure test** — 11 rigid skills × one fresh-subagent pressure-test each, results in `docs/skill-tests/<slug>.md`
4. **Iterate any skill that fails the test** — tighten language, re-dispatch fresh subagent, repeat until HONORED (max 3 iterations; if still failing, split the skill)
5. **`muster init` subcommand** (`internal/cli/init.go`) — scaffolds `.muster/` with a hello spec in a new project dir
6. **README.md** — quickstart, CLI reference, link to design spec commit SHA, contribution pointers
7. **`.goreleaser.yaml`** — darwin/amd64, darwin/arm64, linux/amd64, linux/arm64 + checksums
8. **Linter config** — `.golangci.yaml` with conservative defaults
9. **CI stub** — `.github/workflows/test.yml` running `go vet`, `go test`, `golangci-lint`, `go build`
10. **Final smoke test from a clean clone** — verified by a subagent in an empty temp directory

**Phase 4 DoD gates (from design spec §9.2):**
- All previous gates pass
- Every rigid skill has a HONORED pressure-test file at `docs/skill-tests/<slug>.md`
- `muster init && muster run testdata/hello.md` works on a clean clone
- `go vet ./... && golangci-lint run` clean
- `goreleaser --snapshot --clean` builds binaries for all four targets

**Promise:** `<promise>PHASE4_DONE</promise>`

---

## Execution Handoff

This plan is designed to be driven by `/ralph-loop:ralph-loop`, one session per phase. The Phase 1 task list is detailed to the step level because it's the bootstrap. Phases 2–4 are outlined with DoD gates and task lists; each phase's ralph-loop session will expand its own task list at the start of the session using `superpowers:writing-plans` on the phase outline.

**Never** run a single ralph-loop across multiple phases. Commit after each phase, close the session, start a fresh one for the next phase.

**To start Phase 1 right now:**

1. Verify this plan is committed in `claude-toolkit` (the spec's handoff repo)
2. Open a fresh Claude Code session in any working directory (muster repo will be created by Task 1.1)
3. Paste the **Phase 1 prompt** from §Phase Launch Prompts into that session
4. Let ralph-loop run up to 30 iterations; do not babysit
5. When `<promise>PHASE1_DONE</promise>` appears, commit the final state in the muster repo and close the session
6. Repeat for Phases 2, 3, 4 — each in its own fresh session

---

## Self-Review Results

**1. Spec coverage.** Every section of the design spec maps to at least one task or phase:
- §1 Identity → Task 1.1 (LICENSE, README, repo init)
- §2 Goals/non-goals → Phase DoD gates + the "out of scope" list is respected (no broker, no trees, no dashboard UI)
- §3 Architecture + §4 Components → Phases 1–3 package breakdown matches §9.4 exactly
- §5 Data model → Task 1.5 (manifest), Phase 2 envelope schema task, Phase 3 meeting state
- §6 Run lifecycle → Task 1.6 (CreateRun), Task 1.7 (SpawnAll), Phase 3 finish
- §7 Meeting primitive → Phase 3 tasks 7–14
- §8 Skill library → Phase 4 tasks 1–4
- §9 Implementation plan → this document is the expansion
- §10 Open questions → left unresolved intentionally; revisit during respective phases
- §11 Provenance → preserved via the design spec commit referenced in README

**2. Placeholder scan.** No "TBD" / "TODO" / "fill in" / "similar to Task N" / "add appropriate error handling" in any of the Phase 1 tasks. Phases 2–4 use **outlines intentionally** with the note that detailed tasks are produced at session start — this is not a placeholder, it's a deliberate boundary to respect the "no mega-loops" memory rule.

**3. Type consistency.** `CrewSpec`, `Role`, `MailboxEdge`, `BlackboardKey`, `Manifest`, `Agent`, `AgentStatus`, `Status`, `TerminationSpec`, `Run`, `Spawner`, `MockSpawner` — all used consistently across tasks. Method names: `ParseFile`, `Parse`, `Write`, `Read`, `CreateRun`, `SpawnAll`, `Spawn`, `RunRunCmd`, `RunVersion` — no drift.

**4. Ambiguity check.** The one place where interpretation could drift is "muster init will be added in Phase 4" — I explicitly note Phase 1 relies on a user `mkdir` + `go mod init`, and Phase 4 task 5 is the subcommand. No conflict.

---

**Plan complete and saved to `docs/superpowers/plans/2026-04-07-muster-implementation.md`.**

**Execution path (per user directive): `/ralph-loop:ralph-loop`, one session per phase.**

The Phase 1 prompt in §Phase Launch Prompts is ready to copy-paste into a fresh session.

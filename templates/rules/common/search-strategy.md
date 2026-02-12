# Search Strategy

Use semantic search first. Fall back to targeted tools only when needed.

## Priority Order

| Priority | Tool | When to Use |
|----------|------|-------------|
| 1 | `leann_search` | Natural language queries, exploring unfamiliar code, broad concept search |
| 2 | `Read` | You already know the file path — read specific sections |
| 3 | `Grep` | Exact string/regex match — error messages, function names, imports |
| 4 | `Glob` | Find files by name pattern — `**/*.test.ts`, `**/auth*.go` |

## Startup: Check for LEANN Index

At the start of any exploration or search task, run `leann_list` once to discover available indexes. Cache the result mentally for the session.

If an index exists for the current project, prefer `leann_search` for the first query. It returns relevant code snippets in a single call — no need to guess file paths.

## LEANN Search

```
leann_search(index_name="<project>", query="<natural language>", top_k=10, show_metadata=true)
```

- Use natural language: "authentication middleware", "database connection pooling"
- Use `top_k=5` for focused results, `top_k=10-15` for broader exploration
- If results are relevant, stop — do not redundantly Grep for the same thing

## Grep / Glob Fallback

Use when LEANN is unavailable, returns no results, or you need exact matches:

- **Grep**: `files_with_matches` mode first, then `content` mode with context on matches
- **Glob**: find files by pattern, limit results to avoid noise
- **Search common locations first**: `src/`, `lib/`, `app/`, `pkg/`, `internal/`
- **Use `head_limit`**: cap results at 10-20 to stay focused

## Anti-Patterns

- Do not use `find`, `cat`, `head`, `tail` via Bash for searching — use dedicated tools
- Do not read entire files during search phase — read only after confirming a match
- Do not run multiple Grep queries when a single LEANN query would suffice
- Do not search `node_modules/`, `vendor/`, `.git/`, `dist/`, `build/`, `coverage/`

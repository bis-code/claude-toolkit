---
description: Efficient codebase search using LEANN index or Grep with smart defaults
---

# /search — Smart Codebase Search

Search the codebase efficiently using semantic search (LEANN) when available, with intelligent fallback to Grep. Prioritizes common code locations and limits results to stay focused.

## Arguments: $ARGUMENTS

The search query is the full `$ARGUMENTS` string:
- `/search authentication middleware` — natural language search
- `/search "func HandleLogin"` — exact code search
- `/search --type go token validation` — filter by file type
- `/search --files "*.test.ts" mock patterns` — filter by glob pattern

## Step 1: Check for LEANN Index

If the `leann-server` MCP is available:

```
mcp__leann-server__leann_list()
```

If an index exists for this project, use semantic search:

```
mcp__leann-server__leann_search(
  index_name="<project-index>",
  query="<search query>",
  top_k=10,
  show_metadata=true
)
```

Semantic search is preferred for natural language queries. If results are relevant, stop here.

## Step 2: Grep Fallback

If LEANN is unavailable or results are insufficient, use Grep:

### Search Priority Order

Search these directories first (if they exist):
1. `src/`, `lib/`, `app/`, `pkg/`
2. `components/`, `services/`, `utils/`
3. `api/`, `routes/`, `handlers/`
4. `internal/`, `cmd/`

### Search Strategies

For **natural language** queries, decompose into keywords and search for each:
```
Grep(pattern="keyword1.*keyword2", head_limit=10)
```

For **exact code** queries, search as-is:
```
Grep(pattern="exact string", head_limit=10)
```

For **definition** searches, use language-specific patterns:
- TypeScript/JS: `(function|const|class|interface)\s+<name>`
- Go: `func\s+(\(.*\)\s+)?<name>`, `type\s+<name>\s+(struct|interface)`
- Python: `(def|class)\s+<name>`
- Rust: `(fn|struct|enum|trait)\s+<name>`

## Step 3: Present Results

Format results with file paths and relevant context:

```
Search Results: "<query>"
━━━━━━━━━━━━━━━━━━━━━━━━
Method: LEANN semantic search (or Grep fallback)
Results: N matches

1. src/middleware/auth.ts:24
   export function authenticateRequest(req: Request) {

2. src/services/auth.ts:88
   async function validateToken(token: string): Promise<User> {

3. src/routes/auth.ts:12
   router.post('/login', authenticateRequest, handleLogin)
```

Limit to 10 results maximum. If more exist, note the total count and suggest narrowing the query.

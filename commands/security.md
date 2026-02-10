---
description: OWASP-focused security scan of recent changes or full codebase
---

# /security — Security Scan

Perform an OWASP-focused security review of the codebase or recent changes. Identify vulnerabilities, suggest fixes, and rate severity.

## Arguments: $ARGUMENTS

Parse `$ARGUMENTS` for scope:
- `/security` — scan recent changes (git diff)
- `/security --full` — scan entire codebase (key files only)
- `/security --file path` — scan a specific file
- `/security --deps` — focus on dependency vulnerabilities

## Step 1: Identify Attack Surface

Map the security-relevant code:
- **API endpoints** — routes, controllers, handlers
- **Authentication** — login, token generation, session management
- **Authorization** — role checks, permission guards, resource ownership
- **Data input** — form handlers, query parameters, request bodies
- **External integrations** — webhooks, third-party APIs, payment processing

## Step 2: OWASP Top 10 Checks

Scan for each category:

| # | Category | What to Look For |
|---|----------|-----------------|
| A01 | Broken Access Control | Missing auth middleware, IDOR, privilege escalation |
| A02 | Cryptographic Failures | Weak hashing, plaintext secrets, missing TLS |
| A03 | Injection | SQL injection, XSS, command injection, template injection |
| A04 | Insecure Design | Missing rate limiting, no account lockout |
| A05 | Security Misconfiguration | Debug mode enabled, default credentials, verbose errors |
| A06 | Vulnerable Components | Known CVEs in dependencies |
| A07 | Auth Failures | Weak passwords allowed, missing MFA, session fixation |
| A08 | Data Integrity | Missing signature verification, unsafe deserialization |
| A09 | Logging Failures | Sensitive data in logs, missing audit trail |
| A10 | SSRF | Unvalidated URLs, internal network access |

## Step 3: Secrets Detection

Scan for exposed secrets:
- API keys, tokens, passwords in code
- `.env` files committed or readable
- Hardcoded connection strings
- Private keys or certificates

## Step 4: Dependency Audit

If a package manager is detected:
```bash
npm audit          # Node.js
cargo audit        # Rust
pip-audit          # Python
govulncheck ./...  # Go
```

## Step 5: Report

```
Security Scan Report
━━━━━━━━━━━━━━━━━━━
Scope: <changes|full|file>

Findings:
  CRITICAL: N
  HIGH:     N
  MEDIUM:   N
  LOW:      N

[CRITICAL] A03 — SQL injection in src/api/users.go:84
  → User input passed directly to query without parameterization
  → Fix: Use parameterized query with GORM

[HIGH] A01 — Missing ownership check in src/api/documents.go:120
  → Any authenticated user can access any document by ID
  → Fix: Add user ownership verification before returning resource

Recommendations:
  1. <highest priority fix>
  2. <next priority fix>
```

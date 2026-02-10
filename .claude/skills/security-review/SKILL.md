---
name: security-review
description: "Security review checklist. Use when reviewing auth, API endpoints, or data handling."
---

# Security Review

Structured security analysis for code touching authentication, authorization, APIs, or data handling.

## When to Use

- Adding or modifying authentication flows
- Creating or changing API endpoints
- Handling user input or file uploads
- Working with secrets, tokens, or credentials

## OWASP Top 10 Checklist

- [ ] **Injection** -- Parameterized queries, no string concatenation in SQL/commands
- [ ] **Broken Authentication** -- Secure session management, proper password hashing
- [ ] **Sensitive Data Exposure** -- Encryption at rest and in transit, no secrets in logs
- [ ] **Broken Access Control** -- Authorization checks on every endpoint, ownership validation
- [ ] **Security Misconfiguration** -- No default credentials, secure headers, minimal permissions
- [ ] **XSS** -- Output encoding, CSP headers, sanitized user input
- [ ] **Insecure Deserialization** -- Validate and sanitize all deserialized data
- [ ] **Vulnerable Components** -- Dependencies audited, no known CVEs
- [ ] **Insufficient Logging** -- Security events logged, no sensitive data in logs
- [ ] **SSRF** -- Validate and allowlist outbound URLs, block internal network access

## Auth Review

- [ ] Passwords hashed with bcrypt/argon2 (never MD5/SHA)
- [ ] JWT tokens have expiration and are validated on every request
- [ ] Refresh tokens rotated on use and revocable
- [ ] Failed login attempts are rate-limited
- [ ] Every endpoint checks user permissions (no implicit trust)
- [ ] Users can only access their own resources (ownership check)
- [ ] Subscription tier gating enforced server-side (never client-only)

## Input Validation

- [ ] All user input validated at the API boundary
- [ ] File uploads checked for type, size, and content
- [ ] Query parameters, path parameters, and body all validated
- [ ] Reject unexpected fields (allowlist over denylist)

## Secrets Management

- [ ] No secrets in source code or environment variable defaults
- [ ] Secrets loaded from vault or secure environment at runtime
- [ ] API keys and tokens are never logged (even partially)
- [ ] Webhook signatures verified before processing payloads

## Common Vulnerability Patterns

| Pattern | Risk | Mitigation |
|---------|------|------------|
| `user.id` from request body | Privilege escalation | Use `user.id` from authenticated session |
| String interpolation in SQL | SQL injection | Use parameterized queries or ORM |
| `eval()` or dynamic execution | Remote code execution | Never evaluate user-controlled strings |
| Returning full DB objects | Data leakage | Use DTOs/serializers to control output |
| CORS with `*` origin | Cross-origin attacks | Allowlist specific origins |
| Missing rate limiting | Brute force / DoS | Rate limits at gateway and per-endpoint |

## After Review

- Document any accepted risks with justification
- Create issues for findings that cannot be fixed immediately
- Ensure security-related changes have corresponding tests

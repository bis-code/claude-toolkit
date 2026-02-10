# Security

Secure by default. Every change must consider attack surface.

## Input Validation

- Validate ALL user input at API boundaries — type, length, format, range
- Use allowlists over denylists — reject unknown input rather than filtering known bad input
- Validate Content-Type headers match expected format
- Limit file upload size and validate file types
- Reject oversized payloads early

## Data Access

- Use parameterized queries or ORM-provided methods — never concatenate user input into queries
- Enforce authorization on every endpoint — verify the user owns the resource they are accessing
- Check permissions at the data layer, not just the UI layer
- Apply principle of least privilege — grant minimum access required

## Secrets Management

- Never hardcode secrets, API keys, tokens, or passwords in source code
- Use environment variables or a secrets manager
- Never commit `.env` files, credentials, or private keys
- Rotate secrets when exposed — even in private repos

## Output & Transport

- Sanitize all output to prevent XSS — escape HTML, JavaScript, and URL contexts
- Use HTTPS everywhere — no exceptions
- Set security headers: `Content-Security-Policy`, `X-Content-Type-Options`, `Strict-Transport-Security`
- Return generic error messages to users — log detailed errors server-side only

## Authentication & Sessions

- Hash passwords with bcrypt, scrypt, or argon2 — never MD5 or SHA-1
- Validate JWT signatures and expiration on every request
- Implement rate limiting on auth endpoints
- Invalidate sessions on logout and password change

## Logging

- Never log passwords, tokens, API keys, or PII
- Log authentication events: login, logout, failed attempts, privilege changes
- Mask sensitive fields in logs: show last 4 of card numbers, redact emails

## OWASP Top 10 Awareness

Before any change that handles user input, authentication, or data access, consider:

1. Injection (SQL, NoSQL, OS command, LDAP)
2. Broken authentication
3. Sensitive data exposure
4. XML external entities (XXE)
5. Broken access control
6. Security misconfiguration
7. Cross-site scripting (XSS)
8. Insecure deserialization
9. Using components with known vulnerabilities
10. Insufficient logging and monitoring

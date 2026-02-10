# Go Coding Standards

## Core Principles

- Accept interfaces, return structs
- Make the zero value useful
- Keep packages small and focused on a single responsibility
- Prefer composition over inheritance

## Error Handling

- Always handle errors; never use `_` to discard them
- Wrap errors with context using `fmt.Errorf("operation failed: %w", err)`
- Use custom error types for errors that callers need to inspect
- Use `errors.Is` and `errors.As` for error comparison
- Return errors, don't panic (reserve `panic` for truly unrecoverable states)

## Naming

- Short, lowercase package names (no underscores, no camelCase)
- Exported names are the package's API; keep them intentional
- Avoid stuttering: `user.User` is fine, `user.UserService` is not
- Receiver names: 1-2 letter abbreviation of the type (`func (s *Server)`)

## Context

- Pass `context.Context` as the first parameter to all functions that do I/O
- Never store context in structs
- Use context for cancellation, deadlines, and request-scoped values only
- Create child contexts for sub-operations with tighter deadlines

## Structs and Interfaces

- Define interfaces where they are used, not where they are implemented
- Keep interfaces small (1-3 methods); compose larger ones
- Use struct embedding for shared behavior, not for polymorphism
- Avoid `init()` functions; prefer explicit initialization

## Logging

- Use `log/slog` (Go 1.21+) for structured logging
- Pass a logger via dependency injection, not as a global
- Log at appropriate levels: Error, Warn, Info, Debug
- Include request IDs and relevant context in log entries

## Concurrency

- Start goroutines with clear ownership and lifecycle
- Use `errgroup` for parallel operations with error collection
- Prefer channels for communication, mutexes for state protection
- Always ensure goroutines can be cancelled via context

## Code Organization

```
cmd/           # Entry points (main packages)
internal/      # Private application code
  domain/      # Business logic, no external dependencies
  handler/     # HTTP/gRPC handlers
  repository/  # Data access
  service/     # Application services
pkg/           # Public libraries (use sparingly)
```

## Performance

- Preallocate slices and maps when size is known: `make([]T, 0, n)`
- Use `strings.Builder` for string concatenation in loops
- Avoid unnecessary allocations in hot paths
- Profile before optimizing; use `pprof` and benchmarks

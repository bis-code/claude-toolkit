# Go Patterns

## Functional Options

Use for configurable constructors with clean defaults:

```go
type Option func(*Server)

func WithPort(port int) Option {
    return func(s *Server) { s.port = port }
}

func NewServer(opts ...Option) *Server {
    s := &Server{port: 8080} // sensible defaults
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

- Prefer over config structs when defaults matter
- Each option should be independently optional

## Repository Pattern

- Abstract data access behind interfaces defined in the domain layer
- Repositories return domain types, not database-specific types
- Keep query logic in the repository; business logic in the service
- Use transactions at the service layer, not the repository layer

```go
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}
```

## Middleware Chains

- Use `func(http.Handler) http.Handler` signature for composability
- Order matters: logging -> recovery -> auth -> rate limit -> handler
- Keep middleware focused on a single cross-cutting concern
- Pass request-scoped values via context, not global state

## Dependency Injection

- Inject dependencies via constructors, not global variables
- Wire dependencies in `main()` or a dedicated `wire.go`
- Use interfaces for external dependencies (DB, cache, APIs)
- Avoid DI frameworks; manual wiring is clearer in Go

## Graceful Shutdown

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

// Start server in goroutine
go srv.ListenAndServe()

<-ctx.Done()
shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
srv.Shutdown(shutdownCtx)
```

- Drain in-flight requests before stopping
- Close database connections and flush buffers
- Set a shutdown timeout to prevent hanging

## Configuration

- Use environment variables for deployment config
- Parse config at startup, validate early, fail fast
- Pass config as structs to constructors, not as individual values
- Never read environment variables deep in the codebase

## Worker Pools

- Use bounded goroutines with semaphore or worker pool pattern
- Process items from a channel; control concurrency with pool size
- Always handle panics in worker goroutines with `recover`
- Use `errgroup` with `SetLimit` for simple parallel task execution

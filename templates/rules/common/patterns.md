# Patterns

Established design patterns for consistent, maintainable code. Apply these by default. Deviate only with an explicit reason documented in a comment or ADR.

## Repository Pattern

Isolate all data access behind a repository interface. Application code must not build queries directly.

```typescript
// Interface — what the domain sees
interface UserRepository {
  findById(id: string): Promise<User | null>
  findByEmail(email: string): Promise<User | null>
  save(user: User): Promise<User>
  delete(id: string): Promise<void>
}

// Implementation — where SQL or ORM lives
class PostgresUserRepository implements UserRepository {
  async findById(id: string): Promise<User | null> {
    return db.users.findUnique({ where: { id } })
  }
}
```

Why: swappable implementations, testable with in-memory fakes, clear ownership of query logic.

## API Response Envelope

All API responses use a consistent envelope. Clients check `ok` before accessing `data`.

```typescript
// Success
{ ok: true, data: T }

// Error
{ ok: false, error: { code: string, message: string, details?: unknown } }

// Paginated list
{ ok: true, data: T[], pagination: { cursor: string | null, hasMore: boolean, total?: number } }
```

Never return raw data at the top level. Never mix success and error shapes.

## Skeleton Project Structure

Group by feature/domain, not by type. Each domain owns its handler, service, repository, and tests.

```
src/
  users/
    handler.ts         # HTTP layer — parse request, call service, return envelope
    service.ts         # Business logic — orchestrates repositories, emits events
    repository.ts      # Data access — all SQL/ORM here
    service.test.ts    # Unit tests for service
    repository.test.ts # Integration tests against real DB
  orders/
    ...
  shared/
    db.ts              # DB connection and pool config
    errors.ts          # Shared error types
    middleware.ts      # Auth, logging, rate limiting
```

## Dependency Injection

Pass dependencies explicitly. Do not import singletons inside functions.

```typescript
// GOOD — testable, explicit
class OrderService {
  constructor(
    private readonly orders: OrderRepository,
    private readonly users: UserRepository,
    private readonly events: EventBus,
  ) {}
}

// BAD — hidden coupling, hard to test
class OrderService {
  async create(data: CreateOrderDto) {
    const db = getDatabase() // singleton — impossible to swap in tests
  }
}
```

Wire dependencies at the composition root (main entrypoint), not inside domain classes.

## Error Handling

Define typed errors at domain boundaries. Never throw raw `Error` with string messages across service boundaries.

```typescript
class NotFoundError extends Error {
  constructor(public readonly resource: string, public readonly id: string) {
    super(`${resource} ${id} not found`)
    this.name = 'NotFoundError'
  }
}

class ValidationError extends Error {
  constructor(public readonly fields: Record<string, string>) {
    super('Validation failed')
    this.name = 'ValidationError'
  }
}
```

Map domain errors to HTTP status codes at the handler layer, not in the service.

## When to Deviate

These patterns add structure. For very small scripts, CLIs, or throwaway tools, the overhead is not worth it. Document the exception: `// Skipping repository pattern — single-use migration script`.

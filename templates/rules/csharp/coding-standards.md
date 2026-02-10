# C# Coding Standards

## Nullable Reference Types

- Enable `<Nullable>enable</Nullable>` in all projects
- Use `?` suffix for genuinely nullable types
- Handle nullable warnings; do not suppress with `!` unless justified
- Use `required` keyword for non-optional properties (C# 11+)

## Type Design

- Use `record` types for immutable DTOs and value objects
- Use `record struct` for small, stack-allocated value types
- Use `sealed` on classes not designed for inheritance
- Prefer `init` properties over mutable setters for data classes

## Async/Await

- Use `async/await` for all I/O-bound operations
- Suffix async methods with `Async` (e.g., `GetUserAsync`)
- Never use `.Result` or `.Wait()` on tasks (deadlock risk)
- Use `ConfigureAwait(false)` in library code
- Return `Task` or `ValueTask`, never `async void` (except event handlers)

## LINQ

- Use LINQ for collection transformations; prefer method syntax for complex queries
- Avoid multiple enumerations: materialize with `.ToList()` or `.ToArray()` when needed
- Use `IReadOnlyList<T>` or `IReadOnlyCollection<T>` for return types
- Prefer `Array.Empty<T>()` and `Enumerable.Empty<T>()` over `new List<T>()`

## Dependency Injection

- Register services in `Program.cs` or extension methods (`AddMyServices()`)
- Use constructor injection exclusively
- Prefer `IServiceScopeFactory` over injecting scoped services into singletons
- Register interfaces, not concrete types

## IDisposable

- Implement `IDisposable` / `IAsyncDisposable` for unmanaged resources
- Use `using` declarations (C# 8+) for scoped lifetime
- Never rely on finalizers for cleanup; they are non-deterministic

## Naming Conventions

- `PascalCase` for public members, types, namespaces, methods
- `camelCase` for local variables and parameters
- `_camelCase` for private fields (prefix with underscore)
- `I` prefix for interfaces: `IUserRepository`
- `T` prefix for type parameters: `TResult`

## Error Handling

- Throw specific exceptions: `ArgumentNullException`, `InvalidOperationException`
- Use exception filters: `catch (HttpRequestException ex) when (ex.StatusCode == 404)`
- Use `Result<T>` pattern for expected domain failures
- Log exceptions with structured logging (Serilog, Microsoft.Extensions.Logging)

## Project Organization

```
src/
  MyApp.Api/            # Web host, controllers, middleware
  MyApp.Application/    # Use cases, DTOs, interfaces
  MyApp.Domain/         # Entities, value objects, domain events
  MyApp.Infrastructure/ # Data access, external services
tests/
  MyApp.UnitTests/
  MyApp.IntegrationTests/
```

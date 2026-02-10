# Java Coding Standards

## Modern Java Features

- Use `record` types for DTOs and value objects (Java 16+)
- Use `sealed` interfaces to define closed type hierarchies (Java 17+)
- Use `var` for local variables when the type is obvious from context
- Use text blocks (`"""`) for multiline strings
- Use pattern matching in `instanceof` checks and switch expressions

## Optional

- Return `Optional<T>` for methods that may not produce a result
- Never use `Optional` as a method parameter or field type
- Use `orElseThrow()` with a meaningful exception, not `get()`
- Chain with `map`, `flatMap`, `filter` before unwrapping

## Collections and Streams

- Use `List.of()`, `Map.of()`, `Set.of()` for immutable collections
- Use Streams for declarative data transformations
- Prefer `toList()` (Java 16+) over `collect(Collectors.toList())`
- Avoid side effects in stream operations; use `forEach` only for terminal actions
- Use parallel streams only for CPU-intensive work on large datasets

## Resource Management

- Use try-with-resources for all `AutoCloseable` resources
- Never rely on `finalize()` (deprecated and unreliable)
- Close connections, streams, and clients explicitly
- Use connection pools for database and HTTP connections

## Builder Pattern

Use for objects with many optional parameters:

```java
var config = ServerConfig.builder()
    .port(8080)
    .maxConnections(100)
    .timeout(Duration.ofSeconds(30))
    .build();
```

- Make the constructor private; expose only the builder
- Validate in `build()`, not in setters
- Consider Lombok `@Builder` for reducing boilerplate

## Error Handling

- Use specific exception types: `IllegalArgumentException`, `EntityNotFoundException`
- Catch specific exceptions; never catch `Exception` or `Throwable` broadly
- Use custom exceptions for domain-specific errors
- Include context in exception messages: what failed and why

## Naming Conventions

- `PascalCase` for classes, interfaces, enums, records
- `camelCase` for methods, variables, parameters
- `SCREAMING_SNAKE_CASE` for constants (`static final`)
- Packages are all lowercase, reverse domain: `com.example.myapp`

## Dependency Injection

- Prefer constructor injection over field injection
- Use `final` fields for injected dependencies
- Define service interfaces for testability
- Keep injection configuration in dedicated config classes

## Project Structure

```
src/main/java/com/example/
  config/         # Spring/framework configuration
  domain/         # Entities, value objects, repositories
  application/    # Use cases, DTOs, service interfaces
  infrastructure/ # Data access, external APIs, messaging
  api/            # Controllers, request/response objects
src/test/java/com/example/
  unit/
  integration/
```

## Code Quality

- Enable compiler warnings and treat them as errors in CI
- Use SpotBugs or Error Prone for static analysis
- Use Checkstyle or Spotless for consistent formatting
- Suppress warnings only with an explanatory comment

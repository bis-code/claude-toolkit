# Rust Patterns

## Builder Pattern

Use for constructing complex types with optional configuration:

```rust
pub struct ServerBuilder {
    port: u16,
    max_connections: Option<usize>,
}

impl ServerBuilder {
    pub fn new(port: u16) -> Self {
        Self { port, max_connections: None }
    }

    pub fn max_connections(mut self, n: usize) -> Self {
        self.max_connections = Some(n);
        self
    }

    pub fn build(self) -> Result<Server, ConfigError> {
        // Validate and construct
    }
}
```

- Use consuming `self` (not `&mut self`) for method chaining
- Validate in `build()`, not in setters
- Return `Result` from `build()` if validation can fail

## Newtype Pattern

Wrap primitive types for type safety and domain semantics:

```rust
pub struct UserId(Uuid);
pub struct Email(String);
```

- Prevents mixing up `String` arguments that mean different things
- Implement `From`, `Display`, `Deref` as appropriate
- Use `derive_more` crate to reduce boilerplate

## Typestate Pattern

Encode state transitions in the type system:

```rust
struct Connection<S: State> { /* ... */ _state: PhantomData<S> }
struct Disconnected;
struct Connected;

impl Connection<Disconnected> {
    fn connect(self) -> Result<Connection<Connected>, Error> { /* ... */ }
}

impl Connection<Connected> {
    fn query(&self, sql: &str) -> Result<Rows, Error> { /* ... */ }
}
```

- Prevents invalid state transitions at compile time
- Use for protocols, workflows, and resource lifecycles
- Keep states as zero-sized types (no runtime cost)

## Error Handling Architecture

```rust
// Library errors with thiserror
#[derive(Debug, thiserror::Error)]
pub enum DomainError {
    #[error("user not found: {0}")]
    UserNotFound(UserId),
    #[error("invalid email format")]
    InvalidEmail,
    #[error("database error")]
    Database(#[from] sqlx::Error),
}

// Application errors with anyhow
fn main() -> anyhow::Result<()> {
    let user = find_user(id).context("failed to load user profile")?;
    Ok(())
}
```

## Async with Tokio

- Use `tokio` as the async runtime (default choice for most projects)
- Use `tokio::spawn` for concurrent tasks; `tokio::join!` for parallel awaiting
- Use `tokio::select!` for racing multiple futures
- Prefer `async fn` over returning `impl Future` for clarity
- Use `tokio::sync::Mutex` only when holding the lock across `.await`; otherwise use `std::sync::Mutex`

## Trait Objects vs Generics

| Use Generics When | Use Trait Objects When |
|-------|-------|
| Performance matters (monomorphization) | You need heterogeneous collections |
| The type is known at compile time | You want to reduce binary size |
| API is small and focused | Plugin architectures / dynamic dispatch |

```rust
// Generic (compile-time dispatch)
fn process<T: Serialize>(item: &T) -> Result<()> { /* ... */ }

// Trait object (runtime dispatch)
fn process(item: &dyn Serialize) -> Result<()> { /* ... */ }
```

## Configuration Pattern

- Use `config` crate for layered configuration (file + env + defaults)
- Parse configuration into strongly typed structs at startup
- Validate early; fail fast with clear error messages
- Pass configuration by reference, not as globals

## Repository / Service Layer

- Define traits for data access; implement for specific backends
- Services accept trait objects or generics for testability
- Keep business logic in service functions, not in repository implementations
- Use `sqlx` for compile-time checked SQL queries

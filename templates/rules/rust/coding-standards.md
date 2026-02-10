# Rust Coding Standards

## Ownership and Borrowing

- Prefer `&str` over `String` in function parameters (accept references)
- Return owned types (`String`, `Vec<T>`) from functions when the caller needs ownership
- Use `Cow<'_, str>` when a function may or may not need to allocate
- Avoid unnecessary cloning; prefer borrowing and lifetime annotations

## Error Handling

- Use `Result<T, E>` for recoverable errors; never `panic!` in library code
- Use `thiserror` for library error types (derive `Error` with context)
- Use `anyhow` for application-level error handling (ergonomic error chaining)
- Use the `?` operator for error propagation; avoid manual `match` on `Result`
- Provide context with `.context()` or `.with_context(|| ...)`

## Derive and Traits

- Derive common traits by default: `Debug`, `Clone`, `PartialEq`
- Add `Eq`, `Hash`, `PartialOrd`, `Ord` when semantically appropriate
- Derive `Serialize` and `Deserialize` (serde) for types that cross boundaries
- Implement `Display` for user-facing error messages and output

## Naming Conventions

- `snake_case` for functions, methods, variables, modules, crates
- `PascalCase` for types, traits, and enum variants
- `SCREAMING_SNAKE_CASE` for constants and statics
- Prefix boolean functions with `is_`, `has_`, `can_`, `should_`

## Module Organization

```
src/
  lib.rs          # Public API, re-exports
  main.rs         # Entry point (binary crates)
  config.rs       # Configuration types
  error.rs        # Error types
  domain/         # Business logic
    mod.rs
  api/            # HTTP handlers
  db/             # Database access
```

- Use `mod.rs` or file-based modules depending on team convention (be consistent)
- Re-export public types in `lib.rs` for a clean API surface
- Keep module depth shallow; prefer flat structures

## Clippy

- Run `cargo clippy` with pedantic lints enabled
- Configure in `Cargo.toml` or `clippy.toml`
- Allow specific lints with `#[allow()]` and a justifying comment
- Treat clippy warnings as errors in CI: `cargo clippy -- -D warnings`

## Lifetimes

- Rely on lifetime elision when possible; annotate only when required
- Use meaningful lifetime names for complex signatures: `'input`, `'conn`
- Prefer owned types in structs over lifetime-parameterized references
- Use `'static` bounds only when truly necessary (e.g., spawning threads)

## Unsafe Code

- Avoid `unsafe` unless absolutely necessary for performance or FFI
- Wrap `unsafe` in safe abstractions with clear invariant documentation
- Add `// SAFETY:` comments explaining why the unsafe block is sound
- Use `miri` to check unsafe code for undefined behavior

## Enums and Pattern Matching

- Use enums for state machines and algebraic data types
- Always handle all variants; avoid wildcard patterns (`_`) in exhaustive matches
- Use `#[non_exhaustive]` on public enums that may gain variants
- Prefer `if let` for single-variant matching

## Performance

- Use iterators and combinators; they are zero-cost abstractions
- Prefer `Vec::with_capacity` when the size is known
- Use `&[T]` instead of `&Vec<T>` in function signatures
- Profile with `cargo flamegraph` before optimizing

# Rust Testing Standards

## Unit Tests

Place unit tests in the same file as the code they test:

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_valid_email_succeeds() {
        let email = Email::parse("user@example.com").unwrap();
        assert_eq!(email.as_str(), "user@example.com");
    }

    #[test]
    fn parse_invalid_email_returns_error() {
        let result = Email::parse("not-an-email");
        assert!(result.is_err());
    }
}
```

- Use `#[cfg(test)]` to exclude test code from production builds
- Test public functions through the public API
- Use `assert_eq!`, `assert_ne!`, `assert!` for standard assertions
- Use `#[should_panic(expected = "message")]` for panic tests

## Doc Tests

- Write doc tests for public API functions (they serve as documentation and tests)
- Use `# ` prefix to hide setup lines from rendered documentation
- Doc tests run with `cargo test` by default

```rust
/// Adds two numbers together.
///
/// ```
/// # use mylib::add;
/// assert_eq!(add(2, 3), 5);
/// assert_eq!(add(0, 0), 0);
/// ```
pub fn add(a: i32, b: i32) -> i32 { a + b }
```

## Integration Tests

- Place in `tests/` directory at the crate root
- Each file in `tests/` is compiled as a separate crate
- Use `tests/common/mod.rs` for shared test utilities
- Integration tests can only access the crate's public API

```
tests/
  common/
    mod.rs          # Shared helpers
  api_tests.rs      # API integration tests
  db_tests.rs       # Database integration tests
```

## Property Testing (Proptest)

- Use `proptest` for property-based testing of pure functions
- Define strategies for generating valid input domain
- Test invariants rather than specific cases

```rust
use proptest::prelude::*;

proptest! {
    #[test]
    fn roundtrip_serialization(value: u64) {
        let serialized = to_bytes(value);
        let deserialized = from_bytes(&serialized).unwrap();
        assert_eq!(value, deserialized);
    }
}
```

## Async Test Support

- Use `#[tokio::test]` for async unit tests
- Use `#[tokio::test(flavor = "multi_thread")]` when testing concurrent behavior
- Use `tokio::time::pause()` for time-dependent tests

## Test Utilities

- Place shared helpers in `tests/common/mod.rs` or a `testutil` module
- Use builder patterns for constructing test data
- Create helper functions with descriptive names: `create_test_user()`
- Use `tempfile` crate for tests that need filesystem access

## Mocking

- Prefer dependency injection with traits over mocking frameworks
- Use `mockall` crate when mock behavior needs to be configurable
- Keep mock setup minimal; only stub what the test requires
- For HTTP testing, use `wiremock` to mock external services

## Coverage

- Use `cargo tarpaulin` or `cargo llvm-cov` for coverage reports
- Focus coverage on business logic and error handling paths
- Ignore generated code and FFI bindings in coverage
- Run coverage in CI; track trends over time

## Best Practices

- Keep tests fast: mock I/O, avoid sleeping, use test databases
- Each test should be independent; no shared mutable state
- Name tests descriptively: `function_scenario_expected`
- Use `#[ignore]` for slow tests; run them separately in CI
- Test error paths as thoroughly as success paths

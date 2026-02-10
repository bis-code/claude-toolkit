# Go Testing Standards

## Table-Driven Tests

Preferred pattern for most unit tests:

```go
tests := []struct {
    name    string
    input   string
    want    string
    wantErr bool
}{
    {"valid input", "hello", "HELLO", false},
    {"empty string", "", "", false},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := Transform(tt.input)
        if tt.wantErr {
            require.Error(t, err)
            return
        }
        require.NoError(t, err)
        assert.Equal(t, tt.want, got)
    })
}
```

## Assertions

- Use `testify/assert` for soft assertions (test continues on failure)
- Use `testify/require` for hard assertions (test stops on failure)
- Use `require` for preconditions; `assert` for the actual test assertions
- Never use `if got != want { t.Errorf(...) }` when testify is available

## Test Helpers

- Place test helpers in `_test.go` files alongside the code they test
- Use `t.Helper()` in all test helper functions for correct line reporting
- Shared test utilities go in a `testutil/` package
- Use `t.Cleanup()` for teardown instead of `defer` when possible

## Integration Tests

- Use Testcontainers for database and external service tests
- Guard integration tests with build tags: `//go:build integration`
- Each test should set up and tear down its own data
- Use a separate database per test or per package to avoid conflicts

## Mocking

- Define interfaces at the consumer, mock at the test level
- Use `testify/mock` or hand-written mocks (prefer hand-written for simplicity)
- Mock external boundaries only (database, HTTP clients, message queues)
- Never mock the code under test

## Golden Files

- Use for testing complex output (JSON responses, templates, CLI output)
- Store golden files in `testdata/` directory
- Update with a `-update` flag: `go test -run TestX -update`
- Review golden file diffs carefully in code review

## Benchmarks

- Use `testing.B` for performance-critical code
- Run with `go test -bench=. -benchmem`
- Compare results with `benchstat` before and after changes
- Include benchmarks for hot paths and allocations

## Test Organization

```
service/
  user.go
  user_test.go           # Unit tests
  user_integration_test.go  # Integration tests (build tagged)
  testdata/
    golden_user.json     # Golden files
```

## Best Practices

- Test public API; internal functions are tested through public behavior
- Use `t.Parallel()` for independent tests to speed up execution
- Keep tests deterministic; avoid time-dependent assertions without mocking time
- Use `testrand` or fixed seeds for tests involving randomness

# TypeScript Testing Standards

## Framework

- Prefer Vitest for new projects (faster, ESM-native, Vite-compatible)
- Jest is acceptable for existing projects; do not migrate without reason
- Use React Testing Library for component tests (not Enzyme)

## Test Structure

- One test file per module: `module.test.ts` or `module.spec.ts`
- Co-locate test files with source files (not in a separate `__tests__` directory)
- Use `describe` blocks to group related tests by function or behavior
- Test names should read as sentences: `it('returns null when user is not found')`

## What to Test

- **Test behavior, not implementation** - test what it does, not how
- Test public API of modules; avoid testing private functions directly
- Cover happy path, error cases, and edge cases
- Test error messages and error types, not just that errors are thrown

## Mocking

- Mock at module boundaries (API calls, database, file system)
- Use `vi.mock()` / `jest.mock()` for module-level mocks
- Prefer dependency injection over module mocking when possible
- Never mock what you don't own without an adapter layer
- Reset mocks between tests: `afterEach(() => vi.restoreAllMocks())`

## React Component Testing

- Query by role, label, or text (accessibility-first selectors)
- Never query by test ID unless no accessible alternative exists
- Use `userEvent` over `fireEvent` for realistic interactions
- Test loading, error, and empty states
- Avoid snapshot tests for components; test specific assertions instead

## Async Testing

- Always `await` async operations; never use `done` callback
- Use `waitFor` for assertions that depend on async state updates
- Use `vi.useFakeTimers()` for time-dependent tests
- Test both resolved and rejected promise paths

## Snapshot Tests

- Use sparingly: only for serialized output (API responses, config generation)
- Never use for UI components (too brittle, low signal)
- Review snapshot changes carefully; do not blindly update

## Test Data

- Use factory functions for test data (e.g., `createMockUser()`)
- Keep test data minimal; only set fields relevant to the test
- Avoid shared mutable test state; create fresh data per test

## Coverage

- Aim for meaningful coverage, not 100%
- Focus coverage on business logic, not UI glue code
- Use `/* v8 ignore next */` for intentionally uncovered lines (e.g., exhaustive switch defaults)

## Integration Tests

- Test API routes with `supertest` or framework-specific test utilities
- Use in-memory databases or Testcontainers for database tests
- Test middleware chains as integrated units

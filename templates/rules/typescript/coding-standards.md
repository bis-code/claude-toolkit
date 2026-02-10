# TypeScript Coding Standards

## Compiler Configuration

- Enable `strict: true` in tsconfig.json (no exceptions)
- Enable `noUncheckedIndexedAccess` for safer array/object access
- Set `target` and `lib` to match your runtime environment

## Variable Declarations

- Prefer `const` over `let`; never use `var`
- Use destructuring for object and array access
- Prefer template literals over string concatenation

## Type Safety

- **Never use `any`** - use `unknown` and narrow with type guards
- Explicit return types on all public/exported functions
- Use discriminated unions over type assertions
- Prefer `interface` for object shapes, `type` for unions and intersections
- Use `satisfies` operator for type-checked object literals
- Leverage `as const` for literal types

## Functions

- Prefer arrow functions for callbacks and inline functions
- Use named functions for top-level declarations (better stack traces)
- Limit function parameters to 3; use an options object beyond that
- Default parameters over optional parameters when a sensible default exists

## Module Organization

- Use barrel exports (`index.ts`) for public API of each module
- Configure path aliases (`@/` prefix) to avoid deep relative imports
- Co-locate types with the code that uses them
- Separate shared types into a dedicated `types/` directory

## Naming Conventions

- `PascalCase` for types, interfaces, enums, and classes
- `camelCase` for variables, functions, and methods
- `SCREAMING_SNAKE_CASE` for constants and enum members
- Prefix interfaces with behavior, not `I` (e.g., `Serializable`, not `ISerializable`)

## Error Handling

- Use custom error classes extending `Error`
- Prefer `Result<T, E>` pattern for expected failures over thrown exceptions
- Always type catch clause variables as `unknown`

## Async Patterns

- Prefer `async/await` over raw Promises
- Use `Promise.all` for independent concurrent operations
- Never fire-and-forget async calls without error handling

## Enums and Constants

- Prefer `as const` objects over `enum` (better tree-shaking)
- Use discriminated unions for state machines
- String unions for simple value sets: `type Status = 'active' | 'inactive'`

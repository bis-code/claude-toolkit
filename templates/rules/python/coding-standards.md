# Python Coding Standards

## Type Hints

- Use type hints on all function signatures (parameters and return types)
- Use `from __future__ import annotations` for modern syntax in older Python
- Use `TypeAlias` for complex type definitions
- Prefer `X | None` over `Optional[X]` (Python 3.10+)

## Data Modeling

- Use `dataclasses` for simple data containers
- Use Pydantic `BaseModel` for data with validation or serialization needs
- Use `NamedTuple` for immutable, lightweight records
- Prefer `Enum` for fixed sets of values; use `StrEnum` when string representation matters

## String Handling

- Use f-strings for interpolation (not `.format()` or `%` formatting)
- Use triple-quoted strings for multiline text
- Use `textwrap.dedent` for indented multiline strings in code

## File and Path Operations

- Use `pathlib.Path` over `os.path` for all path operations
- Use `Path.read_text()` / `Path.write_text()` for simple file I/O
- Use context managers (`with` statement) for file handles and resources

## Imports

- Sort with `isort` (compatible with ruff)
- Group: stdlib, third-party, local (separated by blank lines)
- Prefer explicit imports over wildcard (`from module import *` is forbidden)
- Use absolute imports; relative imports only within packages

## Error Handling

- Catch specific exceptions, never bare `except:`
- Use custom exception classes for domain-specific errors
- Use `raise ... from err` for exception chaining
- Let unexpected exceptions propagate; don't silently swallow errors

## Linting and Formatting

- Use `ruff` for linting and formatting (replaces flake8 + black + isort)
- Enable strict rule sets: `select = ["E", "F", "W", "I", "N", "UP", "B", "SIM"]`
- Use `mypy` with `strict = true` for type checking
- Run linters in CI; block merges on violations

## Functions

- Limit function parameters to 5; use dataclass or TypedDict for complex inputs
- Use `*` to force keyword-only arguments for clarity
- Prefer returning values over mutating arguments
- Use `functools.lru_cache` or `@cache` for expensive pure functions

## Project Structure

```
src/
  mypackage/
    __init__.py
    domain/        # Business logic
    services/      # Application services
    adapters/      # External integrations
    api/           # HTTP/CLI interfaces
tests/
  unit/
  integration/
pyproject.toml     # Single config file for all tools
```

## Virtual Environments

- Always use a virtual environment (venv, uv, poetry)
- Pin dependencies with lock files
- Separate dev dependencies from production dependencies

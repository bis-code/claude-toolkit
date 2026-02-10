# Python Testing Standards

## Framework

- Use `pytest` as the test runner (not `unittest`)
- Use `pytest-cov` for coverage reporting
- Use `pytest-xdist` for parallel test execution
- Configure in `pyproject.toml` under `[tool.pytest.ini_options]`

## Test Structure

- Place tests in a `tests/` directory mirroring `src/` structure
- Name test files `test_<module>.py`
- Name test functions `test_<behavior_being_tested>`
- Use `class` grouping only when tests share significant setup

## Fixtures

- Use `@pytest.fixture` for test setup and teardown
- Prefer factory fixtures over static data fixtures
- Scope fixtures appropriately: `function` (default), `module`, `session`
- Place shared fixtures in `conftest.py` at the appropriate directory level

## Factory Boy

- Use `factory_boy` for generating test data
- Define one factory per model with sensible defaults
- Use `SubFactory` for related objects
- Use `Sequence` for unique fields: `name = factory.Sequence(lambda n: f"user-{n}")`
- Use `LazyAttribute` for computed fields

## Parametrize

- Use `@pytest.mark.parametrize` for testing multiple input variants
- Keep parameter sets readable; use `pytest.param` with `id` for clarity
- Prefer parametrize over copy-pasted test functions

```python
@pytest.mark.parametrize("input_val,expected", [
    pytest.param("hello", "HELLO", id="lowercase"),
    pytest.param("", "", id="empty-string"),
    pytest.param("123", "123", id="numeric"),
])
def test_uppercase(input_val, expected):
    assert to_upper(input_val) == expected
```

## Mocking

- Use `unittest.mock.patch` to mock at module boundaries
- Patch where the name is used, not where it is defined
- Prefer dependency injection over patching when possible
- Use `MagicMock(spec=ClassName)` to catch attribute errors
- Always assert mock calls: `mock.assert_called_once_with(...)`

## Async Testing

- Use `pytest-asyncio` for async test functions
- Mark async tests with `@pytest.mark.asyncio`
- Use `async` fixtures for async setup/teardown

## Django-Specific

- Use `pytest-django` with `@pytest.mark.django_db` for database tests
- Use `client` fixture for view tests; `api_client` for DRF tests
- Use `django.test.override_settings` for config-dependent tests
- Prefer `baker` or `factory_boy` over manual `Model.objects.create()`

## Coverage

- Aim for high coverage on business logic; lower is acceptable for glue code
- Use `# pragma: no cover` sparingly and with justification
- Run coverage in CI; fail on regression below threshold
- Focus on branch coverage, not just line coverage

## Best Practices

- Each test should be independent; no ordering dependencies
- Clean up side effects in fixtures, not in tests
- Keep tests fast: mock I/O, use in-memory databases for unit tests
- Test behavior, not implementation details

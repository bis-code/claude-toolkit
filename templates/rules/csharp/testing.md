# C# Testing Standards

## Framework

- Use xUnit as the test framework (preferred over NUnit/MSTest)
- Use FluentAssertions for readable, expressive assertions
- Use Moq for mocking interfaces and abstract classes
- Use `WebApplicationFactory<T>` for integration testing ASP.NET Core

## Test Naming

Follow the pattern: `Method_Scenario_ExpectedResult`

```csharp
[Fact]
public async Task GetUser_WithValidId_ReturnsUser()

[Fact]
public async Task GetUser_WithInvalidId_ThrowsNotFoundException()

[Fact]
public void Calculate_NegativeInput_ReturnsZero()
```

## Test Structure (AAA)

```csharp
[Fact]
public async Task CreateOrder_ValidItems_ReturnsOrderWithTotal()
{
    // Arrange
    var service = new OrderService(_mockRepo.Object);
    var items = new[] { new OrderItem("SKU-1", 2, 10.00m) };

    // Act
    var result = await service.CreateOrderAsync(items);

    // Assert
    result.Should().NotBeNull();
    result.Total.Should().Be(20.00m);
    result.Items.Should().HaveCount(1);
}
```

## FluentAssertions

- Use `.Should()` for all assertions
- Chain assertions for readability: `.Should().NotBeNull().And.HaveCount(3)`
- Use `.BeEquivalentTo()` for structural comparison
- Use `.Throw<T>()` with `.WithMessage()` for exception testing

## Mocking with Moq

- Mock interfaces, not concrete classes
- Use `Mock.Of<T>()` for simple mocks; `new Mock<T>()` for setup-heavy mocks
- Verify method calls with `mock.Verify()` sparingly (avoid over-specification)
- Use `It.IsAny<T>()` for arguments you don't care about

## Integration Tests

- Use `WebApplicationFactory<Program>` for API integration tests
- Override services with test doubles in `ConfigureTestServices`
- Use Testcontainers for real database tests
- Isolate tests: each test gets its own scope and database transaction

```csharp
public class UserApiTests : IClassFixture<WebApplicationFactory<Program>>
{
    private readonly HttpClient _client;

    public UserApiTests(WebApplicationFactory<Program> factory)
    {
        _client = factory.CreateClient();
    }
}
```

## Test Data

- Use Builder pattern or factory methods for test data
- Use `AutoFixture` for generating arbitrary test data
- Keep test data minimal; only set fields relevant to the test scenario
- Use `Bogus` for realistic fake data when needed

## Theory Tests (Parameterized)

```csharp
[Theory]
[InlineData("", false)]
[InlineData("valid@email.com", true)]
[InlineData("no-at-sign", false)]
public void IsValidEmail_VariousInputs_ReturnsExpected(string email, bool expected)
{
    EmailValidator.IsValid(email).Should().Be(expected);
}
```

## Best Practices

- One assertion concept per test (multiple `Should()` calls are fine if testing one concept)
- Use `IAsyncLifetime` for async setup/teardown instead of constructor
- Mark slow tests with `[Trait("Category", "Integration")]`
- Run unit tests on every build; integration tests in CI pipeline

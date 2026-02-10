# Java Testing Standards

## Framework

- Use JUnit 5 (Jupiter) as the test framework
- Use Mockito for mocking dependencies
- Use AssertJ for fluent, readable assertions
- Use Testcontainers for database and service integration tests

## Test Naming

Follow descriptive naming: `methodName_scenario_expectedBehavior`

```java
@Test
void createOrder_validItems_returnsOrderWithCorrectTotal() { }

@Test
void createOrder_emptyItems_throwsValidationException() { }

@Test
void getUser_nonExistentId_returnsEmpty() { }
```

## Test Structure (AAA)

```java
@Test
void calculateDiscount_goldMember_returns20Percent() {
    // Arrange
    var member = new Member(MemberTier.GOLD);
    var order = new Order(BigDecimal.valueOf(100));

    // Act
    var discount = discountService.calculate(member, order);

    // Assert
    assertThat(discount).isEqualByComparingTo(BigDecimal.valueOf(20));
}
```

## AssertJ

- Use `assertThat()` for all assertions (not JUnit's `assertEquals`)
- Chain assertions: `assertThat(list).hasSize(3).contains("a").doesNotContain("z")`
- Use `extracting()` for asserting on object fields
- Use `assertThatThrownBy()` for exception testing

## Mockito

- Use `@ExtendWith(MockitoExtension.class)` with JUnit 5
- Annotate mocks with `@Mock`, the subject with `@InjectMocks`
- Use `when().thenReturn()` for stubbing; `verify()` sparingly
- Use `ArgumentCaptor` when you need to inspect passed arguments
- Never mock types you don't own without an adapter layer

## Spring Boot Tests

- Use `@SpringBootTest` sparingly (it loads the full context; slow)
- Prefer `@WebMvcTest` for controller tests (loads only web layer)
- Use `@DataJpaTest` for repository tests (configures in-memory DB)
- Use `@MockBean` to replace beans in the Spring context
- Use `TestRestTemplate` or `MockMvc` for HTTP-level testing

```java
@WebMvcTest(UserController.class)
class UserControllerTest {
    @Autowired private MockMvc mockMvc;
    @MockBean private UserService userService;

    @Test
    void getUser_existingId_returns200() throws Exception {
        when(userService.findById(1L)).thenReturn(Optional.of(testUser));

        mockMvc.perform(get("/api/users/1"))
            .andExpect(status().isOk())
            .andExpect(jsonPath("$.name").value("Alice"));
    }
}
```

## Testcontainers

- Use for integration tests requiring real databases or services
- Define containers as static fields with `@Container` annotation
- Use `@Testcontainers` annotation on the test class
- Reuse containers across tests with `withReuse(true)` for speed
- Prefer `@DynamicPropertySource` for injecting container connection details

## Parameterized Tests

```java
@ParameterizedTest
@CsvSource({
    "valid@email.com, true",
    "no-at-sign, false",
    "'', false"
})
void isValidEmail_variousInputs_returnsExpected(String email, boolean expected) {
    assertThat(EmailValidator.isValid(email)).isEqualTo(expected);
}
```

## Best Practices

- Each test must be independent; no shared mutable state
- Use `@BeforeEach` for common setup; `@AfterEach` for cleanup
- Keep unit tests under 50ms; integration tests under 5s
- Use `@Tag("integration")` to separate slow tests from fast ones
- Run unit tests on every build; integration tests in CI

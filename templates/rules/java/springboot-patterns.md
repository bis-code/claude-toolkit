# Spring Boot Patterns

## Dependency Injection

- Use constructor injection exclusively (not `@Autowired` on fields)
- Single constructor does not need `@Autowired` annotation
- Use `final` fields for all injected dependencies
- Use `@RequiredArgsConstructor` (Lombok) to reduce boilerplate

```java
@Service
public class OrderService {
    private final OrderRepository orderRepository;
    private final PaymentGateway paymentGateway;

    public OrderService(OrderRepository orderRepository, PaymentGateway paymentGateway) {
        this.orderRepository = orderRepository;
        this.paymentGateway = paymentGateway;
    }
}
```

## Transaction Management

- Apply `@Transactional` at the service layer, not repositories or controllers
- Use `readOnly = true` for query-only methods
- Be explicit about propagation and isolation when non-default is needed
- Never catch exceptions inside `@Transactional` methods without re-throwing

## Exception Handling

- Use `@RestControllerAdvice` for global exception handling
- Map domain exceptions to HTTP status codes in one place
- Return consistent error response format (RFC 7807 Problem Details)
- Log server errors (5xx); don't log client errors (4xx) at ERROR level

```java
@RestControllerAdvice
public class GlobalExceptionHandler {
    @ExceptionHandler(EntityNotFoundException.class)
    public ProblemDetail handleNotFound(EntityNotFoundException ex) {
        return ProblemDetail.forStatusAndDetail(HttpStatus.NOT_FOUND, ex.getMessage());
    }
}
```

## Profiles and Configuration

- Use `application.yml` with profiles: `default`, `dev`, `staging`, `prod`
- Use `@ConfigurationProperties` for typed configuration classes
- Validate config at startup with `@Validated` and Bean Validation annotations
- Never hardcode environment-specific values; always externalize

## REST API Design

- Use `@RestController` with explicit `@RequestMapping` base path
- Return `ResponseEntity<T>` for explicit status code control
- Use `@Valid` on request bodies for input validation
- Version APIs via URL path (`/api/v1/`) or header

## Data Access (JPA)

- Use Spring Data JPA repositories for standard CRUD
- Write custom `@Query` methods for complex queries
- Use `@EntityGraph` to avoid N+1 queries
- Prefer JPQL or native queries over Criteria API for readability
- Use Flyway or Liquibase for schema migrations (not `ddl-auto`)

## Actuator and Observability

- Enable Actuator health checks for all external dependencies
- Use Micrometer for custom metrics
- Configure structured logging (JSON) for production
- Expose `/actuator/health` for load balancer health checks
- Secure actuator endpoints; expose only what's needed

## Security (Spring Security)

- Use `SecurityFilterChain` bean configuration (not `WebSecurityConfigurerAdapter`)
- Apply method-level security with `@PreAuthorize` for fine-grained control
- Store passwords with BCrypt; never roll custom hashing
- Use CORS configuration explicitly; don't disable CSRF without justification

## Async Processing

- Use `@Async` with a custom `TaskExecutor` for background tasks
- Use `@Scheduled` for cron-like jobs with `@EnableScheduling`
- For message-driven processing, use Spring AMQP or Spring Kafka
- Always handle errors in async methods; they don't propagate to callers

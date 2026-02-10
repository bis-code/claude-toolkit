# Django / Python Web Patterns

## Model Design

- **Fat models, thin views**: business logic belongs in models and managers
- Use custom managers and querysets for reusable query logic
- Override `save()` sparingly; prefer model methods for business operations
- Use `Meta.constraints` and `Meta.indexes` for database-level integrity
- Always define `__str__` on models for admin and debugging clarity

## Views and URLs

- Prefer class-based views for CRUD; function-based for custom logic
- Keep views thin: validate input, call service/model, return response
- Use `reverse()` and named URLs; never hardcode paths
- Group URLs by app; use namespaces for clarity

## Signals

- Use signals **sparingly** (they obscure control flow)
- Acceptable uses: cache invalidation, audit logging, cross-app notifications
- Never use signals for core business logic
- Document every signal connection with a comment explaining why

## Serializers (DRF)

- Use serializers for input validation, not just output formatting
- Prefer `ModelSerializer` for simple CRUD; `Serializer` for custom logic
- Validate at the serializer level; raise `ValidationError` with clear messages
- Use nested serializers cautiously; prefer flat responses with IDs

## Async Tasks (Celery)

- Use Celery for anything that takes more than 500ms
- Tasks must be idempotent (safe to retry)
- Pass IDs to tasks, not full objects (objects may be stale)
- Set `acks_late=True` and `reject_on_worker_lost=True` for reliability
- Monitor task queues; alert on growing backlogs

## Database

- Use `select_related` and `prefetch_related` to avoid N+1 queries
- Profile queries with Django Debug Toolbar in development
- Write migrations that are backwards-compatible (zero-downtime deploys)
- Never run data migrations and schema migrations in the same file

## Security

- Use Django's built-in CSRF, XSS, and SQL injection protections
- Always use `get_object_or_404` to prevent information leakage
- Set `AUTH_USER_MODEL` to a custom user model from the start
- Never expose internal IDs in URLs; use UUIDs or slugs

## Settings

- Use environment variables for secrets and environment-specific config
- Split settings: `base.py`, `development.py`, `production.py`
- Never commit secrets; use `django-environ` or `python-decouple`
- Set `DEBUG = False` in production; validate with deployment checklist

## Caching

- Cache expensive queries and computed values with Django's cache framework
- Use cache keys that include the model's `updated_at` timestamp
- Invalidate caches explicitly; don't rely on TTL alone for correctness
- Use `cached_property` for expensive per-request computations

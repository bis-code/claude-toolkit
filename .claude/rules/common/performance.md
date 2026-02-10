# Performance

Measure first, optimize second. Never sacrifice readability for speculative performance gains.

## Core Principles

- **Profile before optimizing** — identify the actual bottleneck with data, not intuition
- **Premature optimization is waste** — write correct, readable code first; optimize when measurements demand it
- **Set performance budgets** — define acceptable latency, memory, and bundle size thresholds upfront

## Database & Queries

- **N+1 query detection** — never execute a query inside a loop; use joins, eager loading, or batch fetches
- **Paginate all list endpoints** — never return unbounded result sets; enforce a max page size
- **Index query patterns** — add indexes for columns used in WHERE, JOIN, and ORDER BY clauses
- **Select only needed columns** — avoid `SELECT *` in production queries
- **Use connection pooling** — configure pool size based on expected concurrency

## API & Network

- **Paginate responses** — use cursor-based pagination for large datasets, offset-based for small ones
- **Compress responses** — enable gzip/brotli for API responses and static assets
- **Set cache headers** — use `Cache-Control`, `ETag`, and `Last-Modified` for cacheable resources
- **Batch related requests** — combine multiple API calls into a single request where possible
- **Timeout external calls** — set explicit timeouts on all HTTP clients and database connections

## Application Code

- **Lazy load expensive resources** — defer initialization until first use
- **Avoid blocking the main thread** — offload CPU-intensive work to background workers or queues
- **Use streaming for large data** — process files and datasets incrementally, not all-at-once in memory
- **Cache expensive computations** — memoize pure function results; invalidate on data change
- **Right data structure for the job** — maps for lookups, arrays for iteration, sets for uniqueness checks

## Frontend (When Applicable)

- **Lazy load routes and heavy components** — split bundles by route
- **Optimize images** — use modern formats (WebP/AVIF), responsive sizes, lazy loading
- **Minimize bundle size** — tree-shake unused code, audit dependencies for bloat
- **Debounce user input** — throttle search, resize, scroll, and keystroke handlers
- **Avoid layout thrashing** — batch DOM reads and writes separately

## Caching Strategy

| Layer | Cache Type | Invalidation |
|-------|-----------|-------------|
| Browser | HTTP cache headers | ETags, max-age |
| CDN | Edge cache | Purge on deploy |
| Application | In-memory / Redis | TTL or event-based |
| Database | Query cache | Schema/data change |

- Cache at the layer closest to the consumer
- Always define an invalidation strategy — cache without invalidation is a bug waiting to happen
- Start with short TTLs and increase based on observed hit rates

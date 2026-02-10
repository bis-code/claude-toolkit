# Docker Best Practices

## Multi-Stage Builds

- Use multi-stage builds to keep final images small
- Separate build dependencies from runtime dependencies
- Name stages for clarity: `FROM node:20 AS builder`
- Copy only artifacts needed at runtime from the build stage

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine AS runtime
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
CMD ["node", "dist/main.js"]
```

## Base Images

- Pin base image versions: `node:20.11-alpine`, not `node:latest`
- Prefer `-alpine` or `-slim` variants for smaller images
- Use `distroless` images for production when possible
- Update base images regularly for security patches

## Layer Optimization

- Order Dockerfile instructions from least to most frequently changing
- Copy dependency files first, install, then copy source code
- Combine related `RUN` commands with `&&` to reduce layers
- Clean up package manager caches in the same `RUN` layer

## .dockerignore

Always include a `.dockerignore` file:

```
node_modules
.git
.env
*.md
dist
coverage
.next
__pycache__
```

## Non-Root User

- Never run containers as root in production
- Create a dedicated user in the Dockerfile

```dockerfile
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
```

## Health Checks

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1
```

- Define health checks for all long-running services
- Use lightweight checks (HTTP ping, TCP connect)
- Set appropriate start period for slow-starting applications

## Docker Compose (Development)

- Use `docker-compose.yml` for local development environments
- Define all service dependencies (database, cache, queue)
- Use volumes for source code mounting (live reload)
- Use `.env` files for environment-specific configuration
- Use `depends_on` with `condition: service_healthy` for startup ordering

## Environment Variables

- Use `ENV` for build-time defaults; override at runtime
- Never hardcode secrets in Dockerfiles
- Use `ARG` for build-time variables that shouldn't persist
- Document required environment variables in a `.env.example` file

## Caching

- Leverage Docker build cache by ordering layers correctly
- Use BuildKit (`DOCKER_BUILDKIT=1`) for faster builds and better caching
- Use `--mount=type=cache` for package manager caches (BuildKit)

```dockerfile
RUN --mount=type=cache,target=/root/.npm npm ci
```

## Image Size

- Remove unnecessary files after installation
- Don't install debugging tools in production images
- Use `.dockerignore` aggressively
- Audit image contents with `docker history` and `dive`

## Logging

- Log to stdout/stderr, not to files
- Use structured logging (JSON) for production
- Let the container runtime handle log aggregation
- Set appropriate log levels via environment variables

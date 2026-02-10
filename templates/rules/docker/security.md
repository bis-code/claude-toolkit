# Docker Security

## Image Scanning

- Scan images for vulnerabilities before deployment
- Use `docker scout`, `trivy`, or `grype` for vulnerability scanning
- Integrate scanning into CI/CD pipeline; fail builds on critical CVEs
- Regularly scan running images; vulnerabilities are discovered post-build
- Pin to specific image digests for reproducibility in production

## Secrets Management

- **Never put secrets in Dockerfiles** (build args are visible in image history)
- **Never put secrets in environment variables in docker-compose.yml** committed to git
- Use Docker Secrets (Swarm) or external secret managers (Vault, AWS SSM)
- Use BuildKit `--mount=type=secret` for build-time secrets
- Use `.env` files only for development; never commit them

```dockerfile
# Build-time secret (BuildKit)
RUN --mount=type=secret,id=npm_token \
    NPM_TOKEN=$(cat /run/secrets/npm_token) npm ci
```

## Filesystem

- Use read-only root filesystem where possible: `--read-only`
- Mount writable volumes only for directories that need writes (tmp, logs, data)
- Use `tmpfs` mounts for ephemeral write needs

```yaml
services:
  app:
    read_only: true
    tmpfs:
      - /tmp
    volumes:
      - app-data:/data
```

## Capabilities

- Drop all capabilities by default; add only what's needed
- Never run with `--privileged` in production

```yaml
services:
  app:
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE  # Only if binding to ports < 1024
```

## Network Isolation

- Use custom bridge networks; avoid the default bridge
- Expose only necessary ports to the host
- Use internal networks for service-to-service communication
- Never expose database ports to the host in production

```yaml
networks:
  frontend:
  backend:
    internal: true  # No external access

services:
  api:
    networks: [frontend, backend]
  db:
    networks: [backend]  # Only accessible from backend network
```

## Resource Limits

- Set memory and CPU limits to prevent resource exhaustion
- Use `pids-limit` to prevent fork bombs
- Set `no-new-privileges` to prevent privilege escalation

```yaml
services:
  app:
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
    security_opt:
      - no-new-privileges:true
    pids_limit: 100
```

## Container Runtime

- Use a minimal base image (Alpine, distroless, scratch)
- Don't install SSH, sudo, or package managers in production images
- Remove setuid/setgid binaries when possible
- Use `--no-cache` when installing packages to avoid caching sensitive data

## Supply Chain Security

- Use official images from verified publishers
- Pin images by digest, not just tag: `node@sha256:abc123...`
- Sign and verify images with Docker Content Trust or cosign
- Use a private registry for internal images; scan on push
- Audit and minimize the number of third-party base images

## Audit and Monitoring

- Log container events (start, stop, exec, attach)
- Monitor for unexpected processes inside containers
- Alert on containers running as root or with elevated privileges
- Use admission controllers (Kubernetes) or policies to enforce standards
- Review Docker daemon configuration against CIS Docker Benchmark

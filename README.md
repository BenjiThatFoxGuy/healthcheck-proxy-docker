# wait-for-health-check

A **single Go binary, ~2MB**, zero-dependency container that watches a TCP target and reports its own Docker HEALTHCHECK status. Use it as a healthcheck proxy so other services can `depends_on` it - even across Docker Compose stacks.

## Why not just use wait-for-it as an entrypoint?

| | wait-for-it (entrypoint) | wait-for-health-check (healthcheck proxy) |
|---|---|---|
| Depends on via `depends_on: condition: service_healthy` | No - it's part of the app, not a separate service | Yes - it advertises its own health |
| Visible in `docker compose ps` as its own service | No | Yes - you can see its status |
| Works as a first-class service in Compose | No | Yes |
| Multiple dependencies | Requires chaining scripts | Drop in another watcher |

## How it works

```text
redis:6379
  │
  ▼
redis-watcher ──► HEALTHY (when redis reachable)
  │
  ▼
app: depends_on redis-watcher: condition: service_healthy
```

## Quick start

The prebuilt image is available on GHCR:

```bash
docker run -d ghcr.io/benjithatfoxguy/healthcheck-proxy-docker:latest
```

For local development or on-demand builds, you can also build from source:

```bash
docker build -t healthcheck-proxy-docker .
```

**Note:** Building from source is only necessary for local development. Use the GHCR image for production and all other use cases.

## Usage

```yaml
services:
  redis:
    image: redis:7-alpine

  redis-watcher:
    image: ghcr.io/benjithatfoxguy/healthcheck-proxy-docker:latest
    environment:
      TARGET_HOST: redis
      TARGET_PORT: "6379"
      TIMEOUT: 2s # per-attempt timeout (default 2s)
    networks: [shared-net]

  app:
    build: .
    depends_on:
      redis-watcher:
        condition: service_healthy
    networks: [shared-net]
```

## Augmenting existing services

Wait-for-health-check is also useful within a single Compose project to add health reporting to services that don't provide their own `HEALTHCHECK` — without forking their image or building a custom sidecar. Drop a watcher alongside any service and use `depends_on: condition: service_healthy` to gate your own workloads:

```yaml
services:
  legacy-service:
    image: my-org/proprietary-app:3.1
    # no HEALTHCHECK provided by the upstream image

  legacy-service-watcher:
    image: ghcr.io/benjithatfoxguy/healthcheck-proxy-docker:latest
    environment:
      TARGET_HOST: legacy-service
      TARGET_PORT: "8080"
    depends_on:
      - legacy-service
    networks: [app-net]

  my-app:
    build: .
    depends_on:
      legacy-service-watcher:
        condition: service_healthy
    networks: [app-net]
```

This works for proprietary images, legacy containers, or any service from a registry you don't control — no fork required.

## Configuration

| Env var | Default | Description |
|---|---|---|
| `TARGET_HOST` | `localhost` | Host to probe |
| `TARGET_PORT` | `80` | Port to probe |
| `TIMEOUT` | `2s` | Per-attempt connect timeout |
| `MAX_WAIT` | `30s` | (wait mode) total time to wait before failing |
| `INTERVAL` | `250ms` | (wait mode) delay between attempts |

## Modes

- **hold** (default container entrypoint): runs forever so the container stays up. Docker's `HEALTHCHECK` runs separately.
- **probe** (used by Docker `HEALTHCHECK`): probes once, exits 0 if reachable, 1 if not.
- **wait** (any other arg, or no args): loops until reachable (or `MAX_WAIT` exceeded), then exits 0/1. Useful for ad-hoc scripts.

## Cross-stack note

Compose can't `depends_on` services defined in another Compose project. This pattern still works across stacks **as long as both stacks attach to a shared external Docker network** (so the watcher can resolve/reach the target).

## Building

```bash
# Requires Docker (no local Go needed)
docker build -t wait-for-health-check .
```

Final image is `scratch` (empty) - just the static Go binary. ~2MB.

## License

MIT

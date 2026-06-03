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

```
redis:6379
    │
    ▼
redis-watcher  ──►  HEALTHY (when redis reachable)
    │
    ▼
app: depends_on redis-watcher: condition: service_healthy
```

## Usage

```yaml
services:
  redis:
    image: redis:7-alpine

  redis-watcher:
    build: .
    environment:
      TARGET_HOST: redis
      TARGET_PORT: "6379"
      TIMEOUT: 2s       # per-attempt timeout (default 2s)
    networks: [shared-net]

  app:
    build: .
    depends_on:
      redis-watcher:
        condition: service_healthy
    networks: [shared-net]
```

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


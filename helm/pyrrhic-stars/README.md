# Pyrrhic Stars Helm chart

Deploys the Pyrrhic Stars backend and web surfaces to Kubernetes:

| Component     | Source                | Port(s)        | Default |
| ------------- | --------------------- | -------------- | ------- |
| `gateway`     | `server/` (Go)        | 7777 TCP/WS, 7778 UDP | enabled |
| `landing`     | `web/landing/` (Astro SSR) | 4321      | enabled |
| `combat-logs` | `web/combat-logs/` (nginx) | 80        | enabled |
| `zone`        | `server/` (Go)        | 8081           | disabled (stub) |
| `chat`        | `server/` (Go)        | 8082           | disabled (stub) |

The `gateway`, `zone`, and `chat` binaries all live in the **same server image**
(`server/Dockerfile`) and are selected via the container `command`. `zone` and
`chat` have empty `main.go` today, so they are disabled by default.

## Backing services are external

This chart does **not** deploy Postgres, Redis, ClickHouse, or Ory Kratos. Run
them separately (managed services, Bitnami charts, operators) and wire them in
through `config`:

```yaml
config:
  redis:
    addr: my-redis.default.svc:6379
  postgres:
    driver: postgres
    dsn: postgres://user:pass@my-pg:5432/codex?sslmode=disable
  clickhouse:
    addr: my-clickhouse.default.svc:9000   # empty disables combat-log persistence
    database: codex
    user: codex
    password: changeme
  kratos:
    publicUrl: http://kratos.default.svc:4433
```

Sensitive values (`POSTGRES_DSN`, `CLICKHOUSE_PASSWORD`) are rendered into a
Secret. To manage credentials yourself, create a Secret with those keys and set
`existingSecret: my-secret`.

## Images

CI (`.github/workflows/ci.yml`) builds and pushes all three images to GHCR on
every push to `main`, tagged with the commit SHA and `latest`:

- `ghcr.io/lanathlor/pyrrhic-stars/gateway` (server image; ships gateway/zone/chat)
- `ghcr.io/lanathlor/pyrrhic-stars/landing`
- `ghcr.io/lanathlor/pyrrhic-stars/combat-logs`

`image.registry` is the shared prefix; each component appends its repository. The
tag defaults to `image.tag` (`latest`), or set a per-image tag. Pin to a SHA in
production:

```yaml
image:
  registry: ghcr.io/lanathlor/pyrrhic-stars
  tag: "<git-sha>"        # applies to all three unless overridden per image
  server:     { repository: gateway }
  landing:    { repository: landing }
  combatLogs: { repository: combat-logs }
```

## Networking

The WS/TCP path and the UDP path are deliberately separate Services:

- **WebSocket (TCP 7777)**: front the `gateway` Service with the Ingress for
  WSS + TLS. The chart sets `proxy-read/send-timeout: 3600` so ingress-nginx
  doesn't sever long-lived upgrades at its 60s default.
- **UDP (7778)**: cannot traverse an HTTP Ingress, and folding it into the TCP
  Service forces a fragile mixed-protocol LoadBalancer. Instead, `gateway.udpService`
  renders a dedicated UDP-only `LoadBalancer` Service (`-gateway-udp`). Add
  `external-dns` annotations / `loadBalancerIP` for a stable client endpoint.

The `combat-logs` pod proxies `/api/` to the in-cluster `gateway` Service via a
chart-rendered nginx config (the image's baked `gateway:7777` upstream is the
docker-compose hostname and does not resolve in-cluster).

Enable Ingress with host-based routing:

```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - { host: pyrrhicstars.com,      service: landing,    paths: [{ path: /, pathType: Prefix }] }
    - { host: api.pyrrhicstars.com,  service: gateway,    paths: [{ path: /, pathType: Prefix }] }
    - { host: logs.pyrrhicstars.com, service: combatLogs, paths: [{ path: /, pathType: Prefix }] }
  tls:
    - secretName: pyrrhicstars-tls
      hosts: [pyrrhicstars.com, api.pyrrhicstars.com, logs.pyrrhicstars.com]
```

## Install

```sh
helm lint helm/pyrrhic-stars
helm install pyrrhic-stars helm/pyrrhic-stars -n pyrrhic-stars --create-namespace \
  -f my-values.yaml
```

Render without installing:

```sh
helm template pyrrhic-stars helm/pyrrhic-stars -f my-values.yaml
```

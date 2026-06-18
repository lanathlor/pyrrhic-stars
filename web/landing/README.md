# Pyrrhic Stars: Landing Page

Public marketing site + devlog for Pyrrhic Stars. Built with Astro 6 (Node SSR),
Tailwind 4, content collections, and a single Postgres-backed email signup
endpoint.

## Layout

- `src/pages/index.astro`: hero, features, classes, devlog preview, subscribe
- `src/pages/devlog/`: devlog index + individual posts (content collection)
- `src/pages/api/subscribe.ts`: POST email signup, writes to Postgres
- `src/content/devlog/*.md`: markdown posts, frontmatter validated by zod
- `src/lib/db.ts`: `pg.Pool` singleton + `CREATE TABLE IF NOT EXISTS` on boot
- `src/lib/classes.ts`: class data, mirrors `docs/design/classes/README.md`

## Develop

```sh
pnpm install
pnpm run dev          # http://localhost:4321
```

Or via docker-compose at the repo root:

```sh
docker compose up landing postgres
```

## Build & verify

```sh
pnpm run check        # astro check (type + content schema)
pnpm run build        # → dist/server/entry.mjs + dist/client/
pnpm run preview      # serve the build locally
```

## Docker

```sh
docker build -t pyrrhic-stars-landing:dev .
docker run --rm -p 4321:4321 \
  -e POSTGRES_DSN=postgres://codex:codex@host.docker.internal:5432/codex \
  pyrrhic-stars-landing:dev
```

## Environment variables

Public (compiled into prerendered pages; must start with `PUBLIC_`):

| Var                       | Purpose                                  | Default        |
|---------------------------|------------------------------------------|----------------|
| `PUBLIC_SITE_URL`         | Canonical origin for OG, sitemap, RSS    | `http://localhost:4321` |
| `PUBLIC_DISCORD_URL`      | Discord invite link                      | falls back to committed default in `src/consts.ts` |
| `PUBLIC_REPO_URL`         | Public source repository URL             | falls back to committed default in `src/consts.ts` |
| `PUBLIC_DOWNLOAD_LINUX`   | Linux build download URL                 | unset → button disabled |
| `PUBLIC_DOWNLOAD_WINDOWS` | Windows build download URL               | unset → button disabled |
| `PUBLIC_STEAM_URL`        | Steam wishlist URL                       | unset → button hidden   |

Server-only:

| Var            | Purpose                                | Default |
|----------------|----------------------------------------|---------|
| `POSTGRES_DSN` | Postgres connection string for signups | unset → /api/subscribe returns 503 |
| `HOST`         | Bind address                           | `0.0.0.0` |
| `PORT`         | Bind port                              | `4321`    |

## Subscribe endpoint

`POST /api/subscribe` accepts `application/x-www-form-urlencoded` or JSON:

```
email=user@example.com&consent=true
```

- 202 on accept (insert is fire-and-forget, idempotent on email)
- 400 on missing/invalid email or missing consent
- 429 on rate limit (5 / 15 min / IP, in-memory)
- 503 if `POSTGRES_DSN` is unset

Honeypot field `company`: if present and non-empty, the response is 200 and no
write happens. Real users never fill it; bots usually do.

## Conventions

- No "MMO" appears in any rendered page (per `docs/project/marketing.md`).
- Class accent colors are muted; chrome stays grey-blue (per `docs/design/ui-language.md`).
- All visible buttons have a graceful "disabled" state when their env var is unset.

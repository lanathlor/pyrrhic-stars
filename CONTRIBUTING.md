# Contributing to Pyrrhic Stars

Thanks for your interest in contributing. Pyrrhic Stars is an action MMO where every
class plays a different game genre, built with a Godot 4 client and a Go server. This
is a solo project being developed in the open, so contributions, bug reports, and
ideas are all welcome.

## License of contributions

This project is licensed under the **GNU Affero General Public License v3.0** (see
[`LICENSE`](LICENSE)). By submitting a contribution (code, assets, docs, or anything
else), you agree that your work is licensed under the AGPL-3.0 and that you have the
right to license it that way.

If you contribute third-party assets, they must be under an AGPL-compatible license,
and you must add an entry to [`ATTRIBUTION.md`](ATTRIBUTION.md) with the author,
source, license, and where the asset is used.

## Development environment

The project ships a Nix flake that pins every tool (Go, Godot 4, Blender, and
friends).

- With [direnv](https://direnv.net/): `direnv allow` and the shell loads automatically.
- Without direnv: `nix develop`.

If you prefer not to use Nix, you will need to install the equivalents yourself:

- **Go** 1.25+
- **Godot** 4 (used both as the client and as the headless e2e runner)
- **Blender** (only for asset work)
- [**just**](https://github.com/casey/just) as the task runner
- **Docker** + Docker Compose (for Redis, Postgres, and Kratos auth)
- **Git LFS** (binary assets are stored via LFS)

### First-time setup

```sh
just setup   # points core.hooksPath at .githooks (installs the pre-commit checks)
```

This is important: the pre-commit hook runs lint, format checks, and tests on the
files you stage, which catches most CI failures before you push.

## Running the project

```sh
just up          # bring up the full stack (server + infra) via Docker Compose
just up-infra    # just Redis + Postgres
just up-auth     # Kratos auth for local login
just web         # export the client to web and serve it
just logs        # follow container logs
```

For the native client, open `client/` in the Godot editor and run it.

## Tests

End-to-end scenarios run a live server and drive the Godot client (headless by
default):

```sh
just e2e test_connect_hub
just e2e test_zone_cycle,test_connect_hub
just e2e test_zone_cycle --headed   # watch it run in a window
```

Server unit tests and client tests run through the `just server test` and
`just client test` recipes, which the pre-commit hook invokes automatically for
changed files.

When fixing a bug, please follow a **red/green** flow: write a failing test that
reproduces the bug first, then make it pass.

## Code style

Style is enforced by tooling, not by hand. The pre-commit hook runs:

- Go: `just server lint`, `just server fmt-check`, `just server test`
- GDScript: `just client lint`, `just client fmt-check`, `just client test`

**Do not bypass the hooks** (no `git commit --no-verify`). If a check fails, fix the
underlying issue rather than skipping it.

A few project conventions worth knowing:

- **No em dashes** in user-facing copy. Use a hyphen, colon, middle dot, or restructure.
- **Terminology**: User = account, Character = avatar, Player = the in-world entity.
- **Level geometry** lives in `shared/levels/*.json`, exported from Godot scenes -
  never hand-edit those JSON files; edit the scene and re-export.
- The server is **server-authoritative**; gameplay logic and validation belong on the
  server, not the client.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/), matching the
existing history:

```
feat(web): set the Discord invite as a committed default
docs(class): describe Blade Dancer as a combo fighter
refactor(web): rewrite landing copy in a plainer voice
chore: add .secrets to gitignore
```

Format: `type(scope): summary`. Common types are `feat`, `fix`, `docs`, `refactor`,
`chore`, `test`. Keep the summary in the imperative mood.

## Pull requests

1. Fork the repo and create a branch off `main`.
2. Run `just setup` so the hooks are active.
3. Make your change, with tests where it makes sense.
4. Make sure the pre-commit checks pass (lint, format, tests).
5. Open a PR describing what changed and why. Screenshots or short clips are very
   helpful for anything visual.

## Questions

Open an issue, or join the Discord (linked from the landing page) to ask before
starting on anything large, so we can make sure it fits the direction of the project.

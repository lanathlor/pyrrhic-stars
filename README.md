# Pyrrhic Stars

**A co-op action game where every class plays a different genre.**

Five of you take on the same dungeon and the same boss, but what you are actually
doing depends on the class you picked. The Gunner is playing a first-person
shooter. The Vanguard is in a Souls-like, reading the boss for an opening. The
Blade Dancer is working through a state machine of blade configs. Same fight, and
none of you are really playing the same game.

This is a PvE co-op action MMO, server-authoritative and built in the open. It is
very early: an extremely rough early access of some sorts. The server and systems
are further along than the art, because I come from backend development rather
than games.

> Not a WoW clone. Not tab-target. Not the traditional tank/healer/DPS trinity.
> Every point of damage a boss deals should be something a player could have
> avoided by playing better.

## The idea

I have played co-op games for about twenty years, and they nearly all work the
same way. You bring a tank, a healer, and some damage, and whatever class you
picked, the job underneath was the same one: be in the right place and hit your
buttons without dying. Your class decided which buttons. It almost never decided
what the game actually felt like to play. That is the part this project tries to
break.

Inspirations are not hard to spot: World of Warcraft, Furi, and the Souls games.

## Classes

Each class plays a fundamentally different game genre. The camera, input model,
HUD, and core gameplay loop differ per class.

| Class            | Genre                       | Flux Usage |
| ---------------- | --------------------------- | ---------- |
| Gunner           | First-Person Shooter        | Minimal    |
| Vanguard         | Souls-like Action Melee     | Minimal    |
| Arcanotechnicien | Tactical Flux Channeling    | Primary    |
| Engineer         | Deployable Management       | Moderate   |
| Blade Dancer     | Positional Combo Fighter    | Moderate   |
| Tutelaire        | Aura Positioning and Triage | Heavy      |

## Setting

A sci-fi military universe. Humanity is organized into an Empire.
"Arcanotechnique" is a fusion of magic and technology powered by an energy called
**le Flux**. Flux is a fundamental physical energy, not mystical magic: a fireball
produces real heat, a rock projectile produces real impact. The source does not
matter, the physical effect does.

The world is based on the [Codex RPG - Arcanotechnique](https://github.com/lanathlor/rpg)
tabletop universe.

## How progression will work

In v1, the goal is four progress tracks, each its own way to play:

-   **Mercenary** — Mythic+ style dungeons.
-   **Paragon** — raids.
-   **Hero** — Monster Hunter style: solo or small-group fights against one big boss.
-   **Explorer** — open-world survival.

Each track awards the gear that is best for that track. No levels, no endless
grind, no fetch quests, and nothing drops on a dice roll. You clear something, it
pays out in tokens, and you spend those on the gear you actually want.

## Tech stack

-   **Client** — Godot 4 (GDScript), per-class controllers over a shared character node.
-   **Server** — Go game server: gateway and zone services, WebSocket transport, a
    20Hz tick loop, server-authoritative combat with client prediction and reconciliation.
-   **Infra** — Redis (hot state), PostgreSQL (persistence), Ory Kratos (auth).
-   **Assets** — Blender workspace, CC0 packs and Mixamo, stored via Git LFS.
-   **Dev env** — a Nix flake pins every tool.

## Repo structure

```
pyrrhic-stars/
├── client/    # Godot 4 project (GDScript)
├── server/    # Go game server (gateway, zone, chat services)
├── blender/   # Blender asset workspace (models, props, kits)
├── shared/    # shared data (level JSON, etc.)
├── web/       # landing page (Astro)
├── docs/      # documentation tree (see docs/INDEX.md)
├── ops/       # operational tooling
├── flake.nix  # Nix dev shell (godot_4 + blender + go + ...)
└── justfile   # task runner
```

## Getting started

The project ships a Nix flake that pins every tool (Go, Godot 4, Blender, and
friends).

-   With [direnv](https://direnv.net/): `direnv allow` and the shell loads automatically.
-   Without direnv: `nix develop`.

If you prefer not to use Nix, you will need Go 1.25+, Godot 4, Blender (for asset
work), [just](https://github.com/casey/just), Docker + Docker Compose, and Git LFS.

```sh
just setup       # point core.hooksPath at .githooks (installs pre-commit checks)
just up          # bring up the full stack (server + infra) via Docker Compose
just up-infra    # just Redis + Postgres
just up-auth     # Kratos auth for local login
just web         # export the client to web and serve it
just logs        # follow container logs
```

For the native client, open `client/` in the Godot editor and run it.

### Tests

End-to-end scenarios run a live server and drive the Godot client (headless by
default):

```sh
just e2e test_connect_hub
just e2e test_zone_cycle --headed   # watch it run in a window
```

Server and client unit tests run through `just server test` and `just client
test`, which the pre-commit hook invokes automatically for changed files.

## Project status

Currently in **Phase 0 (Proof of Concept)**: five players in a hub, pick a class,
group up, walk into a dungeon, fight one real boss, leave. The goal of this phase
is to answer one question: does multi-gameplay-mode co-op actually feel good?

See [`ROADMAP.md`](ROADMAP.md) for detailed progress and the
[documentation index](docs/INDEX.md) for design and architecture docs.

## Contributing

Anyone is welcome. The whole thing is open source, and the spots where I need help
most are art and animation, world detailing, sound and music, code, and honest
feedback on what feels good and what falls flat.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the full guide, and
[`ATTRIBUTION.md`](ATTRIBUTION.md) for third-party asset credits.

Say hi on [Discord](https://discord.gg/UD5cChCGtd).

## License

Pyrrhic Stars is licensed under the **GNU Affero General Public License v3.0**.
See [`LICENSE`](LICENSE).

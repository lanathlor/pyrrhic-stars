# Codex Online - Documentation Index

Project codename for the action MMO based on the "Codex RPG - Arcanotechnique" TTRPG universe.
TTRPG source material: https://github.com/lanathlor/rpg

## Game Design

- [Vision & Setting](design/vision.md) — core pitch, sci-fi universe, Flux as physical energy
- [Combat Principles](design/combat.md) — no threat table, all damage avoidable, protect the caster
- [HUD Graphical Language](design/ui-language.md) — minimalist MMO HUD rules, ElvUI-inspired but not cloned
- [UI Screens & Menus](design/ui-screens.md) — pause, menu, character, group, and prompt design rules

### Core Systems

- [Flux System](design/systems/flux.md) — reserve/afflux/recovery, commitment, channeling, spell slots
- [Resistance System](design/systems/resistance.md) — RMEC/RRAD/RINT, physical-nature-based damage
- [Affinity System](design/systems/affinity.md) — general + specific affinities, progression
- [Defense System](design/systems/defense.md) — Score de Defense formula
- [Stat System](design/systems/stats.md) — Hull/Output/Plating + Tempo/Identity/Mastery per class
- [Item System](design/systems/items.md) — 6 slots, 3 stat lines each, deterministic gearing
- [Overflux System](design/systems/overflux.md) — modular difficulty conditions, player-composed loadouts, reward tiers
- [Seasonal System](design/systems/seasons.md) — decay, season clock, token economy

### Classes (each plays a different game genre)

- [Overview](design/classes/README.md) — 6 classes summary table
- [Gunner](design/classes/gunner.md) — FPS (Assault / Marksman / Chasseur)
- [Vanguard](design/classes/vanguard.md) — Souls-like (Blade / Shield / Shadow)
- [Arcanotechnicien](design/classes/arcanotechnicien.md) — Tactical caster (Destroyer / Battlemage / Harmonist)
- [Engineer](design/classes/engineer.md) — Deployables (Architect / Pilot / Saboteur)
- [Blade Dancer](design/classes/blade-dancer.md) — State machine (Single / Multi Blade)
- [Tutelaire](design/classes/tutelaire.md) — Aura positioning (Guardian / Adjudicator)

### Content

- [Dungeons](design/content/dungeons.md) — handcrafted, modular kits, first dungeon: Derelict City
- [Content Tracks](design/content/tracks.md) — M+, Adventure, Explorer, Raid — universal track pattern
- [Long-term Vision](design/content/long-term.md) — space, PvP, open world, crafting (NOT phase 0-2)

## Technical Architecture

- [Domain Model](architecture/domain-model.md) — User vs Character vs Player distinction
- [Stack & Server](architecture/overview.md) — Go server, Redis/PostgreSQL, k3s, scaling targets
- [Client Architecture](architecture/client.md) — per-class Godot controllers, shared character node
- [Networking Model](architecture/networking.md) — client-predicted, server-authoritative
- [Zone Sharding & AOI](architecture/sharding.md) — seamless world sharding, ghost entities, transport abstraction
- [AI & Encounter System](architecture/ai.md) — BT executor, entity context, threat awareness, encounter definitions
- [AI Testing & Balance](architecture/testing.md) — three test modes, encounter specs, fuzz simulation
- [AI Long-term Vision](architecture/ai-vision.md) — full entity context API, ML training, OSS extraction
- [Combat Logging & Observer](architecture/combat_logs.md) — event logging, replay, spectator, fight export

## Pipeline

- [Asset Pipeline](pipeline/assets.md) — Blender, CC0 packs, Mixamo, workflow, art direction
- [Testing Strategy](pipeline/testing.md) — Go tests, GdUnit, integration tests, navmesh validation
- [Level Markers Guide](pipeline/level-markers-guide.md) — Godot node groups/metadata → JSON for Go server

## Project

- [Development Phases](project/phases.md) — Phase 0 checklist, Phase 1-3 roadmap
- [Monetization](project/monetization.md) — timeline, revenue expectations
- [Marketing](project/marketing.md) — build in public, key clip, Steam wishlists
- [Alpha/Beta Wipes](project/wipes.md) — wipe policy, what persists, communication

## Repo Structure

```
mmorpg/
├── client/          # Godot 4 project (GDScript)
├── server/          # Go game server (gateway, zone, chat services)
├── blender/         # Blender asset workspace (models, props, kits)
├── docs/            # This documentation tree
├── flake.nix        # Nix dev shell (godot_4 + blender)
```

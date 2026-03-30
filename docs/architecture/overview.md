# Technical Architecture

## Stack

- **Game Server**: Go (controller/service/repo pattern)
- **Game Client**: Godot 4 (GDScript)
- **Hot State Store**: Redis (player positions, combat state, Flux, buffs)
- **Persistent Store**: PostgreSQL (characters, inventory, progression, leaderboards)
- **Infrastructure**: k3s (Kubernetes), Ceph (storage), Helm charts
- **Asset Pipeline**: Blender + free CC0 asset packs + Godot editor
- **CI/CD**: GitLab (or GitHub Actions)

## Server Architecture

```
                    +----------------+
                    |    Gateway     |
                    |  (stateless)   |
                    +-------+--------+
                            |
              +-------------+-------------+
              |             |             |
        +-----v-----+ +----v------+ +----v------+
        |  Zone Svc  | | Zone Svc  | | Zone Svc  |
        | (Hub Area) | | (Dungeon) | | (Dungeon) |
        +-----+------+ +----+------+ +----+------+
              |             |             |
              +-------------+-------------+
                            |
                    +-------v--------+
                    |     Redis      |
                    |  (game state)  |
                    +-------+--------+
                            |
                    +-------v--------+
                    |   PostgreSQL   |
                    | (persistence)  |
                    +----------------+
```

Each zone/dungeon instance is a separate Go process with its own tick loop. Stateful within itself, stateless from infrastructure perspective. All persistent state in Redis (hot) and PostgreSQL (cold).

Zone handoff: player transitions between zones go through gateway. Old zone writes final state to Redis, new zone reads it. Brief loading screen.

Dungeon instances: spawned on demand as k3s pods. Destroyed when party exits.

Cross-zone services (chat, party, guild, friends): separate stateless Go services using Redis pub/sub.

## Scaling Targets

| Scale | Concurrent | Infrastructure | Cost |
|---|---|---|---|
| Friends | 10-50 | Homelab k3s | Electricity |
| Early access | 1,000-5,000 | Small managed k8s (Hetzner/Scaleway) | 200-500 EUR/month |
| Growth | 20,000-50,000 | Multi-node k8s, Redis Cluster, PG replicas | 2,000-8,000 EUR/month |

Architecture supports all tiers without redesign. Just add pods.

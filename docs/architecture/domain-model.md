# Domain Model

Three distinct levels represent a person playing the game:

## User

**Package:** `persistence.User`

A **User** is a real person's account. It lives in the database and holds identity fields: UUID, username, and later password, email, etc. One User can own many Characters.

- Identified by a client-generated UUID (primary key)
- Created on first connection, never deleted
- Username is set once at creation (not updatable yet)

## Character

**Package:** `persistence.Character`

A **Character** is the persistent, "cold" representation of a playable avatar. It belongs to a User and stores class, name, and last-known hub position. It lives in the database and is loaded/saved across sessions.

- Identified by auto-increment ID
- Belongs to a User via `UserID` foreign key
- Stores: class, display name, last hub position (X/Y/Z + rotation)
- One User can have up to 100 Characters
- Character names are globally unique

## Player

**Package:** `entity.Player`

A **Player** is the hot, transient, in-game entity for a currently connected and playing Character. It exists only in memory while the user is in a zone and is destroyed on disconnect or zone leave.

- Identified by `PeerID` (uint16, allocated per zone)
- Stores: position, velocity, rotation, health, combat state, class-specific mechanics, animation state
- Created when a Character enters a zone, destroyed when they leave
- No direct reference to `persistence.User` or `persistence.Character` — the `session.Session` bridges them

## Session

**Package:** `session.Session`

The **Session** bridges all three layers for a single WebSocket connection:

```
Session
├── ID          uint32   — connection-scoped identifier
├── UserUUID    string   — persistent User identity
├── Username    string   — from User (account-level)
├── CharID      uint     — selected Character (persistence PK)
├── CharName    string   — Character display name
├── Class       string   — Character class
├── ZoneID      string   — current zone
└── PeerID      uint16   — zone-local Player entity ID
```

## Relationships

```
persistence.User (account, cold)
    │
    │  1 : N  (via UserID)
    ▼
persistence.Character (avatar, cold)
    │
    │  selected via Session
    ▼
entity.Player (in-game entity, hot)
    ▲
    │  bridges all three
session.Session (per-connection)
```

## Naming Rules

| Term      | Means                          | Where it lives | Lifetime          |
|-----------|--------------------------------|----------------|-------------------|
| User      | The person / account           | Database        | Permanent         |
| Character | A saved avatar                 | Database        | Permanent         |
| Player    | A live, in-game entity         | Memory          | Zone session      |
| Session   | A WebSocket connection context | Memory          | Connection        |

When naming variables, parameters, struct fields, and log keys:
- `userUUID`, `userID` — refers to the User account
- `charID`, `charName` — refers to the Character
- `peerID`, `player` — refers to the in-game Player entity
- `sess`, `sessionID` — refers to the connection Session

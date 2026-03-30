# Networking Model

Client-server authoritative. Server runs the simulation. Client does prediction for responsiveness.

- Movement: client-predicted, server-reconciled
- Abilities: client sends intent, server validates and executes
- Hit detection: server-authoritative for melee/AoE, hybrid for hitscan (client sends ray, server validates line of sight)
- State sync: server pushes deltas to clients per tick
- Protocol: WebSocket for reliability or UDP with custom reliability layer for performance (evaluate in Phase 1)

# Testing Strategy

## Server Tests (Go)

Standard Go `testing` package. Test-driven development for all game logic.

- Unit tests for Flux system, resistance calculations, affinity validation, damage formulas
- Integration tests for zone handoff, dungeon instance lifecycle, persistence
- SOAP/HTTP test harness for automated encounter validation

## Client Tests (Godot)

GdUnit for GDScript unit testing. Headless Godot for automated testing.

- Unit tests for the Blade Dancer configuration system, aura management, input processing
- Scene tests: programmatic assertions on node visibility, interaction triggers, UI state

## Integration Tests

Go test runner that spins up a server instance, connects simulated clients, and runs through scenarios:

- Character creation, ability usage, damage calculation verification
- Boss phase transitions at correct HP thresholds
- Flux channeling and interruption timing
- Cross-class interaction (does the Tutelaire aura actually reduce Gunner damage taken?)

## Navmesh Validation (later)

C++ or Go module that validates all spawn points are on valid navmesh, reachable, and not inside terrain. Runs in CI on every content commit.

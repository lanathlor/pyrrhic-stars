package system

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
)

func makeHubWorld(players map[uint16]*entity.Player) *World {
	return &World{
		ZoneType:      0, // hub
		TickNum:       100,
		State:         StateLobby, // hub never enters StateFight
		Players:       players,
		Level:         level.NewHubLevel(),
		AbilityEngine: ability.NewEngine(nil),
	}
}

func abilityPayload(action uint8) []byte {
	return codec.EncodeAbilityInput(action, 0.0)
}

// ---------------------------------------------------------------------------
// Dodge — works in any zone state
// ---------------------------------------------------------------------------

func TestDodge_ConsumesStamina_InHub(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	before := p.Resources["stamina"].Current
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Resources["stamina"].Current != before-20.0 {
		t.Errorf("stamina = %f, want %f (dodge costs 20)", p.Resources["stamina"].Current, before-20.0)
	}
	if p.Resources["stamina"].DelayTimer != 0.6 {
		t.Errorf("stamina delay = %f, want 0.6", p.Resources["stamina"].DelayTimer)
	}
}

func TestDodge_BlockedWhenInsufficientStamina(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Resources["stamina"].Current = 10.0
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Resources["stamina"].Current != 10.0 {
		t.Errorf("stamina = %f, want 10.0 (insufficient stamina)", p.Resources["stamina"].Current)
	}
	if p.Invincible {
		t.Error("should not grant i-frames with insufficient stamina")
	}
}

func TestDodge_RepeatedDrainsStamina(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})
	is := &InputSystem{}

	for i := 0; i < 5; i++ {
		w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
		is.Tick(w, 0.05)
	}

	if p.Resources["stamina"].Current != 0.0 {
		t.Errorf("stamina = %f after 5 dodges, want 0.0", p.Resources["stamina"].Current)
	}

	// 6th should be blocked — no stamina, no i-frames
	p.Invincible = false // reset from previous dodge
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is.Tick(w, 0.05)

	if p.Resources["stamina"].Current != 0.0 {
		t.Errorf("stamina = %f after 6th dodge, want 0.0", p.Resources["stamina"].Current)
	}
	if p.Invincible {
		t.Error("6th dodge should not grant i-frames (no stamina)")
	}
}

func TestDodge_DoesNotIncrementOnslaught(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	// Enemy in front of player (player faces -Z by default)
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -2.0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	is := &InputSystem{}

	// Build onslaught via melee hit, then reset to 0
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionMelee)}}
	is.Tick(w, 0.05)

	type resettable interface{ Reset() }
	if r, ok := p.AbilityState["onslaught"].(resettable); ok {
		r.Reset()
	}
	// Clear cooldown so state is clean
	delete(p.Cooldowns, "cleave")

	// Dodge — should NOT increment onslaught
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is.Tick(w, 0.05)

	type stacker interface{ StackCount() int }
	if s, ok := p.AbilityState["onslaught"].(stacker); ok {
		if s.StackCount() != 0 {
			t.Errorf("onslaught stacks = %d after dodge, want 0 (dodge should not build onslaught)", s.StackCount())
		}
	}
}

func TestDodge_PreservesOnslaughtStacks(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	// Enemy in front of player (player faces -Z by default)
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -2.0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	is := &InputSystem{}

	// Build onslaught via melee hit
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionMelee)}}
	is.Tick(w, 0.05)

	type stacker interface{ StackCount() int }
	ons, ok := p.AbilityState["onslaught"].(stacker)
	if !ok || ons.StackCount() == 0 {
		t.Fatal("melee should have built onslaught stacks")
	}
	stacksBefore := ons.StackCount()

	// Dodge — stacks should be preserved (not reset, not incremented)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is.Tick(w, 0.05)

	if ons.StackCount() != stacksBefore {
		t.Errorf("onslaught stacks = %d after dodge, want %d (dodge should preserve stacks)", ons.StackCount(), stacksBefore)
	}

	// Take damage during i-frames — stacks should still be preserved
	p.ApplyDamage(50)
	if ons.StackCount() != stacksBefore {
		t.Errorf("onslaught stacks = %d after damage during i-frames, want %d", ons.StackCount(), stacksBefore)
	}
}

func TestDodge_GrantsIFrames(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if !p.Invincible {
		t.Error("dodge should grant i-frames")
	}
	if p.InvincibleTimer != 0.15 {
		t.Errorf("invincible timer = %f, want 0.15", p.InvincibleTimer)
	}
}

func TestDodge_IFramesBlockDamage(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	// Dodge to get i-frames
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Damage should be absorbed during i-frames
	dmg := p.ApplyDamage(50)
	if dmg != 0 {
		t.Errorf("damage during i-frames = %f, want 0", dmg)
	}
	if p.Health != p.MaxHealth {
		t.Errorf("health = %f, want %f (no damage during i-frames)", p.Health, p.MaxHealth)
	}
}

func TestDodge_IFramesExpireAfterTimer(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	// Dodge to get i-frames
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Tick past the i-frame duration (0.15s = 3 ticks at 0.05s)
	for i := 0; i < 4; i++ {
		w.AbilityEngine.TickPlayer(p, 0.05, nil)
	}

	if p.Invincible {
		t.Error("i-frames should expire after 0.15s")
	}

	// Damage should now go through
	dmg := p.ApplyDamage(50)
	if dmg == 0 {
		t.Error("damage after i-frames expired should not be 0")
	}
}

func TestDodge_DeadPlayerCannotDodge(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Alive = false
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Resources["stamina"].Current != 100.0 {
		t.Errorf("dead player stamina = %f, want 100 (no cost)", p.Resources["stamina"].Current)
	}
	if p.Invincible {
		t.Error("dead player should not get i-frames")
	}
}

// ---------------------------------------------------------------------------
// Combat abilities — work in hub (no enemies = no damage, but stamina/cooldowns apply)
// ---------------------------------------------------------------------------

func TestMelee_ConsumesStamina_InHub(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	before := p.Resources["stamina"].Current
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionMelee)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Resources["stamina"].Current != before-10.0 {
		t.Errorf("stamina = %f, want %f (melee costs 10 in hub)", p.Resources["stamina"].Current, before-10.0)
	}
	if p.Cooldowns["cleave"] <= 0 {
		t.Error("melee cooldown should be set after melee in hub")
	}
}

func TestHeavy_ConsumesStamina_InHub(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	before := p.Resources["stamina"].Current
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionHeavy)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Resources["stamina"].Current != before-20.0 {
		t.Errorf("stamina = %f, want %f (heavy costs 20 in hub)", p.Resources["stamina"].Current, before-20.0)
	}
}

func TestShoot_WorksInHub(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionShoot)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Cooldowns["fire_shot"] <= 0 {
		t.Error("shoot should set fire cooldown in hub")
	}
}

// ---------------------------------------------------------------------------
// Full pipeline: InputSystem + CombatSystem together
// ---------------------------------------------------------------------------

func TestDodge_FullPipeline_Hub(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	inputSys := &InputSystem{}
	combatSys := &CombatSystem{}

	if p.Resources["stamina"].Current != 100.0 {
		t.Fatalf("initial stamina = %f, want 100.0", p.Resources["stamina"].Current)
	}

	// Tick 1: dodge
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	inputSys.Tick(w, 0.05)
	combatSys.Tick(w, 0.05)

	if p.Resources["stamina"].Current != 80.0 {
		t.Errorf("tick 1: stamina = %f, want 80.0", p.Resources["stamina"].Current)
	}

	// Ticks 2-5: regen blocked by delay
	for i := 2; i <= 5; i++ {
		inputSys.Tick(w, 0.05)
		combatSys.Tick(w, 0.05)
	}

	if p.Resources["stamina"].Current != 80.0 {
		t.Errorf("tick 5: stamina = %f, want 80.0 (delay still active)", p.Resources["stamina"].Current)
	}

	// Ticks 6-13: delay expires, regen starts
	for i := 6; i <= 13; i++ {
		inputSys.Tick(w, 0.05)
		combatSys.Tick(w, 0.05)
	}

	if p.Resources["stamina"].Current <= 80.0 {
		t.Errorf("tick 13: stamina = %f, want > 80.0 (regen should have started)", p.Resources["stamina"].Current)
	}
	if p.Resources["stamina"].Current >= 100.0 {
		t.Errorf("tick 13: stamina = %f, want < 100.0", p.Resources["stamina"].Current)
	}
}

// ---------------------------------------------------------------------------
// Combat abilities — work in StateFight with enemies
// ---------------------------------------------------------------------------

func TestMelee_ConsumesStamina_InFight(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0, Z: 2.0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})

	before := p.Resources["stamina"].Current
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionMelee)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Resources["stamina"].Current != before-10.0 {
		t.Errorf("stamina = %f, want %f (melee costs 10)", p.Resources["stamina"].Current, before-10.0)
	}
}

func TestHeavy_ConsumesStamina_InFight(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0, Z: 2.0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})

	before := p.Resources["stamina"].Current
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionHeavy)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Resources["stamina"].Current != before-20.0 {
		t.Errorf("stamina = %f, want %f (heavy costs 20)", p.Resources["stamina"].Current, before-20.0)
	}
}

// ---------------------------------------------------------------------------
// handlePlayerInput
// ---------------------------------------------------------------------------

func TestHandlePlayerInput_AcceptsNearbyPosition(t *testing.T) {
	lvl := level.NewArenaLevel()
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 48}
	p.SpawnTick = 0 // no spawn grace

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	// Move 2 units in Z (well within 10-unit teleport threshold)
	payload := codec.EncodePlayerInput(nil, 0, 0.1, 46, 1.5, 100, 3, 0.1)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Position.Z > 46.1 || p.Position.Z < 45.9 {
		t.Errorf("position Z = %f, want ~46.0 (accepted)", p.Position.Z)
	}
	if p.RotationY != 1.5 {
		t.Errorf("rotation = %f, want 1.5", p.RotationY)
	}
	if p.VisualState != 3 {
		t.Errorf("visual_state = %d, want 3", p.VisualState)
	}
}

func TestHandlePlayerInput_RejectsTeleport(t *testing.T) {
	lvl := level.NewArenaLevel()
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 48}
	p.SpawnTick = 0

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	// Teleport 15 units away (> 10 unit threshold, dist^2 = 225 > 100)
	payload := codec.EncodePlayerInput(nil, 15, 0.1, 48, 0, 100, 0, 0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Position should be unchanged
	if p.Position.X != 0 {
		t.Errorf("position X = %f, want 0 (teleport rejected)", p.Position.X)
	}
}

func TestHandlePlayerInput_SpawnGraceRejectsPosition(t *testing.T) {
	lvl := level.NewArenaLevel()
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 48}
	p.SpawnTick = 95 // spawned 5 ticks ago (< 10 grace ticks)

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	payload := codec.EncodePlayerInput(nil, 1, 0.1, 47, 0, 100, 0, 0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// During spawn grace, position should be unchanged
	if p.Position.X != 0 || p.Position.Z != 48 {
		t.Errorf("position = (%f, %f), want (0, 48) (spawn grace rejects)", p.Position.X, p.Position.Z)
	}
}

func TestHandlePlayerInput_AfterSpawnGraceAccepts(t *testing.T) {
	lvl := level.NewArenaLevel()
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 48}
	p.SpawnTick = 80 // spawned 20 ticks ago (>= 10 grace ticks)

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	payload := codec.EncodePlayerInput(nil, 1, 0.1, 47, 0, 100, 0, 0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Position.X < 0.9 || p.Position.X > 1.1 {
		t.Errorf("position X = %f, want ~1 (grace expired, should accept)", p.Position.X)
	}
}

func TestHandlePlayerInput_YBoundsRejection(t *testing.T) {
	lvl := level.NewArenaLevel()
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 48}
	p.SpawnTick = 0

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	// Try to go above Y bounds (arena max Y is 6.0)
	payload := codec.EncodePlayerInput(nil, 0, 100.0, 48, 0, 100, 0, 0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Position should remain unchanged (Y out of bounds rejects entire update)
	if p.Position.Y != 0.1 {
		t.Errorf("position Y = %f, want 0.1 (Y bounds rejection)", p.Position.Y)
	}
}

func TestHandlePlayerInput_YBelowBoundsRejection(t *testing.T) {
	lvl := level.NewArenaLevel()
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 48}
	p.SpawnTick = 0

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	// Try to go below Y bounds (arena min Y is -1.0)
	payload := codec.EncodePlayerInput(nil, 0, -50.0, 48, 0, 100, 0, 0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Position.Y != 0.1 {
		t.Errorf("position Y = %f, want 0.1 (below Y bounds rejection)", p.Position.Y)
	}
}

func TestHandlePlayerInput_UnknownPeerIgnored(_ *testing.T) {
	lvl := level.NewArenaLevel()
	w := &World{
		ZoneType: 1,
		TickNum:  100,
		Players:  map[uint16]*entity.Player{},
		Level:    lvl,
	}

	payload := codec.EncodePlayerInput(nil, 0, 0.1, 48, 0, 100, 0, 0)
	w.InputQueue = []InputMsg{{PeerID: 99, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	// Should not panic
	is.Tick(w, 0.05)
}

func TestHandlePlayerInput_NilPayloadIgnored(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 5, Y: 0.1, Z: 5}
	lvl := level.NewArenaLevel()

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	// Payload too short for DecodePlayerInput (needs 16 bytes minimum)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: []byte{1, 2}}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Position unchanged
	if p.Position.X != 5 {
		t.Errorf("position X = %f, want 5 (nil payload ignored)", p.Position.X)
	}
}

func TestHandlePlayerInput_ClampsToLevelBounds(t *testing.T) {
	lvl := level.NewArenaLevel()
	p := entity.NewPlayer(1, entity.ClassGunner)
	// Start near the boundary so the move is close enough to not be rejected
	p.Position = entity.Vec3{X: 18, Y: 0.1, Z: 48}
	p.SpawnTick = 0

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	// Try to move slightly beyond X boundary (arena max X is 19.5)
	payload := codec.EncodePlayerInput(nil, 25, 0.1, 48, 0, 100, 0, 0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Position should be clamped, not rejected (dist = 7 units, 49 < 100)
	if p.Position.X > lvl.PlayerBoundsMaxX+0.01 {
		t.Errorf("position X = %f, should be clamped to %f", p.Position.X, lvl.PlayerBoundsMaxX)
	}
}

// ---------------------------------------------------------------------------
// handleInteractInput
// ---------------------------------------------------------------------------

func TestHandleInteractInput_ClassSelect(t *testing.T) {
	tests := []struct {
		name      string
		className string
		wantClass string
	}{
		{"select gunner", entity.ClassGunner, entity.ClassGunner},
		{"select vanguard", entity.ClassVanguard, entity.ClassVanguard},
		{"select blade_dancer", entity.ClassBladeDancer, entity.ClassBladeDancer},
		{"invalid class ignored", "invalid_class", entity.ClassGunner}, // stays original
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := entity.NewPlayer(1, entity.ClassGunner)
			w := makeHubWorld(map[uint16]*entity.Player{1: p})

			payload := codec.EncodeInteractInput(message.InteractClassSelect, tc.className)
			w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpInteractInput, Payload: payload}}

			is := &InputSystem{}
			is.Tick(w, 0.05)

			if p.ClassID != tc.wantClass {
				t.Errorf("class = %q, want %q", p.ClassID, tc.wantClass)
			}
		})
	}
}

func TestHandleInteractInput_ClassSelect_ResetsStats(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	// Gunner default maxHP = 150
	if p.MaxHealth != 150 {
		t.Fatalf("gunner MaxHealth = %f, want 150", p.MaxHealth)
	}

	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	payload := codec.EncodeInteractInput(message.InteractClassSelect, entity.ClassVanguard)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpInteractInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.ClassID != entity.ClassVanguard {
		t.Errorf("class = %q, want 'vanguard'", p.ClassID)
	}
	if p.MaxHealth != 200 {
		t.Errorf("MaxHealth = %f, want 200 (vanguard)", p.MaxHealth)
	}
	if p.Resources["stamina"].Max != 100 {
		t.Errorf("MaxStamina = %f, want 100 (vanguard)", p.Resources["stamina"].Max)
	}
}

func TestHandleInteractInput_ReadyToggle(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Ready = false

	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	payload := codec.EncodeInteractInput(message.InteractReadyToggle, "")
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpInteractInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if !p.Ready {
		t.Error("ready should be true after toggle")
	}

	// Toggle again
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpInteractInput, Payload: payload}}
	is.Tick(w, 0.05)

	if p.Ready {
		t.Error("ready should be false after second toggle")
	}
}

func TestHandleInteractInput_ExitPortal(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	hubRespawnCalled := false

	lvl := level.NewArenaLevel()
	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFightOver,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
		OnPlayerRespawnHub: func(peerID uint16) {
			hubRespawnCalled = true
			if peerID != 1 {
				t.Errorf("respawn peerID = %d, want 1", peerID)
			}
		},
	}
	w.BossDefeated = true

	payload := codec.EncodeInteractInput(message.InteractExitPortal, "")
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpInteractInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if !hubRespawnCalled {
		t.Error("OnPlayerRespawnHub should be called for exit portal after boss defeated")
	}
}

func TestHandleInteractInput_ExitPortal_NotFightOver(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	hubRespawnCalled := false

	lvl := level.NewArenaLevel()
	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight, // not FightOver
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
		OnPlayerRespawnHub: func(_ uint16) {
			hubRespawnCalled = true
		},
	}
	w.BossDefeated = true

	payload := codec.EncodeInteractInput(message.InteractExitPortal, "")
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpInteractInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if hubRespawnCalled {
		t.Error("exit portal should only work in StateFightOver")
	}
}

func TestHandleInteractInput_ExitPortal_BossNotDefeated(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	hubRespawnCalled := false

	lvl := level.NewArenaLevel()
	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFightOver,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
		OnPlayerRespawnHub: func(_ uint16) {
			hubRespawnCalled = true
		},
	}
	w.BossDefeated = false

	payload := codec.EncodeInteractInput(message.InteractExitPortal, "")
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpInteractInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if hubRespawnCalled {
		t.Error("exit portal should only work after boss defeated")
	}
}

func TestHandleInteractInput_UnknownPeerIgnored(_ *testing.T) {
	w := makeHubWorld(map[uint16]*entity.Player{})

	payload := codec.EncodeInteractInput(message.InteractReadyToggle, "")
	w.InputQueue = []InputMsg{{PeerID: 99, Opcode: message.OpInteractInput, Payload: payload}}

	is := &InputSystem{}
	// Should not panic
	is.Tick(w, 0.05)
}

func TestHandleInteractInput_NilPayload(_ *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpInteractInput, Payload: nil}}

	is := &InputSystem{}
	// Should not panic
	is.Tick(w, 0.05)
}

// ---------------------------------------------------------------------------
// handleRespawnRequest
// ---------------------------------------------------------------------------

func TestHandleRespawnRequest_HubRespawn(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = false
	p.Health = 0
	hubRespawnCalled := false

	lvl := level.NewArenaLevel()
	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
		OnPlayerRespawnHub: func(peerID uint16) {
			hubRespawnCalled = true
			if peerID != 1 {
				t.Errorf("hub respawn peerID = %d, want 1", peerID)
			}
		},
	}

	payload := codec.EncodeRespawnRequest(1) // 1 = hub
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpRespawnRequest, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if !hubRespawnCalled {
		t.Error("hub respawn callback should be called")
	}
}

func TestHandleRespawnRequest_ArenaRespawn(t *testing.T) {
	tests := []struct {
		name           string
		state          GameFlowState
		bossGateActive bool
		wantAlive      bool
	}{
		{"in StateFightOver", StateFightOver, false, true},
		{"in StateLobby", StateLobby, false, true},
		{"in StateSpawned", StateSpawned, false, true},
		{"in StateFight boss gate active", StateFight, true, false},
		{"in StateFight trash (no boss gate)", StateFight, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := entity.NewPlayer(1, entity.ClassGunner)
			p.Alive = false
			p.Health = 0

			lvl := level.NewArenaLevel()
			w := &World{
				ZoneType:       1,
				TickNum:        100,
				State:          tc.state,
				BossGateActive: tc.bossGateActive,
				Players:        map[uint16]*entity.Player{1: p},
				Level:          lvl,
			}

			payload := codec.EncodeRespawnRequest(0) // 0 = arena
			w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpRespawnRequest, Payload: payload}}

			is := &InputSystem{}
			is.Tick(w, 0.05)

			if p.Alive != tc.wantAlive {
				t.Errorf("alive = %v, want %v", p.Alive, tc.wantAlive)
			}
			if tc.wantAlive {
				if p.Health != p.MaxHealth {
					t.Errorf("health = %f, want %f", p.Health, p.MaxHealth)
				}
				if p.Position.Z != 48 {
					t.Errorf("position Z = %f, want 48 (warmup)", p.Position.Z)
				}
			}
		})
	}
}

func TestHandleRespawnRequest_AlivePlayerIgnored(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	// Player is alive
	origHealth := p.Health

	lvl := level.NewArenaLevel()
	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFightOver,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	payload := codec.EncodeRespawnRequest(0) // arena
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpRespawnRequest, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Alive player should not be affected
	if p.Health != origHealth {
		t.Errorf("health changed for alive player: %f -> %f", origHealth, p.Health)
	}
}

func TestHandleRespawnRequest_NilPayload(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = false

	lvl := level.NewArenaLevel()
	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFightOver,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    lvl,
	}

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpRespawnRequest, Payload: nil}}

	is := &InputSystem{}
	// Should not panic
	is.Tick(w, 0.05)

	if p.Alive {
		t.Error("player should remain dead with nil payload")
	}
}

func TestHandleRespawnRequest_UnknownPeerIgnored(_ *testing.T) {
	lvl := level.NewArenaLevel()
	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFightOver,
		Players:  map[uint16]*entity.Player{},
		Level:    lvl,
	}

	payload := codec.EncodeRespawnRequest(0)
	w.InputQueue = []InputMsg{{PeerID: 99, Opcode: message.OpRespawnRequest, Payload: payload}}

	is := &InputSystem{}
	// Should not panic
	is.Tick(w, 0.05)
}

// ---------------------------------------------------------------------------
// InputSystem.Tick — dispatching
// ---------------------------------------------------------------------------

func TestInputSystem_ClearsQueueAfterProcessing(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	w.InputQueue = []InputMsg{
		{PeerID: 1, Opcode: message.OpAbilityInput, Payload: abilityPayload(entity.ActionShoot)},
		{PeerID: 1, Opcode: message.OpAbilityInput, Payload: abilityPayload(entity.ActionShoot)},
	}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if len(w.InputQueue) != 0 {
		t.Errorf("input queue = %d, want 0 (should be cleared after tick)", len(w.InputQueue))
	}
}

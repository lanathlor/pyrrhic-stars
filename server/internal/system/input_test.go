package system

import (
	"testing"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
)

func makeHubWorld(players map[uint16]*entity.Player) *World {
	return &World{
		ZoneType: 0, // hub
		TickNum:  100,
		State:    StateLobby, // hub never enters StateFight
		Players:  players,
		Level:    level.NewHubLevel(),
	}
}

func abilityPayload(action uint8) []byte {
	return codec.EncodeAbilityInput(action, 0.0)
}

// ---------------------------------------------------------------------------
// Dodge — works in any zone state
// ---------------------------------------------------------------------------

func TestDodge_ConsumesStamina_InHub(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	before := p.Stamina
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Stamina != before-20.0 {
		t.Errorf("stamina = %f, want %f (dodge costs 20)", p.Stamina, before-20.0)
	}
	if p.StaminaDelay != 0.6 {
		t.Errorf("stamina delay = %f, want 0.6", p.StaminaDelay)
	}
}

func TestDodge_BlockedWhenInsufficientStamina(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.Stamina = 10.0
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Stamina != 10.0 {
		t.Errorf("stamina = %f, want 10.0 (insufficient stamina)", p.Stamina)
	}
}

func TestDodge_RepeatedDrainsStamina(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	w := makeHubWorld(map[uint16]*entity.Player{1: p})
	is := &InputSystem{}

	for i := 0; i < 5; i++ {
		w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
		is.Tick(w, 0.05)
	}

	if p.Stamina != 0.0 {
		t.Errorf("stamina = %f after 5 dodges, want 0.0", p.Stamina)
	}

	// 6th should be blocked
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	is.Tick(w, 0.05)

	if p.Stamina != 0.0 {
		t.Errorf("stamina = %f after 6th dodge, want 0.0", p.Stamina)
	}
}

// ---------------------------------------------------------------------------
// Combat abilities — work in hub (no enemies = no damage, but stamina/cooldowns apply)
// ---------------------------------------------------------------------------

func TestMelee_ConsumesStamina_InHub(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	before := p.Stamina
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionMelee)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Stamina != before-10.0 {
		t.Errorf("stamina = %f, want %f (melee costs 10 in hub)", p.Stamina, before-10.0)
	}
	if p.FireCooldown <= 0 {
		t.Error("fire cooldown should be set after melee in hub")
	}
}

func TestHeavy_ConsumesStamina_InHub(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	before := p.Stamina
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionHeavy)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Stamina != before-20.0 {
		t.Errorf("stamina = %f, want %f (heavy costs 20 in hub)", p.Stamina, before-20.0)
	}
}

func TestShoot_WorksInHub(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionShoot)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.FireCooldown <= 0 {
		t.Error("shoot should set fire cooldown in hub")
	}
}

// ---------------------------------------------------------------------------
// Full pipeline: InputSystem + CombatSystem together
// ---------------------------------------------------------------------------

func TestDodge_FullPipeline_Hub(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	w := makeHubWorld(map[uint16]*entity.Player{1: p})

	inputSys := &InputSystem{}
	combatSys := &CombatSystem{}

	if p.Stamina != 100.0 {
		t.Fatalf("initial stamina = %f, want 100.0", p.Stamina)
	}

	// Tick 1: dodge
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionDodge)}}
	inputSys.Tick(w, 0.05)
	combatSys.Tick(w, 0.05)

	if p.Stamina != 80.0 {
		t.Errorf("tick 1: stamina = %f, want 80.0", p.Stamina)
	}

	// Ticks 2-5: regen blocked by delay
	for i := 2; i <= 5; i++ {
		inputSys.Tick(w, 0.05)
		combatSys.Tick(w, 0.05)
	}

	if p.Stamina != 80.0 {
		t.Errorf("tick 5: stamina = %f, want 80.0 (delay still active)", p.Stamina)
	}

	// Ticks 6-13: delay expires, regen starts
	for i := 6; i <= 13; i++ {
		inputSys.Tick(w, 0.05)
		combatSys.Tick(w, 0.05)
	}

	if p.Stamina <= 80.0 {
		t.Errorf("tick 13: stamina = %f, want > 80.0 (regen should have started)", p.Stamina)
	}
	if p.Stamina >= 100.0 {
		t.Errorf("tick 13: stamina = %f, want < 100.0", p.Stamina)
	}
}

// ---------------------------------------------------------------------------
// Combat abilities — work in StateFight with enemies
// ---------------------------------------------------------------------------

func TestMelee_ConsumesStamina_InFight(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0, Z: 2.0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})

	before := p.Stamina
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionMelee)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Stamina != before-10.0 {
		t.Errorf("stamina = %f, want %f (melee costs 10)", p.Stamina, before-10.0)
	}
}

func TestHeavy_ConsumesStamina_InFight(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0, Z: 2.0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})

	before := p.Stamina
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: abilityPayload(entity.ActionHeavy)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Stamina != before-20.0 {
		t.Errorf("stamina = %f, want %f (heavy costs 20)", p.Stamina, before-20.0)
	}
}

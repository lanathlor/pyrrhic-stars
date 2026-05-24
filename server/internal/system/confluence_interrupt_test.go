package system

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/entity"
)

// sustain def for testing — CancelOnDamage | CancelOnMove, with sustain.
var testSustainDef = ability.AbilityDef{
	ID:                "test_sustain_channel",
	Name:              "Test Sustain",
	CommitTime:        2.0,
	ExecuteTime:       0.1,
	Cooldown:          5.0,
	CancelConditions:  uint8(ability.CancelOnMove) | uint8(ability.CancelOnDamage),
	Sustain:           true,
	SustainCostPerSec: 0, // no cost — prevent flux-based cancel
	SustainEffect:     5,
	SustainInterval:   0.5,
	SustainCooldown:   5.0,
	Hit:               ability.HitDef{Type: ability.HitAllyTarget},
}

func harmonistWithConfluence(id uint16, stacks int) *entity.Player {
	p := entity.NewPlayerWithSpec(id, entity.ClassArcanotechnicien, "harmonist")
	p.Confluence.Stacks = stacks
	return p
}

func runnerInSustain(def *ability.AbilityDef, startTick uint32, pos entity.Vec3) *ability.PlayerAbilityRunner {
	r := &ability.PlayerAbilityRunner{}
	r.StartSustain(def, pos, startTick)
	return r
}

func runnerInCommit(def *ability.AbilityDef) *ability.PlayerAbilityRunner {
	r := &ability.PlayerAbilityRunner{}
	r.Start(def)
	return r
}

func TestConfluence_DropsOnCancelOnDamage(t *testing.T) {
	p := harmonistWithConfluence(1, 3)
	p.LastDamageTick = 200 // damage after sustain start

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 200
	runner := runnerInSustain(&testSustainDef, 100, p.Position) // sustain started at tick 100
	w.AbilityRunners[1] = runner

	sys := CombatSystem{}
	sys.Tick(w, 0.05)

	if p.Confluence.Stacks != 0 {
		t.Errorf("CancelOnDamage: Confluence.Stacks = %d, want 0", p.Confluence.Stacks)
	}
}

func TestConfluence_DropsOnCancelOnMove(t *testing.T) {
	p := harmonistWithConfluence(1, 3)
	startPos := entity.Vec3{X: 0, Y: 0, Z: 0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	runner := runnerInSustain(&testSustainDef, 100, startPos)
	w.AbilityRunners[1] = runner

	// Move player far from sustain start
	p.Position = entity.Vec3{X: 5, Y: 0, Z: 0}

	sys := CombatSystem{}
	sys.Tick(w, 0.05)

	if p.Confluence.Stacks != 0 {
		t.Errorf("CancelOnMove: Confluence.Stacks = %d, want 0", p.Confluence.Stacks)
	}
}

func TestConfluence_DropsOnInsufficientFlux(t *testing.T) {
	p := harmonistWithConfluence(1, 3)

	// Use a def that costs flux during sustain
	fluxDef := testSustainDef
	fluxDef.SustainCostPerSec = 100 // very expensive
	fluxDef.SustainInterval = 0.5

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	runner := runnerInSustain(&fluxDef, 100, p.Position)
	w.AbilityRunners[1] = runner

	// Drain all flux so sustain tick fails
	p.SetAllFluxPoolsCurrent(0)

	// Tick enough times for a sustain tick to fire (0.5s at 0.05 per tick = 10 ticks)
	sys := CombatSystem{}
	for i := 0; i < 12; i++ {
		sys.Tick(w, 0.05)
	}

	if p.Confluence.Stacks != 0 {
		t.Errorf("InsufficientFlux: Confluence.Stacks = %d, want 0", p.Confluence.Stacks)
	}
}

func TestConfluence_DropsOnDodgeCancel(t *testing.T) {
	// Harmonist has no dodge — use a Vanguard with Confluence for this test
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Confluence = &entity.ConfluenceState{Stacks: 3, MaxStacks: 5, DecayRate: 1.0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	runner := runnerInCommit(&testSustainDef)
	w.AbilityRunners[1] = runner

	// Simulate dodge input (action 3)
	payload := codec.EncodeAbilityInput(entity.ActionDodge, 0, 0)
	handleAbilityInput(w, 1, payload)

	if p.Confluence.Stacks != 0 {
		t.Errorf("Dodge: Confluence.Stacks = %d, want 0", p.Confluence.Stacks)
	}
}

func TestConfluence_DropsOnESCCancel(t *testing.T) {
	p := harmonistWithConfluence(1, 3)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	runner := runnerInSustain(&testSustainDef, 100, p.Position)
	w.AbilityRunners[1] = runner

	// Simulate ESC cancel (action 255)
	payload := codec.EncodeAbilityInput(255, 0, 0)
	handleAbilityInput(w, 1, payload)

	if p.Confluence.Stacks != 0 {
		t.Errorf("ESC: Confluence.Stacks = %d, want 0", p.Confluence.Stacks)
	}
}

func TestConfluence_DropsOnNewChanneledAbility(t *testing.T) {
	p := harmonistWithConfluence(1, 3)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	runner := runnerInSustain(&testSustainDef, 100, p.Position)
	w.AbilityRunners[1] = runner

	// Map a channeled ability to an action so handleAbilityInput can find it
	p.ActionMap[51] = "mending_beam"

	// Simulate committing a new channeled ability (action 51)
	payload := codec.EncodeAbilityInput(51, 0, 0)
	handleAbilityInput(w, 1, payload)

	if p.Confluence.Stacks != 0 {
		t.Errorf("NewChanneled: Confluence.Stacks = %d, want 0", p.Confluence.Stacks)
	}
}

func TestConfluence_DropsOnInstantAbilityCancel(t *testing.T) {
	p := harmonistWithConfluence(1, 3)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	runner := runnerInSustain(&testSustainDef, 100, p.Position)
	w.AbilityRunners[1] = runner

	// Map an instant ability (no CommitTime)
	p.ActionMap[50] = "mending_surge"

	ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
	ally.Health = 50
	w.Players[2] = ally

	payload := codec.EncodeAbilityInput(50, 0, 0)
	handleAbilityInput(w, 1, payload)

	// Interrupt drops stacks to 0, then mending_surge calls OnAbilityComplete (+1).
	// Net result: stacks = 1 (not the original 3).
	if p.Confluence.Stacks != 1 {
		t.Errorf("InstantAbility: Confluence.Stacks = %d, want 1 (0 from interrupt + 1 from new ability)", p.Confluence.Stacks)
	}
}

func TestConfluence_NotDroppedWhenRunnerIdle(t *testing.T) {
	p := harmonistWithConfluence(1, 3)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	// Runner is idle (no runner set, or idle phase)
	runner := &ability.PlayerAbilityRunner{} // idle by default
	w.AbilityRunners[1] = runner

	// ESC cancel with idle runner should NOT drop stacks
	payload := codec.EncodeAbilityInput(255, 0, 0)
	handleAbilityInput(w, 1, payload)

	if p.Confluence.Stacks != 3 {
		t.Errorf("Idle: Confluence.Stacks = %d, want 3 (should NOT drop)", p.Confluence.Stacks)
	}
}

package bosstest

import (
	"fmt"
	"math/rand/v2"
	"time"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/system"
)

const (
	defaultMaxTicks = 12000 // 10 minutes at 20Hz
	defaultDt       = 0.05  // 50ms per tick (20Hz)
)

// PuppetConfig defines a single player in a simulation party.
type PuppetConfig struct {
	Class   string
	Profile BotProfile
}

// SimConfig configures a single simulation run.
type SimConfig struct {
	Boss        string
	Party       []PuppetConfig
	MaxTicks    int
	Seed        uint64
	Sink        combatlog.EventSink
	GroupID     string
	RunID       string
	PuppetTrees *PuppetTreeRegistry // optional: YAML-defined puppet BTs
}

// AbilityResult tracks per-ability combat statistics from a simulation run.
type AbilityResult struct {
	Name        string
	Hits        int
	Kills       int
	Dodges      int
	TotalDamage float32
}

// SimResult holds the outcome of a single simulation run.
type SimResult struct {
	Outcome       combatlog.Outcome
	Duration      time.Duration
	TotalTicks    int
	PhasesReached []int
	TreeReport    *TreeReport
	ClassDamage   map[string]float32        // class → total damage dealt to boss
	AbilityStats  map[string]*AbilityResult // ability name → stats
	CompName      string                    // composition name (set by runner)
}

// RunSimulation executes a single boss fight simulation using the real game server pipeline.
func RunSimulation(cfg SimConfig) SimResult {
	def := enemyai.DefRegistry[cfg.Boss]
	if def == nil {
		panic(fmt.Sprintf("RunSimulation: boss %q not in DefRegistry", cfg.Boss))
	}

	maxTicks := cfg.MaxTicks
	if maxTicks == 0 {
		maxTicks = defaultMaxTicks
	}

	rng := rand.New(rand.NewPCG(cfg.Seed, cfg.Seed+42))

	// Load real arena level (obstacles, bounds, etc.)
	lvl := level.NewArenaLevel()

	// Create enemy
	enemy := entity.NewEnemy(1, def.MaxHealth, def.Name)
	enemy.Alive = true
	enemy.IsBoss = true
	enemy.LeashRadius = 100
	enemy.AggroRadius = 50

	// Create ability engine
	engine := ability.NewEngine(nil)

	// Create brain and instrument tree for health tracking
	brain := enemyai.NewBrainSeeded(def, enemy, engine, cfg.Seed)
	brain.BoundsMinX = lvl.EnemyBoundsMinX
	brain.BoundsMaxX = lvl.EnemyBoundsMaxX
	brain.BoundsMinZ = lvl.EnemyBoundsMinZ
	brain.BoundsMaxZ = lvl.EnemyBoundsMaxZ

	instrumented := InstrumentTree(brain.Tree())
	brain.SetTree(instrumented.Root)

	// Create puppets and register as players
	puppets := make([]*PlayerPuppet, len(cfg.Party))
	playerMap := make(map[uint16]*entity.Player, len(cfg.Party))
	for i, pc := range cfg.Party {
		pp := NewPuppet(uint16(i+1), pc.Class, pc.Profile, cfg.Seed+uint64(i)*100, cfg.Boss, cfg.PuppetTrees)
		pp.Player.SpawnTick = 0 // no spawn grace period
		puppets[i] = pp
		playerMap[pp.Player.ID] = pp.Player
	}

	// Compute group-size scaling (HP: 1x→4x, Damage: 1x→2x over 1→5 players)
	groupSize := len(cfg.Party)
	hpMult := float32(1.0 + 0.75*float64(groupSize-1))
	dmgMult := float32(1.0 + 0.25*float64(groupSize-1))
	enemy.MaxHealth *= hpMult
	enemy.Health = enemy.MaxHealth

	// Build system.World — the same state container the real server uses
	w := system.World{
		ZoneID:          fmt.Sprintf("%s_%d", cfg.GroupID, cfg.Seed),
		ZoneType:        1, // instanced
		RunID:           cfg.RunID,
		State:           system.StateFight,
		EnemyDamageMult: dmgMult,
		Players:         playerMap,
		Enemies:         []*entity.Enemy{enemy},
		Brains:          []enemyai.BrainTicker{brain},
		Level:           lvl,
		AbilityEngine:   engine,
		PatternEngine:   combat.NewPatternEngine(),
		PatternRng:      rng,
		Clients:         make(map[uint16]*system.Client),
		CombatLogSink:   cfg.Sink,
		SendBuf:         make([]byte, 0, 4096),
		DamageBuf:       make([]byte, 0, 256),
		GameFlowBuf:     make([]byte, 0, 256),
		LobbyBuf:        make([]byte, 0, 512),
	}
	if w.CombatLogSink == nil {
		w.CombatLogSink = combatlog.NullSink{}
	}

	// Force boss into chase immediately (skip patrol→aggro)
	enemy.State = entity.EnemyChase
	enemy.TargetPlayerID = 1

	// Set up combat log session
	instanceID := fmt.Sprintf("%s_%d", cfg.GroupID, cfg.Seed)
	session := combatlog.NewSession(
		w.CombatLogSink, instanceID, cfg.GroupID, cfg.Boss,
		"sim", cfg.RunID, 0, combatlog.SourceSimulation, 0,
	)
	for _, pp := range puppets {
		session.AddParticipant(combatlog.ParticipantLog{
			EntityID:   combatlog.FormatPlayerID(pp.Player.ID),
			Name:       fmt.Sprintf("%s_%s", pp.Profile, pp.Player.ClassID),
			Class:      pp.Player.ClassID,
			IsBot:      true,
			BotProfile: string(pp.Profile),
		})
	}
	session.AddParticipant(combatlog.ParticipantLog{
		EntityID: combatlog.FormatEnemyID(enemy.ID),
		Name:     def.Name,
		Class:    "enemy",
	})
	// Pre-populate combat logs so AISystem doesn't create a duplicate
	w.CombatLogs = map[int]*combatlog.EncounterSession{-1: session}

	// System pipeline: same as real arena minus GameFlowSystem and NetworkSystem
	pipeline := []system.System{
		&system.InputSystem{},
		&system.AISystem{},
		&system.CombatSystem{},
		&system.PhysicsSystem{},
	}

	// Track phases reached
	phasesReached := map[int]bool{1: true}
	classDmg := make(map[string]float32)
	abilStats := make(map[string]*AbilityResult)
	var replayBuf []byte

	// Simulation loop
	var outcome combatlog.Outcome
	tick := 0
	for ; tick < maxTicks; tick++ {
		w.TickNum = uint32(tick + 1)
		w.DamageEvents = w.DamageEvents[:0]
		w.GameFlowEvents = w.GameFlowEvents[:0]

		bossHP := enemy.Health / enemy.MaxHealth
		currentPhase := enemy.Phase

		// Phase tracking
		if !phasesReached[currentPhase] {
			phasesReached[currentPhase] = true
			session.CheckPhaseChange(w.TickNum, 0, currentPhase, bossHP)
		}

		// Resolve active ability for puppet context
		var activeAbil *ability.AbilityDef
		if abil := def.AbilityByIndex(enemy.ActiveAbility); abil != nil {
			resolved := def.ResolveAbility(abil, enemy.Phase)
			activeAbil = &resolved
		}

		// --- Puppet BT ticks: generate input messages ---
		for _, pp := range puppets {
			if !pp.Player.Alive {
				continue
			}
			ctx := &PuppetContext{
				Puppet:     pp,
				World:      &w,
				Boss:       enemy,
				BossDef:    def,
				ActiveAbil: activeAbil,
				AllPuppets: puppets,
				Dt:         defaultDt,
			}
			pp.Tick(ctx)
		}

		// --- Run real system pipeline ---
		bossHPBefore := enemy.Health
		for _, sys := range pipeline {
			sys.Tick(&w, defaultDt)
		}

		// --- Track player damage to boss ---
		bossHPDelta := bossHPBefore - enemy.Health
		if bossHPDelta > 0 {
			// Attribute damage via DamageEvents (player→enemy)
			for _, ev := range w.DamageEvents {
				if ev.SourcePeerID != 0 && ev.SourceType == combat.SourcePlayerAttack {
					if p, ok := w.Players[ev.SourcePeerID]; ok {
						classDmg[p.ClassID] += ev.Amount
					}
				}
			}
		}

		// --- Track boss ability stats (enemy→player damage) ---
		abilName := ""
		if abil := def.AbilityByIndex(enemy.ActiveAbility); abil != nil {
			abilName = abil.Name
		}
		for _, ev := range w.DamageEvents {
			if ev.SourcePeerID == 0 && ev.TargetPeerID != 0 {
				if abilName != "" {
					ar := trackAbility(abilStats, abilName)
					ar.Hits++
					ar.TotalDamage += ev.Amount
				}
				if p, ok := w.Players[ev.TargetPeerID]; ok && !p.Alive {
					if abilName != "" {
						trackAbility(abilStats, abilName).Kills++
					}
				}
			}
		}

		// Record replay frame
		replayBuf = replayBuf[:0]
		replayBuf = codec.AppendEncodeWorldState(replayBuf, w.TickNum, w.Players, w.Enemies, w.Projectiles, nil)
		session.Recorder.AppendFrame(replayBuf)

		// Check termination
		if enemy.Health <= 0 || !enemy.Alive {
			enemy.Health = 0
			enemy.Alive = false
			outcome = combatlog.OutcomePlayerWin
			break
		}
		allDead := true
		for _, p := range w.Players {
			if p.Alive {
				allDead = false
				break
			}
		}
		if allDead {
			outcome = combatlog.OutcomeBossWin
			break
		}
	}

	if outcome == "" {
		outcome = combatlog.OutcomeTimeout
	}

	// Only finalize if CombatSystem didn't already (player_win is finalized
	// by checkEnemyGroupDead inside CombatSystem.Tick).
	if w.CombatLogs[-1] != nil {
		session.Finalize(outcome, uint32(tick))
	}

	// Build phases list
	phases := make([]int, 0, len(phasesReached))
	for p := range phasesReached {
		phases = append(phases, p)
	}

	return SimResult{
		Outcome:       outcome,
		Duration:      time.Duration(tick) * 50 * time.Millisecond,
		TotalTicks:    tick,
		PhasesReached: phases,
		TreeReport:    instrumented.Report(),
		ClassDamage:   classDmg,
		AbilityStats:  abilStats,
	}
}

func trackAbility(m map[string]*AbilityResult, name string) *AbilityResult {
	if r, ok := m[name]; ok {
		return r
	}
	r := &AbilityResult{Name: name}
	m[name] = r
	return r
}

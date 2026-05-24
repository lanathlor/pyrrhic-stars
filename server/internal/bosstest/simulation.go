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
	Spec    string // spec ID (empty = class default)
	Profile BotProfile
	Loadout []string // optional: ability IDs for loadout slots (overrides spec default)
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
	SpecDamage    map[string]float32        // spec → total damage dealt to boss
	SpecHealing   map[string]float32        // spec → total healing done
	SpecPlayers   map[string]int            // spec → number of players with that spec
	AbilityStats  map[string]*AbilityResult // ability name → stats
	CompName      string                    // composition name (set by runner)
}

// simState holds the fully initialised state handed off from setupSimulation to
// the RunSimulation tick loop.
type simState struct {
	def           *enemyai.EnemyDef
	enemy         *entity.Enemy
	puppets       []*PlayerPuppet
	specPlayers   map[string]int
	world         system.World
	session       *combatlog.EncounterSession
	pipeline      []system.System
	instrumented  *InstrumentedTree
	phasesReached map[int]bool
	specDmg       map[string]float32
	specHealing   map[string]float32
	abilStats     map[string]*AbilityResult
}

// enemySetup holds the initialised enemy, engine, brain, level, and
// instrumented tree before players are added.
type enemySetup struct {
	def          *enemyai.EnemyDef
	enemy        *entity.Enemy
	engine       *ability.Engine
	brain        enemyai.BrainTicker
	lvl          *level.Level
	instrumented *InstrumentedTree
}

// initEnemy creates the enemy entity, ability engine, and instrumented brain.
func initEnemy(def *enemyai.EnemyDef, seed uint64) enemySetup {
	lvl := level.NewArenaLevel()

	enemy := entity.NewEnemy(1, def.MaxHealth, def.Name)
	enemy.Alive = true
	enemy.IsBoss = true
	enemy.LeashRadius = 100
	enemy.AggroRadius = 50

	engine := ability.NewEngine(nil)

	brain := enemyai.NewBrainSeeded(def, enemy, engine, seed)
	brain.BoundsMinX = lvl.EnemyBoundsMinX
	brain.BoundsMaxX = lvl.EnemyBoundsMaxX
	brain.BoundsMinZ = lvl.EnemyBoundsMinZ
	brain.BoundsMaxZ = lvl.EnemyBoundsMaxZ

	instrumented := InstrumentTree(brain.Tree())
	brain.SetTree(instrumented.Root)

	return enemySetup{
		def:          def,
		enemy:        enemy,
		engine:       engine,
		brain:        brain,
		lvl:          lvl,
		instrumented: instrumented,
	}
}

// initPuppets creates the puppet players and returns them along with the
// player map and per-spec player counts.
func initPuppets(cfg SimConfig) ([]*PlayerPuppet, map[uint16]*entity.Player, map[string]int) {
	puppets := make([]*PlayerPuppet, len(cfg.Party))
	playerMap := make(map[uint16]*entity.Player, len(cfg.Party))
	for i, pc := range cfg.Party {
		pp := NewPuppet(uint16(i+1), pc.Class, pc.Spec, pc.Profile, cfg.Seed+uint64(i)*100, cfg.Boss, cfg.PuppetTrees)
		pp.Player.SpawnTick = 0 // no spawn grace period

		// Apply custom loadout if specified (overrides spec default).
		if len(pc.Loadout) > 0 {
			loadout := &entity.Loadout{}
			for j, abilID := range pc.Loadout {
				if j < 6 {
					loadout.Slots[j] = abilID
				}
			}
			pp.Player.Loadout = loadout
			pp.Player.ApplyLoadout()
		}

		puppets[i] = pp
		playerMap[pp.Player.ID] = pp.Player
	}
	specPlayers := make(map[string]int)
	for _, pp := range puppets {
		specPlayers[pp.Player.SpecID]++
	}
	return puppets, playerMap, specPlayers
}

// setupSimulation initialises every object needed by the tick loop.
func setupSimulation(cfg SimConfig) simState {
	def := enemyai.DefRegistry[cfg.Boss]
	if def == nil {
		panic(fmt.Sprintf("RunSimulation: boss %q not in DefRegistry", cfg.Boss))
	}

	rng := rand.New(rand.NewPCG(cfg.Seed, cfg.Seed+42))

	es := initEnemy(def, cfg.Seed)
	puppets, playerMap, specPlayers := initPuppets(cfg)

	// Compute group-size scaling (HP: 1x→4x, Damage: 1x→2x over 1→5 players)
	groupSize := len(cfg.Party)
	hpMult := float32(1.0 + 0.75*float64(groupSize-1))
	dmgMult := float32(1.0 + 0.25*float64(groupSize-1))
	es.enemy.MaxHealth *= hpMult
	es.enemy.Health = es.enemy.MaxHealth

	w := buildWorld(cfg, es, playerMap, dmgMult, rng)

	// Force boss into chase immediately (skip patrol→aggro)
	es.enemy.State = entity.EnemyChase
	es.enemy.TargetPlayerID = 1

	sess := buildCombatSession(cfg, &w, puppets, es.enemy, def)

	// System pipeline: same as real arena minus GameFlowSystem and NetworkSystem
	pipeline := []system.System{
		&system.InputSystem{},
		&system.AISystem{},
		&system.CombatSystem{},
		&system.PhysicsSystem{},
	}

	return simState{
		def:           def,
		enemy:         es.enemy,
		puppets:       puppets,
		specPlayers:   specPlayers,
		world:         w,
		session:       sess,
		pipeline:      pipeline,
		instrumented:  es.instrumented,
		phasesReached: map[int]bool{1: true},
		specDmg:       make(map[string]float32),
		specHealing:   make(map[string]float32),
		abilStats:     make(map[string]*AbilityResult),
	}
}

// buildWorld constructs the system.World used by the simulation.
func buildWorld(cfg SimConfig, es enemySetup, playerMap map[uint16]*entity.Player, dmgMult float32, rng *rand.Rand) system.World {
	w := system.World{
		ZoneID:          fmt.Sprintf("%s_%d", cfg.GroupID, cfg.Seed),
		ZoneType:        1, // instanced
		RunID:           cfg.RunID,
		State:           system.StateFight,
		EnemyDamageMult: dmgMult,
		Players:         playerMap,
		Enemies:         []*entity.Enemy{es.enemy},
		Brains:          []enemyai.BrainTicker{es.brain},
		Level:           es.lvl,
		AbilityEngine:   es.engine,
		PatternEngine:   combat.NewPatternEngine(),
		PatternRng:      rng,
		AbilityRunners:  make(map[uint16]*ability.PlayerAbilityRunner),
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
	return w
}

// buildCombatSession creates the encounter session, registers all participants,
// and pre-populates w.CombatLogs so the AISystem doesn't create a duplicate.
func buildCombatSession(cfg SimConfig, w *system.World, puppets []*PlayerPuppet, enemy *entity.Enemy, def *enemyai.EnemyDef) *combatlog.EncounterSession {
	instanceID := fmt.Sprintf("%s_%d", cfg.GroupID, cfg.Seed)
	sess := combatlog.NewSession(
		w.CombatLogSink, instanceID, cfg.GroupID, cfg.Boss,
		"sim", cfg.RunID, 0, combatlog.SourceSimulation, 0,
	)
	for _, pp := range puppets {
		sess.AddParticipant(combatlog.ParticipantLog{
			EntityID:   combatlog.FormatPlayerID(pp.Player.ID),
			Name:       fmt.Sprintf("%s_%s_%s", pp.Profile, pp.Player.ClassID, pp.Player.SpecID),
			Class:      pp.Player.ClassID,
			IsBot:      true,
			BotProfile: string(pp.Profile),
		})
	}
	sess.AddParticipant(combatlog.ParticipantLog{
		EntityID: combatlog.FormatEnemyID(enemy.ID),
		Name:     def.Name,
		Class:    "enemy",
	})
	// Pre-populate combat logs so AISystem doesn't create a duplicate
	w.CombatLogs = map[int]*combatlog.EncounterSession{-1: sess}
	return sess
}

// RunSimulation executes a single boss fight simulation using the real game server pipeline.
func RunSimulation(cfg SimConfig) SimResult {
	maxTicks := cfg.MaxTicks
	if maxTicks == 0 {
		maxTicks = defaultMaxTicks
	}

	st := setupSimulation(cfg)

	outcome, tick := runTickLoop(&st, maxTicks)

	if outcome == "" {
		outcome = combatlog.OutcomeTimeout
	}

	// Only finalize if CombatSystem didn't already (player_win is finalized
	// by checkEnemyGroupDead inside CombatSystem.Tick).
	if st.world.CombatLogs[-1] != nil {
		st.session.Finalize(outcome, uint32(tick))
	}

	return SimResult{
		Outcome:       outcome,
		Duration:      time.Duration(tick) * 50 * time.Millisecond,
		TotalTicks:    tick,
		PhasesReached: collectPhases(st.phasesReached),
		TreeReport:    st.instrumented.Report(),
		SpecDamage:    st.specDmg,
		SpecHealing:   st.specHealing,
		SpecPlayers:   st.specPlayers,
		AbilityStats:  st.abilStats,
	}
}

// runTickLoop advances the simulation one tick at a time until maxTicks,
// boss death, or party wipe. Returns the outcome and final tick count.
func runTickLoop(st *simState, maxTicks int) (combatlog.Outcome, int) {
	var replayBuf []byte
	var outcome combatlog.Outcome
	tick := 0
	for ; tick < maxTicks; tick++ {
		st.world.TickNum = uint32(tick + 1)
		st.world.DamageEvents = st.world.DamageEvents[:0]
		st.world.GameFlowEvents = st.world.GameFlowEvents[:0]

		bossHP := st.enemy.Health / st.enemy.MaxHealth
		currentPhase := st.enemy.Phase
		if !st.phasesReached[currentPhase] {
			st.phasesReached[currentPhase] = true
			st.session.CheckPhaseChange(st.world.TickNum, 0, currentPhase, bossHP)
		}

		var activeAbil *ability.AbilityDef
		if abil := st.def.AbilityByIndex(st.enemy.ActiveAbility); abil != nil {
			resolved := st.def.ResolveAbility(abil, st.enemy.Phase)
			activeAbil = &resolved
		}

		tickPuppets(st.puppets, &st.world, st.enemy, st.def, activeAbil)

		bossHPBefore := st.enemy.Health
		for _, sys := range st.pipeline {
			sys.Tick(&st.world, defaultDt)
		}

		collectTickStats(&st.world, st.enemy, st.def, bossHPBefore, st.specDmg, st.specHealing, st.abilStats)

		replayBuf = replayBuf[:0]
		replayBuf = codec.AppendEncodeWorldState(replayBuf, st.world.TickNum, st.world.Players, st.world.Enemies, st.world.Projectiles, nil)
		st.session.Recorder.AppendFrame(replayBuf)

		if terminated, o := checkTermination(st.enemy, st.world.Players); terminated {
			outcome = o
			break
		}
	}
	return outcome, tick
}

// checkTermination tests whether the fight has ended. It returns true along
// with the outcome when either the boss or all players are dead.
func checkTermination(enemy *entity.Enemy, players map[uint16]*entity.Player) (bool, combatlog.Outcome) {
	if enemy.Health <= 0 || !enemy.Alive {
		enemy.Health = 0
		enemy.Alive = false
		return true, combatlog.OutcomePlayerWin
	}
	for _, p := range players {
		if p.Alive {
			return false, ""
		}
	}
	return true, combatlog.OutcomeBossWin
}

// collectPhases converts the phases-reached bool map to a sorted int slice.
func collectPhases(phasesReached map[int]bool) []int {
	phases := make([]int, 0, len(phasesReached))
	for p := range phasesReached {
		phases = append(phases, p)
	}
	return phases
}

// tickPuppets runs one BT tick for each alive puppet, generating their input
// messages for the current tick.
func tickPuppets(puppets []*PlayerPuppet, w *system.World, enemy *entity.Enemy, def *enemyai.EnemyDef, activeAbil *ability.AbilityDef) {
	for _, pp := range puppets {
		if !pp.Player.Alive {
			continue
		}
		ctx := &PuppetContext{
			Puppet:     pp,
			World:      w,
			Boss:       enemy,
			BossDef:    def,
			ActiveAbil: activeAbil,
			AllPuppets: puppets,
			Dt:         defaultDt,
		}
		pp.Tick(ctx)
	}
}

// collectTickStats attributes damage events from the current tick to the
// per-spec damage map, per-spec healing map, and per-ability stats map.
func collectTickStats(
	w *system.World,
	enemy *entity.Enemy,
	def *enemyai.EnemyDef,
	bossHPBefore float32,
	specDmg map[string]float32,
	specHealing map[string]float32,
	abilStats map[string]*AbilityResult,
) {
	// Track player damage to boss.
	if bossHPBefore > enemy.Health {
		for _, ev := range w.DamageEvents {
			if ev.SourcePeerID != 0 && ev.SourceType == combat.SourcePlayerAttack {
				if p, ok := w.Players[ev.SourcePeerID]; ok {
					specDmg[p.SpecID] += ev.Amount
				}
			}
		}
	}

	// Track healing done.
	for _, ev := range w.DamageEvents {
		if ev.SourceType == combat.SourcePlayerHeal && ev.Amount > 0 {
			if p, ok := w.Players[ev.SourcePeerID]; ok {
				specHealing[p.SpecID] += ev.Amount
			}
		}
	}

	// Track boss ability stats (enemy→player damage).
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
			if p, ok := w.Players[ev.TargetPeerID]; ok && !p.Alive && abilName != "" {
				trackAbility(abilStats, abilName).Kills++
			}
		}
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

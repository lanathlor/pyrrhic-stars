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
	"codex-online/server/internal/overflux"
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
	Boss        string   // single-enemy encounter / puppet-tree lookup label
	Enemies     []string // enemy def names to spawn (empty = [Boss]); >1 = trash pack
	Party       []PuppetConfig
	MaxTicks    int
	Seed        uint64
	Sink        combatlog.EventSink
	GroupID     string
	RunID       string
	PuppetTrees *PuppetTreeRegistry // optional: YAML-defined puppet BTs
	Overflux    *overflux.State     // optional: overflux conditions for difficulty scaling
}

// enemyDefs returns the normalized list of enemy def names to spawn.
func (c SimConfig) enemyDefs() []string {
	if len(c.Enemies) > 0 {
		return c.Enemies
	}
	return []string{c.Boss}
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
	TreeReports   map[string]*TreeReport    // enemy def name → behavior-tree report
	SpecDamage    map[string]float32        // spec → total damage dealt to enemies
	SpecHealing   map[string]float32        // spec → total healing done
	SpecPlayers   map[string]int            // spec → number of players with that spec
	AbilityStats  map[string]*AbilityResult // ability name → stats
	CompName      string                    // composition name (set by runner)
	OverfluxName  string                    // overflux config name (set by runner, empty = baseline)
	OverfluxScore int                       // total overflux score (0 = baseline)
}

// enemyInst is one spawned enemy with its def, brain, and instrumented tree.
type enemyInst struct {
	def          *enemyai.EnemyDef
	enemy        *entity.Enemy
	brain        enemyai.BrainTicker
	instrumented *InstrumentedTree
}

// simState holds the fully initialised state handed off from setupSimulation to
// the RunSimulation tick loop.
type simState struct {
	enemies        []enemyInst
	puppets        []*PlayerPuppet
	specPlayers    map[string]int
	world          system.World
	session        *combatlog.EncounterSession
	sessionKey     int // CombatLogs key the session is stored under (-1 boss, packGroupID pack)
	pipeline       []system.System
	phasesReached  map[int]bool
	specDmg        map[string]float32
	specHealing    map[string]float32
	abilStats      map[string]*AbilityResult
	sourceTypeAbil map[uint8]string // enemy SourceType → ability name (pack attribution)
	isPack         bool
}

// initEnemies spawns every enemy in the encounter, sharing one ability engine
// and arena level. A boss (single enemy) keeps ID 1 / IsBoss; pack enemies use
// the real-zone 1000+i ID range, share packGroupID so they aggro and finalize
// together, and are spread over a small formation.
func initEnemies(defNames []string, seed uint64, isPack bool) (*ability.Engine, *level.Level, []enemyInst) { //nolint:revive // flag-parameter: boss vs pack spawn differs in ID/group/position
	lvl, err := level.Load("arena")
	if err != nil {
		panic(fmt.Sprintf("bosstest: %v", err))
	}
	engine := ability.NewEngine(nil)
	positions := packFormation(len(defNames))

	insts := make([]enemyInst, 0, len(defNames))
	for i, name := range defNames {
		def := enemyai.DefRegistry[name]
		if def == nil {
			panic(fmt.Sprintf("RunSimulation: enemy %q not in DefRegistry", name))
		}

		var id uint16 = 1
		if isPack {
			id = uint16(1000 + i)
		}
		enemy := entity.NewEnemy(id, def.MaxHealth, def.Name)
		enemy.Alive = true
		enemy.LeashRadius = 100
		enemy.AggroRadius = 50
		if isPack {
			enemy.GroupID = packGroupID
			enemy.Position = positions[i]
			enemy.LeashOrigin = positions[i]
		} else {
			enemy.IsBoss = true
		}

		brain := enemyai.NewBrainSeeded(def, enemy, engine, seed+uint64(i))
		brain.BoundsMinX = lvl.EnemyBoundsMinX
		brain.BoundsMaxX = lvl.EnemyBoundsMaxX
		brain.BoundsMinZ = lvl.EnemyBoundsMinZ
		brain.BoundsMaxZ = lvl.EnemyBoundsMaxZ

		instrumented := InstrumentTree(brain.Tree())
		brain.SetTree(instrumented.Root)

		insts = append(insts, enemyInst{def: def, enemy: enemy, brain: brain, instrumented: instrumented})
	}
	return engine, lvl, insts
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
	defNames := cfg.enemyDefs()
	isPack := len(defNames) > 1

	rng := rand.New(rand.NewPCG(cfg.Seed, cfg.Seed+42))

	engine, lvl, insts := initEnemies(defNames, cfg.Seed, isPack)

	// Apply overflux variants (BT replacement, ability overrides) before
	// instrumentation so the variant tree gets profiled — per enemy.
	if cfg.Overflux != nil {
		reinstrumentForOverflux(insts, cfg.Overflux)
	}

	puppets, playerMap, specPlayers := initPuppets(cfg)

	// Compute group-size scaling (HP: 1x→4x, Damage: 1x→2x over 1→5 players).
	groupSize := len(cfg.Party)
	hpMult := float32(1.0 + 0.75*float64(groupSize-1))
	dmgMult := float32(1.0 + 0.25*float64(groupSize-1))
	if cfg.Overflux != nil {
		hpMult *= cfg.Overflux.HPMultiplier()
		dmgMult *= cfg.Overflux.DamageMultiplier()
	}

	scaleAndEngage(insts, hpMult)

	w := buildWorld(cfg, engine, lvl, insts, playerMap, dmgMult, rng)

	sessionKey := -1
	if isPack {
		sessionKey = packGroupID
	}
	sess := buildCombatSession(cfg, &w, puppets, insts, sessionKey)

	// System pipeline: same as real arena minus GameFlowSystem and NetworkSystem.
	pipeline := []system.System{
		&system.InputSystem{},
		&system.AISystem{},
		&system.CombatSystem{},
		&system.PhysicsSystem{},
	}

	return simState{
		enemies:        insts,
		puppets:        puppets,
		specPlayers:    specPlayers,
		world:          w,
		session:        sess,
		sessionKey:     sessionKey,
		pipeline:       pipeline,
		phasesReached:  map[int]bool{1: true},
		specDmg:        make(map[string]float32),
		specHealing:    make(map[string]float32),
		abilStats:      make(map[string]*AbilityResult),
		sourceTypeAbil: sourceTypeAbilities(defsOf(insts)),
		isPack:         isPack,
	}
}

// reinstrumentForOverflux applies overflux variants to every brain and
// re-instruments its (possibly replaced) tree so profiling covers the variant.
func reinstrumentForOverflux(insts []enemyInst, oflx *overflux.State) {
	for i := range insts {
		insts[i].brain.ApplyOverfluxVariants(oflx)
		insts[i].instrumented = InstrumentTree(insts[i].brain.Tree())
		insts[i].brain.SetTree(insts[i].instrumented.Root)
	}
}

// scaleAndEngage applies the group-size HP multiplier to every enemy and forces
// them into chase immediately (skip patrol→aggro), targeting player 1.
func scaleAndEngage(insts []enemyInst, hpMult float32) {
	for i := range insts {
		e := insts[i].enemy
		e.MaxHealth *= hpMult
		e.Health = e.MaxHealth
		e.State = entity.EnemyChase
		e.TargetPlayerID = 1
	}
}

// defsOf extracts the EnemyDef pointers from the enemy instances.
func defsOf(insts []enemyInst) []*enemyai.EnemyDef {
	defs := make([]*enemyai.EnemyDef, len(insts))
	for i := range insts {
		defs[i] = insts[i].def
	}
	return defs
}

// buildWorld constructs the system.World used by the simulation.
func buildWorld(cfg SimConfig, engine *ability.Engine, lvl *level.Level, insts []enemyInst, playerMap map[uint16]*entity.Player, dmgMult float32, rng *rand.Rand) system.World {
	enemies := make([]*entity.Enemy, len(insts))
	brains := make([]enemyai.BrainTicker, len(insts))
	for i := range insts {
		enemies[i] = insts[i].enemy
		brains[i] = insts[i].brain
	}

	w := system.World{
		ZoneID:          fmt.Sprintf("%s_%d", cfg.GroupID, cfg.Seed),
		ZoneType:        1, // instanced
		RunID:           cfg.RunID,
		EnemyDamageMult: dmgMult,
		OverfluxState:   cfg.Overflux,
		Players:         playerMap,
		Enemies:         enemies,
		Brains:          brains,
		Level:           lvl,
		AbilityEngine:   engine,
		PatternEngine:   combat.NewPatternEngine(),
		PatternRng:      rng,
		AbilityRunners:  make(map[uint16]*ability.PlayerAbilityRunner),
		Clients:         make(map[uint16]*system.Client),
		CombatLogSink:   cfg.Sink,
		SendBuf:         make([]byte, 0, 4096),
		DamageBuf:       make([]byte, 0, 256),
		GameFlowBuf:     make([]byte, 0, 256),
	}
	if w.CombatLogSink == nil {
		w.CombatLogSink = combatlog.NullSink{}
	}
	w.InitGateStates()
	return w
}

// buildCombatSession creates the encounter session, registers all participants,
// and pre-populates w.CombatLogs under sessionKey so the AISystem doesn't create
// a duplicate and CombatSystem.checkEnemyGroupDead can finalize it.
func buildCombatSession(cfg SimConfig, w *system.World, puppets []*PlayerPuppet, insts []enemyInst, sessionKey int) *combatlog.EncounterSession {
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
	for i := range insts {
		e := insts[i].enemy
		sess.AddParticipant(combatlog.ParticipantLog{
			EntityID: combatlog.FormatEnemyID(e.ID),
			Name:     insts[i].def.Name,
			Class:    "enemy",
		})
	}
	// Pre-populate combat logs so AISystem doesn't create a duplicate.
	w.CombatLogs = map[int]*combatlog.EncounterSession{sessionKey: sess}
	return sess
}

// RunSimulation executes a single encounter simulation using the real game server pipeline.
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

	// Only finalize if CombatSystem didn't already (player_win is finalized by
	// checkEnemyGroupDead inside CombatSystem.Tick, which deletes the session).
	if st.world.CombatLogs[st.sessionKey] != nil {
		st.session.Finalize(outcome, uint32(tick))
	}

	return SimResult{
		Outcome:       outcome,
		Duration:      time.Duration(tick) * 50 * time.Millisecond,
		TotalTicks:    tick,
		PhasesReached: collectPhases(st.phasesReached),
		TreeReports:   mergeTreeReportsByDef(st.enemies),
		SpecDamage:    st.specDmg,
		SpecHealing:   st.specHealing,
		SpecPlayers:   st.specPlayers,
		AbilityStats:  st.abilStats,
	}
}

// mergeTreeReportsByDef merges the instrumented trees of every enemy, bucketed by
// def name. Enemies of the same def share an identical tree topology, so their
// reports merge safely by node index.
func mergeTreeReportsByDef(insts []enemyInst) map[string]*TreeReport {
	reports := make(map[string]*TreeReport)
	for i := range insts {
		name := insts[i].def.Name
		rep := insts[i].instrumented.Report()
		if existing, ok := reports[name]; ok {
			MergeTreeReport(existing, rep)
		} else {
			reports[name] = CloneTreeReport(rep)
		}
	}
	for _, rep := range reports {
		ClassifyTreeReport(rep)
	}
	return reports
}

// runTickLoop advances the simulation one tick at a time until maxTicks, all
// enemies dead, or party wipe. Returns the outcome and final tick count.
func runTickLoop(st *simState, maxTicks int) (combatlog.Outcome, int) {
	var replayBuf []byte
	var outcome combatlog.Outcome
	tick := 0
	for ; tick < maxTicks; tick++ {
		st.world.TickNum = uint32(tick + 1)
		st.world.DamageEvents = st.world.DamageEvents[:0]
		st.world.GameFlowEvents = st.world.GameFlowEvents[:0]

		// Phase tracking only applies to single-enemy (boss) encounters.
		if !st.isPack {
			primary := st.enemies[0]
			bossHP := primary.enemy.Health / primary.enemy.MaxHealth
			currentPhase := primary.enemy.Phase
			if !st.phasesReached[currentPhase] {
				st.phasesReached[currentPhase] = true
				st.session.CheckPhaseChange(st.world.TickNum, 0, currentPhase, bossHP)
			}
		}

		tickPuppets(st.puppets, &st.world, st.enemies)

		for _, sys := range st.pipeline {
			sys.Tick(&st.world, defaultDt)
		}

		collectTickStats(&st.world, st.enemies, st.sourceTypeAbil, st.isPack, st.specDmg, st.specHealing, st.abilStats)

		replayBuf = replayBuf[:0]
		replayBuf = codec.AppendEncodeWorldState(replayBuf, st.world.TickNum, st.world.Players, st.world.Enemies, st.world.Projectiles, nil)
		st.session.Recorder.AppendFrame(replayBuf)

		if terminated, o := checkTermination(st.enemies, st.world.Players); terminated {
			outcome = o
			break
		}
	}
	return outcome, tick
}

// checkTermination tests whether the fight has ended. It returns true along with
// the outcome when either every enemy or all players are dead.
func checkTermination(insts []enemyInst, players map[uint16]*entity.Player) (bool, combatlog.Outcome) {
	anyEnemyAlive := false
	for i := range insts {
		e := insts[i].enemy
		if e.Health <= 0 {
			e.Health = 0
			e.Alive = false
		}
		if e.Alive {
			anyEnemyAlive = true
		}
	}
	if !anyEnemyAlive {
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

// nearestEnemyInst returns the closest alive enemy to pos, or nil if all dead.
func nearestEnemyInst(insts []enemyInst, pos entity.Vec3) *enemyInst {
	var best *enemyInst
	var bestDist float32
	for i := range insts {
		e := insts[i].enemy
		if !e.Alive {
			continue
		}
		d := e.Position.Flat().DistanceTo(pos.Flat())
		if best == nil || d < bestDist {
			best = &insts[i]
			bestDist = d
		}
	}
	return best
}

// tickPuppets runs one BT tick for each alive puppet against its nearest alive
// enemy, generating their input messages for the current tick.
func tickPuppets(puppets []*PlayerPuppet, w *system.World, insts []enemyInst) {
	for _, pp := range puppets {
		if !pp.Player.Alive {
			continue
		}
		inst := nearestEnemyInst(insts, pp.Player.Position)
		if inst == nil {
			continue // all enemies dead; the loop will terminate this tick
		}

		var activeAbil *ability.AbilityDef
		if abil := inst.def.AbilityByIndex(inst.enemy.ActiveAbility); abil != nil {
			resolved := inst.def.ResolveAbility(abil, inst.enemy.Phase)
			activeAbil = &resolved
		}

		ctx := &PuppetContext{
			Puppet:     pp,
			World:      w,
			Boss:       inst.enemy,
			BossDef:    inst.def,
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
	insts []enemyInst,
	sourceTypeAbil map[uint8]string,
	isPack bool,
	specDmg map[string]float32,
	specHealing map[string]float32,
	abilStats map[string]*AbilityResult,
) {
	for _, ev := range w.DamageEvents {
		switch {
		case ev.SourceType == combat.SourcePlayerAttack && ev.SourcePeerID != 0:
			// Player damage to enemies (party DPS vs the encounter).
			if p, ok := w.Players[ev.SourcePeerID]; ok {
				specDmg[p.SpecID] += ev.Amount
			}
		case ev.SourceType == combat.SourcePlayerHeal && ev.Amount > 0:
			if p, ok := w.Players[ev.SourcePeerID]; ok {
				specHealing[p.SpecID] += ev.Amount
			}
		case ev.SourcePeerID == 0 && ev.TargetPeerID != 0:
			// Enemy → player damage. Events carry no per-enemy id, so attribute
			// by SourceType for packs, or the boss's active ability for a boss.
			abilName := enemyAbilityName(insts, sourceTypeAbil, isPack, ev.SourceType)
			if abilName == "" {
				continue
			}
			ar := trackAbility(abilStats, abilName)
			ar.Hits++
			ar.TotalDamage += ev.Amount
			if p, ok := w.Players[ev.TargetPeerID]; ok && !p.Alive {
				ar.Kills++
			}
		}
	}
}

// enemyAbilityName resolves the ability name to credit an enemy→player damage
// event to. Packs attribute by SourceType; a boss uses its current active ability.
func enemyAbilityName(insts []enemyInst, sourceTypeAbil map[uint8]string, isPack bool, sourceType uint8) string { //nolint:revive // flag-parameter: pack attributes by source type, boss by active ability
	if isPack {
		return sourceTypeAbil[sourceType]
	}
	primary := insts[0]
	if abil := primary.def.AbilityByIndex(primary.enemy.ActiveAbility); abil != nil {
		return abil.Name
	}
	return ""
}

func trackAbility(m map[string]*AbilityResult, name string) *AbilityResult {
	if r, ok := m[name]; ok {
		return r
	}
	r := &AbilityResult{Name: name}
	m[name] = r
	return r
}

package bosstest

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
	"codex-online/server/internal/overflux"
	"codex-online/server/internal/system"
)

// InstanceConfig configures a full-dungeon simulation run: the real level with
// every spawn, gates, checkpoints, and respawns — puppets advance from the
// lobby to the final boss like a live group.
type InstanceConfig struct {
	Level    string // level name (e.g. "arena")
	Party    []PuppetConfig
	MaxTicks int
	Seed     uint64

	// ChainPull: while every listed patrol group still has alive members,
	// puppets advance onto the LAST group so earlier groups aggro en route
	// and the packs are fought together (AoE-friendly chain pulling).
	ChainPull   bool
	ChainGroups []int // defaults to [1, 2] (the hallway packs)

	// RespawnDelayTicks is how long a dead puppet waits before requesting a
	// respawn (and between retries while a boss room is sealed). Default 60.
	RespawnDelayTicks int

	// Overflux difficulty conditions, applied to every enemy in the level
	// (variant trees/abilities for defs that declare them, HP/damage
	// multipliers, Wounded Prey regen on the engaged boss).
	Overflux *overflux.State

	PuppetTrees *PuppetTreeRegistry

	// debugHook, when set (in-package tests), is invoked every tick after the
	// pipeline so stuck runs can be inspected.
	debugHook func(tick int, w *system.World, insts []enemyInst, puppets []*PlayerPuppet)
}

// ClassDamage splits one player's outgoing damage by target kind.
type ClassDamage struct {
	Boss  float32
	Trash float32
}

// InstanceResult holds the outcome and pacing metrics of one full-dungeon run.
type InstanceResult struct {
	Outcome      combatlog.Outcome
	TotalTicks   int
	ClearSeconds float32 // fight start → final boss dead (0 unless cleared)
	OverTime     bool    // cleared, but past the level's clear timer
	Deaths       int     // individual player deaths (wipes included)
	Wipes        int     // full-party wipes

	// DeathsBySection counts player deaths per dungeon section (derived from
	// the victim's Z against the level's gate lines).
	DeathsBySection map[string]int

	// SegmentTicks is the completion tick (from fight start) of each content
	// milestone: "packs" (hallway groups dead), "boss1", "decline" (decline
	// groups dead), "boss2". Missing key = never completed (timeout).
	SegmentTicks map[string]int

	// DowntimeTicks counts fight-started ticks with no enemy engaged: travel,
	// run-backs, regrouping — the time a group "loses" outside combat.
	DowntimeTicks int

	// BossEngagedTicks / TrashEngagedTicks count ticks with a boss (resp. a
	// non-boss enemy) engaged — the correct DPS denominators per target kind.
	// The windows can overlap (e.g. stalkers during the Aceras fight).
	BossEngagedTicks  int
	TrashEngagedTicks int

	// ClassDamage sums player→enemy damage per party slot ("class/spec"),
	// split by boss vs trash targets.
	ClassDamage map[string]*ClassDamage

	// AbilityCommits counts enemy ability commits ("def:ability"), from the
	// zone coordination bus. Shows what the encounter actually cast.
	AbilityCommits map[string]int
}

const defaultInstanceMaxTicks = 24000 // 20 minutes at 20Hz

// instanceState is everything setupInstance builds for the tick loop.
type instanceState struct {
	lvl     *level.Level
	w       system.World
	insts   []enemyInst
	puppets []*PlayerPuppet
}

// setupInstance loads the level, spawns enemies and puppets, applies
// group-size/overflux scaling and readies the lobby, mirroring what the zone
// does for a live group.
func setupInstance(cfg InstanceConfig) *instanceState {
	lvl, err := level.Load(cfg.Level)
	if err != nil {
		panic(fmt.Sprintf("RunInstance: %v", err))
	}
	engine := ability.NewEngine(nil)
	insts := initInstanceEnemies(lvl, engine, cfg.Seed)

	// Overflux variants (variant trees/abilities) before the tick loop so the
	// instrumented trees cover the variant, matching the boss sim.
	if cfg.Overflux != nil {
		reinstrumentForOverflux(insts, cfg.Overflux)
	}

	puppets, playerMap, _ := initPuppets(SimConfig{
		Party:       cfg.Party,
		Seed:        cfg.Seed,
		PuppetTrees: cfg.PuppetTrees,
	})
	for _, pp := range puppets {
		pp.NoBoundsClamp = true // full-level navigation, not boss-room clamping
	}

	// Group-size scaling, matching zone.rescaleEnemies (overflux multipliers included).
	groupSize := len(cfg.Party)
	hpMult := float32(1.0 + 0.75*float64(groupSize-1))
	dmgMult := float32(1.0 + 0.25*float64(groupSize-1))
	if cfg.Overflux != nil {
		hpMult *= cfg.Overflux.HPMultiplier()
		dmgMult *= cfg.Overflux.DamageMultiplier()
	}
	for i := range insts {
		e := insts[i].enemy
		e.MaxHealth *= hpMult
		e.Health = e.MaxHealth
	}

	rng := rand.New(rand.NewPCG(cfg.Seed, cfg.Seed+42))
	w := buildWorld(SimConfig{GroupID: "instance", Seed: cfg.Seed, Overflux: cfg.Overflux}, engine, lvl, insts, playerMap, dmgMult, rng)
	w.EnemyHPMult = hpMult

	system.InitInstance(&w)
	system.SpawnPlayers(&w)
	for _, pp := range puppets {
		pp.Player.Ready = true // lobby ready-up → countdown → fight start
	}
	return &instanceState{lvl: lvl, w: w, insts: insts, puppets: puppets}
}

// intOr returns v, or def when v is zero.
func intOr(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

// chainPullGroups returns the groups to chain-pull (defaulting to the two
// hallway packs), or nil when chain-pulling is disabled.
func chainPullGroups(cfg InstanceConfig) []int {
	if !cfg.ChainPull {
		return nil
	}
	if len(cfg.ChainGroups) == 0 {
		return []int{1, 2}
	}
	return cfg.ChainGroups
}

// newPuppetTrackers allocates the per-puppet tick-loop state: last-tick
// aliveness (everyone starts alive), respawn countdowns, and sticky targets.
func newPuppetTrackers(n int) ([]bool, []int, []*enemyInst) {
	prevAlive := make([]bool, n)
	for i := range prevAlive {
		prevAlive[i] = true
	}
	return prevAlive, make([]int, n), make([]*enemyInst, n)
}

// newInstanceResult builds an empty result that reads as a timeout until the
// run proves otherwise.
func newInstanceResult() InstanceResult {
	return InstanceResult{
		Outcome:         combatlog.OutcomeTimeout,
		DeathsBySection: make(map[string]int),
		SegmentTicks:    make(map[string]int),
		ClassDamage:     make(map[string]*ClassDamage),
		AbilityCommits:  make(map[string]int),
	}
}

// RunInstance executes one full-dungeon simulation using the real system
// pipeline including GameFlowSystem (gates, boss activation/reset, wipes,
// checkpoint respawns).
func RunInstance(cfg InstanceConfig) InstanceResult {
	maxTicks := intOr(cfg.MaxTicks, defaultInstanceMaxTicks)
	respawnDelay := intOr(cfg.RespawnDelayTicks, 60)
	chainGroups := chainPullGroups(cfg)

	st := setupInstance(cfg)
	lvl, insts, puppets := st.lvl, st.insts, st.puppets
	w := st.w

	// Real zone pipeline order, minus NPC/Network (nothing to render or route).
	pipeline := []system.System{
		&system.CombatSystem{},
		&system.InputSystem{},
		&system.GameFlowSystem{},
		&system.AISystem{},
		&system.PhysicsSystem{},
	}

	res := newInstanceResult()
	prevAlive, respawnTimers, stickyTargets := newPuppetTrackers(len(puppets))
	sections := gateSections(lvl)
	milestones := instanceMilestones(insts)

	for tick := range maxTicks {
		w.TickNum = uint32(tick + 1)
		w.DamageEvents = w.DamageEvents[:0]
		w.GameFlowEvents = w.GameFlowEvents[:0]

		// Puppets idle through the lobby countdown, then advance.
		if w.FightStartTick != 0 {
			tickInstancePuppets(puppets, &w, insts, stickyTargets, chainGroups)
		}
		queueRespawns(&w, puppets, prevAlive, respawnTimers, respawnDelay, &res, sections)

		for _, sys := range pipeline {
			sys.Tick(&w, defaultDt)
		}

		insts = adoptSpawnedInsts(&w, insts)
		collectInstanceTickStats(&w, insts, puppets, &res, milestones, tick)

		if cfg.debugHook != nil {
			cfg.debugHook(tick, &w, insts, puppets)
		}

		for _, evt := range w.GameFlowEvents {
			if evt.FlowType == message.FlowAllDead {
				res.Wipes++
			}
		}

		if w.BossDefeated {
			markInstanceWin(&res, &w, lvl, tick)
			break
		}
	}
	if res.TotalTicks == 0 {
		res.TotalTicks = maxTicks
	}
	collectAbilityCommits(&w, &res)
	return res
}

// markInstanceWin records the final-boss kill: outcome, run length, and
// whether the clear beat the level's timer.
func markInstanceWin(res *InstanceResult, w *system.World, lvl *level.Level, tick int) {
	res.Outcome = combatlog.OutcomePlayerWin
	res.TotalTicks = tick + 1
	clearTicks := w.TickNum - w.FightStartTick
	res.ClearSeconds = float32(clearTicks) * defaultDt
	if lvl.ClearTimeSeconds > 0 {
		res.OverTime = res.ClearSeconds > lvl.ClearTimeSeconds
	}
}

// gateSection is a Z-band of the dungeon between two gate lines.
type gateSection struct {
	name string
	minZ float32
	maxZ float32
}

// gateSections derives dungeon sections from the level's gate positions.
// Sections are the natural narrative beats of the arena: lobby, hallway,
// boss-1 room, decline, final room. Missing gates degrade gracefully.
func gateSections(lvl *level.Level) []gateSection {
	gateZ := map[string]float32{}
	for i := range lvl.Gates {
		gateZ[lvl.Gates[i].ID] = lvl.Gates[i].Position.Z
	}
	const inf = float32(1e9)
	var sections []gateSection
	lo := -inf
	if z, ok := gateZ["aceras_gate"]; ok {
		sections = append(sections, gateSection{"aceras", lo, z})
		lo = z
	}
	if z, ok := gateZ["decline_gate"]; ok {
		sections = append(sections, gateSection{"decline", lo, z})
		lo = z
	}
	if z, ok := gateZ["boss_gate"]; ok {
		sections = append(sections, gateSection{"boss1", lo, z})
		lo = z
	}
	if z, ok := gateZ["lobby_gate"]; ok {
		sections = append(sections, gateSection{"hallway", lo, z})
		lo = z
	}
	sections = append(sections, gateSection{"lobby", lo, inf})
	return sections
}

func sectionAt(sections []gateSection, z float32) string {
	for _, s := range sections {
		if z >= s.minZ && z < s.maxZ {
			return s.name
		}
	}
	return "unknown"
}

// milestone is a content beat whose completion tick is recorded once.
type milestone struct {
	name string
	done func() bool
}

// instanceMilestones builds the ordered content beats from the spawned
// enemies: hallway packs (groups 1+2), boss 1, decline packs (groups 3+4),
// final boss.
func instanceMilestones(insts []enemyInst) []milestone {
	groupDead := func(gids ...int) func() bool {
		return func() bool {
			for i := range insts {
				e := insts[i].enemy
				if e.Alive {
					if slices.Contains(gids, e.GroupID) {
						return false
					}
				}
			}
			return true
		}
	}
	bossDead := func(num int) func() bool {
		return func() bool {
			for i := range insts {
				e := insts[i].enemy
				if e.IsBoss && e.BossNum == num && e.Alive {
					return false
				}
			}
			return true
		}
	}
	return []milestone{
		{"packs", groupDead(1, 2)},
		{"boss1", bossDead(1)},
		{"decline", groupDead(3, 4)},
		{"boss2", bossDead(2)},
	}
}

// collectInstanceTickStats records per-tick metrics: milestone completion,
// downtime (no enemy engaged), and per-class damage split by target kind.
func collectInstanceTickStats(w *system.World, insts []enemyInst, puppets []*PlayerPuppet, res *InstanceResult, milestones []milestone, tick int) {
	if w.FightStartTick == 0 {
		return
	}

	for i := range milestones {
		if _, done := res.SegmentTicks[milestones[i].name]; !done && milestones[i].done() {
			res.SegmentTicks[milestones[i].name] = tick - int(w.FightStartTick)
		}
	}

	recordEngagement(w, insts, res)
	recordClassDamage(w, insts, puppets, res)
}

// recordEngagement bumps the boss/trash engagement and downtime counters for
// this tick.
func recordEngagement(w *system.World, insts []enemyInst, res *InstanceResult) {
	bossEngaged, trashEngaged := false, false
	for i := range insts {
		e := insts[i].enemy
		if e.Alive && e.State != entity.EnemyPatrol && e.State != entity.EnemyIdle {
			if e.IsBoss {
				bossEngaged = true
			} else {
				trashEngaged = true
			}
			if bossEngaged && trashEngaged {
				break
			}
		}
	}
	if bossEngaged {
		res.BossEngagedTicks++
	}
	if trashEngaged {
		res.TrashEngagedTicks++
	}
	if !bossEngaged && !trashEngaged && !w.BossDefeated {
		res.DowntimeTicks++
	}
}

// recordClassDamage attributes this tick's player→enemy damage events to the
// dealing class, split boss vs trash.
func recordClassDamage(w *system.World, insts []enemyInst, puppets []*PlayerPuppet, res *InstanceResult) {
	var isBoss map[uint16]bool
	for _, ev := range w.DamageEvents {
		if ev.SourceType != 0 || ev.SourcePeerID == 0 || ev.TargetPeerID < 1000 {
			continue // not player→enemy damage
		}
		if isBoss == nil {
			isBoss = make(map[uint16]bool, len(insts))
			for i := range insts {
				isBoss[insts[i].enemy.ID] = insts[i].enemy.IsBoss
			}
		}
		key := classKeyFor(puppets, ev.SourcePeerID)
		if key == "" {
			continue
		}
		cd := res.ClassDamage[key]
		if cd == nil {
			cd = &ClassDamage{}
			res.ClassDamage[key] = cd
		}
		if isBoss[ev.TargetPeerID] {
			cd.Boss += ev.Amount
		} else {
			cd.Trash += ev.Amount
		}
	}
}

func classKeyFor(puppets []*PlayerPuppet, peerID uint16) string {
	for _, pp := range puppets {
		if pp.Player.ID == peerID {
			return pp.Player.ClassID + "/" + pp.Player.SpecID
		}
	}
	return ""
}

// collectAbilityCommits counts every enemy ability commit recorded on the
// zone coordination bus (RetainAll worlds keep the full timeline).
func collectAbilityCommits(w *system.World, res *InstanceResult) {
	if w.Bus == nil {
		return
	}
	for _, evt := range w.Bus.Events() {
		if evt.Channel == enemyai.ChanCommitStarted && evt.Ability != "" {
			res.AbilityCommits[evt.SourceDef+":"+evt.Ability]++
		}
	}
}

// initInstanceEnemies spawns every enemy from the level's spawn list,
// mirroring zone.spawnEnemies (IDs 1000+i, patrol state, boss/gate/aggro fields).
func initInstanceEnemies(lvl *level.Level, engine *ability.Engine, seed uint64) []enemyInst {
	insts := make([]enemyInst, 0, len(lvl.EnemySpawns))
	for i, sp := range lvl.EnemySpawns {
		def := enemyai.DefRegistry[sp.DefName]
		if def == nil {
			panic(fmt.Sprintf("RunInstance: enemy %q not in DefRegistry", sp.DefName))
		}
		e := entity.NewEnemy(uint16(1000+i), def.MaxHealth, sp.DefName)
		e.IsBoss = sp.IsBoss
		e.PatrolA = sp.PatrolA
		e.PatrolB = sp.PatrolB
		e.AggroRadius = sp.AggroRadius
		e.LeashOrigin = sp.Position
		e.LeashRadius = sp.LeashRadius
		e.GroupID = sp.GroupID
		e.AggroMaxZ = sp.AggroMaxZ
		e.BossNum = sp.BossNum
		e.BossGateID = sp.BossGateID
		if e.IsBoss && e.BossNum == 0 {
			e.BossNum = 1
		}
		if e.IsBoss && e.BossGateID == "" {
			e.BossGateID = "boss_gate"
		}
		e.Reset(sp.Position, entity.EnemyPatrol)

		brain := enemyai.NewBrainSeeded(def, e, engine, seed+uint64(i))
		brain.BoundsMinX = lvl.EnemyBoundsMinX
		brain.BoundsMaxX = lvl.EnemyBoundsMaxX
		brain.BoundsMinZ = lvl.EnemyBoundsMinZ
		brain.BoundsMaxZ = lvl.EnemyBoundsMaxZ
		instrumented := InstrumentTree(brain.Tree())
		brain.SetTree(instrumented.Root)

		insts = append(insts, enemyInst{def: def, enemy: e, brain: brain, instrumented: instrumented})
	}
	return insts
}

// tickInstancePuppets advances every alive puppet against its current target:
// the nearest alive enemy with stickiness (see stickTarget), or — while every
// chain-pull group still lives (nil chainGroups disables chain-pulling) — the
// nearest member of the last chain group, so earlier packs aggro on the way
// in and get fought together.
func tickInstancePuppets(puppets []*PlayerPuppet, w *system.World, insts []enemyInst, sticky []*enemyInst, chainGroups []int) {
	chainTarget := -1
	if len(chainGroups) > 0 && allGroupsAlive(insts, chainGroups) {
		chainTarget = chainGroups[len(chainGroups)-1]
	}
	for i, pp := range puppets {
		if !pp.Player.Alive {
			continue
		}
		var inst *enemyInst
		if chainTarget >= 0 {
			inst = nearestGroupEnemyInst(insts, pp.Player.Position, chainTarget)
		}
		if inst == nil {
			inst = stickTarget(pp, sticky[i], nearestEnemyInst(insts, pp.Player.Position))
		}
		sticky[i] = inst
		if inst == nil {
			continue
		}

		// Rally before a boss pull: while the target is a still-patrolling
		// boss, puppets at its doorstep hold until the whole alive party has
		// caught up. Prevents split pulls where the gate seals stragglers
		// out — the sealed-out half otherwise idles at preferred range
		// against a closed gate while the inside half can neither finish
		// nor reset the boss (seen as instance-fuzz timeouts).
		if inst.enemy.IsBoss && inst.enemy.State == entity.EnemyPatrol &&
			partyRearDistance(puppets, inst.enemy.Position) > 25 &&
			pp.Player.Position.Flat().DistanceTo(inst.enemy.Position.Flat()) < inst.enemy.AggroRadius+6 {
			continue
		}

		var activeAbil *ability.AbilityDef
		if abil := inst.def.AbilityByIndex(inst.enemy.ActiveAbility); abil != nil {
			resolved := inst.def.ResolveAbility(abil, inst.enemy.Phase)
			activeAbil = &resolved
		}
		pp.Tick(&PuppetContext{
			Puppet:     pp,
			World:      w,
			Boss:       inst.enemy,
			BossDef:    inst.def,
			ActiveAbil: activeAbil,
			AllPuppets: puppets,
			Dt:         defaultDt,
		})
	}
}

// stickTarget applies target stickiness: keep the current target while it
// lives unless the nearest candidate is at least twice as close. Re-picking
// "nearest" every tick made puppets flap between crossing mobs and chase
// whichever happened to be closest that instant — melee whiffed for whole
// pack fights. Real players commit to a kill target.
func stickTarget(pp *PlayerPuppet, current, nearest *enemyInst) *enemyInst {
	if pp.PreferredRange() >= 5 {
		return nearest // ranged: no travel cost, shoot whatever is in view
	}
	if current == nil || !current.enemy.Alive {
		return nearest
	}
	if nearest == nil || nearest == current {
		return current
	}
	dCur := pp.Player.Position.Flat().DistanceTo(current.enemy.Position.Flat())
	dNew := pp.Player.Position.Flat().DistanceTo(nearest.enemy.Position.Flat())
	if dNew < dCur*0.5 {
		return nearest // something much closer (e.g. a stalker on our back)
	}
	return current
}

// partyRearDistance returns the distance from pos to the farthest ALIVE puppet.
func partyRearDistance(puppets []*PlayerPuppet, pos entity.Vec3) float32 {
	var rear float32
	for _, pp := range puppets {
		if !pp.Player.Alive {
			continue
		}
		if d := pp.Player.Position.Flat().DistanceTo(pos.Flat()); d > rear {
			rear = d
		}
	}
	return rear
}

// allGroupsAlive reports whether every listed GroupID still has an alive member.
func allGroupsAlive(insts []enemyInst, groups []int) bool {
	for _, gid := range groups {
		if nearestGroupEnemyInst(insts, entity.Vec3{}, gid) == nil {
			return false
		}
	}
	return true
}

// nearestGroupEnemyInst returns the closest alive enemy of the given group, or nil.
func nearestGroupEnemyInst(insts []enemyInst, pos entity.Vec3, groupID int) *enemyInst {
	var best *enemyInst
	var bestDist float32
	for i := range insts {
		e := insts[i].enemy
		if !e.Alive || e.GroupID != groupID {
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

// queueRespawns counts alive→dead transitions (tagging the death's dungeon
// section) and injects respawn requests for dead puppets on a retry timer.
// The InputSystem enforces the real rules: blocked while a boss room is
// sealed, checkpoint selection by progression.
func queueRespawns(w *system.World, puppets []*PlayerPuppet, prevAlive []bool, timers []int, delay int, res *InstanceResult, sections []gateSection) {
	for i, pp := range puppets {
		alive := pp.Player.Alive
		if prevAlive[i] && !alive {
			res.Deaths++
			res.DeathsBySection[sectionAt(sections, pp.Player.Position.Z)]++
			timers[i] = delay
		}
		prevAlive[i] = alive
		if alive {
			continue
		}
		timers[i]--
		if timers[i] <= 0 {
			timers[i] = delay // retry cadence while sealed rooms block respawn
			w.InputQueue = append(w.InputQueue, system.InputMsg{
				PeerID:  pp.Player.ID,
				Opcode:  message.OpRespawnRequest,
				Payload: codec.EncodeRespawnRequest(0),
			})
		}
	}
}

// adoptSpawnedInsts wraps mid-fight spawned enemies (summoned adds) into
// enemyInst entries. Shared by the instance runner; the boss-sim variant
// (adoptSpawnedEnemies) mutates its simState in place.
func adoptSpawnedInsts(w *system.World, insts []enemyInst) []enemyInst {
	for len(insts) < len(w.Enemies) {
		i := len(insts)
		e := w.Enemies[i]
		brain := w.Brains[i]
		def := enemyai.DefRegistry[e.DefName]
		if def == nil || brain == nil {
			return insts
		}
		instrumented := InstrumentTree(brain.Tree())
		brain.SetTree(instrumented.Root)
		insts = append(insts, enemyInst{def: def, enemy: e, brain: brain, instrumented: instrumented})
	}
	return insts
}

package bot

import (
	"math"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"codex-online/server/internal/system"
)

const (
	followStartDist    float32 = 3.0
	followTeleportDist float32 = 40.0
	combatDetectRange  float32 = 30.0
)

// TickAll updates all bots for the current tick.
func (m *Manager) TickAll(w *system.World, dt float32) {
	if len(m.bots) == 0 {
		return
	}

	// Collect all puppets once for AllPuppets context.
	allPuppets := make([]*bosstest.PlayerPuppet, 0, len(m.bots))
	for _, lb := range m.bots {
		allPuppets = append(allPuppets, lb.Puppet)
	}

	for _, lb := range m.bots {
		// Auto-respawn dead bots after a delay (bots can't send OpRespawnRequest).
		// No respawn while the boss room is sealed — dead is dead during boss encounters.
		if !lb.Puppet.Player.Alive {
			if !w.BossGateActive {
				lb.deathTimer += dt
				if lb.deathTimer >= BotRespawnDelay {
					respawnBot(lb, w)
				}
			}
			continue
		}

		owner := w.Players[lb.OwnerID]
		// Find enemies near the owner so bots fight the same pack, not wander to distant aggroed enemies.
		searchAnchor := lb.Puppet.Player
		if owner != nil {
			searchAnchor = owner
		}
		enemy := findNearestAggroedEnemy(searchAnchor, w.Enemies)

		if enemy != nil {
			lb.inCombat = true
			// Swap to boss-specific YAML tree on boss fight entry
			if enemy.IsBoss && enemy.DefName != lb.currentBossName {
				pp := lb.Puppet
				res := m.treeReg.Resolve(pp.Player.ClassID, pp.Player.SpecID, enemy.DefName, pp.Profile)
				if res != nil {
					pp.SwapTree(res)
				}
				lb.currentBossName = enemy.DefName
			} else if !enemy.IsBoss && lb.currentBossName != "" {
				// Fighting trash after boss — restore default tree
				lb.Puppet.SwapTree(nil)
				lb.currentBossName = ""
			}
			tickCombat(lb, w, enemy, allPuppets, dt)
		} else {
			lb.inCombat = false
			if lb.currentBossName != "" {
				lb.Puppet.SwapTree(nil)
				lb.currentBossName = ""
			}
			if owner != nil {
				tickFollow(lb, owner, w, dt)
			}
		}
	}
}

// tickFollow moves the bot toward its owner at a comfortable distance.
func tickFollow(lb *LiveBot, owner *entity.Player, w *system.World, dt float32) {
	pp := lb.Puppet
	target := owner.Position.Add(lb.followOffset)

	dx := target.X - pp.Player.Position.X
	dz := target.Z - pp.Player.Position.Z
	dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))

	// Teleport if too far (e.g. zone transfer, elevator)
	if dist > followTeleportDist {
		pp.Player.Position = target
		pp.Player.Position.Y = owner.Position.Y
		emitPositionInput(pp, w)
		return
	}

	if dist > followStartDist {
		pp.MoveToward(target, dt)
	}

	// Face same direction as owner
	pp.Player.RotationY = owner.RotationY
	emitPositionInput(pp, w)
}

// tickCombat runs the puppet BT against the nearest enemy.
func tickCombat(lb *LiveBot, w *system.World, enemy *entity.Enemy, allPuppets []*bosstest.PlayerPuppet, dt float32) {
	pp := lb.Puppet

	// Look up enemy definition for ability resolution
	var bossDef *enemyai.EnemyDef
	if d, ok := enemyai.DefRegistry[enemy.DefName]; ok {
		bossDef = d
	}

	// Resolve active ability for telegraph tracking
	var activeAbil *ability.AbilityDef
	if bossDef != nil {
		if abil := bossDef.AbilityByIndex(enemy.ActiveAbility); abil != nil {
			resolved := bossDef.ResolveAbility(abil, enemy.Phase)
			activeAbil = &resolved
		}
	}

	ctx := &bosstest.PuppetContext{
		Puppet:     pp,
		World:      w,
		Boss:       enemy,
		BossDef:    bossDef,
		ActiveAbil: activeAbil,
		AllPuppets: allPuppets,
		Dt:         dt,
	}
	pp.Tick(ctx)
}

// findNearestAggroedEnemy returns the nearest alive, in-combat enemy within range.
func findNearestAggroedEnemy(p *entity.Player, enemies []*entity.Enemy) *entity.Enemy {
	var best *entity.Enemy
	bestDist := float32(math.MaxFloat32)

	for _, e := range enemies {
		if !e.Alive {
			continue
		}
		if e.State == entity.EnemyPatrol || e.State == entity.EnemyIdle {
			continue
		}
		dx := e.Position.X - p.Position.X
		dz := e.Position.Z - p.Position.Z
		dist := dx*dx + dz*dz
		if dist < combatDetectRange*combatDetectRange && dist < bestDist {
			bestDist = dist
			best = e
		}
	}
	return best
}

// respawnBot queues an arena respawn request, same as a real player clicking respawn.
func respawnBot(lb *LiveBot, w *system.World) {
	lb.deathTimer = 0
	w.InputQueue = append(w.InputQueue, system.InputMsg{
		PeerID:  lb.BotID,
		Opcode:  message.OpRespawnRequest,
		Payload: codec.EncodeRespawnRequest(0), // 0 = arena respawn
	})
}

// emitPositionInput pushes the puppet's position into the world input queue.
func emitPositionInput(pp *bosstest.PlayerPuppet, w *system.World) {
	pos := pp.Player.Position
	payload := codec.EncodePlayerInput(nil,
		pos.X, pos.Y, pos.Z,
		pp.Player.RotationY,
		w.TickNum,
		0, pp.Player.AimPitch,
	)
	w.InputQueue = append(w.InputQueue, system.InputMsg{
		PeerID:  pp.Player.ID,
		Opcode:  message.OpPlayerInput,
		Payload: payload,
	})
}

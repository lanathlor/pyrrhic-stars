package codec

// DecodedWorldState holds the parsed contents of a binary WorldState frame.
type DecodedWorldState struct {
	Tick        uint32
	Players     []DecodedPlayer
	Enemies     []DecodedEnemy
	Projectiles []DecodedProjectile
}

// DecodedFluxPool holds a single school's flux pool from a WorldState frame.
type DecodedFluxPool struct {
	Current, Max float32
}

// DecodedPlayer holds per-player fields from a WorldState frame.
type DecodedPlayer struct {
	PeerID               uint16
	PosX, PosY, PosZ     float32
	RotY                 float32
	Health               float32
	MaxHealth            float32
	State                uint8
	Class                string
	SpecName             string
	Username             string
	VisualState          uint8
	AimPitch             float32
	BuffFlags, Config    uint8
	Stamina, ShieldHP    float32
	Munitions, Resonance float32
	Flux, MaxFlux        float32
	OnslaughtStacks      uint8
	// Gunner Assault state
	Magazine, MagMax, StabilityQ, SteadinessQ, PressureStacks, EnhancedLoaded, AssaultFlags uint8
	SpeedMultQ                                                                              uint8 // quantized 0-255 → 0.0-1.0
	// Flux commitment pools (fixed order: bioarcanotechnic, biometabolic, frost, aerokinetic)
	FluxPools []DecodedFluxPool
}

// DecodedEnemy holds per-enemy fields from a WorldState frame.
type DecodedEnemy struct {
	Alive            bool
	EnemyID          uint16
	PosX, PosY, PosZ float32
	RotY             float32
	Health           float32
	State, Phase     uint8
	MaxHealth        float32
	DefName          string
}

// DecodedProjectile holds per-projectile fields from a WorldState frame.
type DecodedProjectile struct {
	ID               uint32
	PosX, PosY, PosZ float32
	DirX, DirY, DirZ float32
	Speed            float32
	AngularVelocity  float32
	VisualTag        string
}

// DecodeWorldState parses a binary WorldState frame produced by EncodeWorldState.
// Returns the decoded struct and true on success, or zero value and false if the
// buffer is malformed or too short.
func DecodeWorldState(buf []byte) (DecodedWorldState, bool) {
	var ws DecodedWorldState
	off := 0

	if len(buf) < 5 { // tick(4) + player_count(1)
		return ws, false
	}

	ws.Tick = getU32(buf[off:])
	off += 4

	// Players
	playerCount := int(buf[off])
	off++
	ws.Players = make([]DecodedPlayer, playerCount)
	for i := range playerCount {
		p, newOff, ok := decodePlayer(buf, off)
		if !ok {
			return ws, false
		}
		ws.Players[i] = p
		off = newOff
	}

	// Enemies
	if off >= len(buf) {
		return ws, false
	}
	enemyCount := int(buf[off])
	off++
	ws.Enemies = make([]DecodedEnemy, enemyCount)
	for i := range enemyCount {
		e, newOff, ok := decodeEnemy(buf, off)
		if !ok {
			return ws, false
		}
		ws.Enemies[i] = e
		off = newOff
	}

	// Projectiles
	if off >= len(buf) {
		return ws, false
	}
	projCount := int(buf[off])
	off++
	ws.Projectiles = make([]DecodedProjectile, projCount)
	for i := range projCount {
		p, newOff, ok := decodeProjectile(buf, off)
		if !ok {
			return ws, false
		}
		ws.Projectiles[i] = p
		off = newOff
	}

	return ws, true
}

// decodePlayer parses one player entry starting at buf[off].
// Returns the decoded player, new offset, and success.
func decodePlayer(buf []byte, off int) (DecodedPlayer, int, bool) {
	var p DecodedPlayer
	var ok bool
	p, off, ok = decodePlayerSpatial(buf, off, p)
	if !ok {
		return p, off, false
	}
	p, off, ok = decodePlayerResources(buf, off, p)
	if !ok {
		return p, off, false
	}
	return p, off, true
}

// decodePlayerSpatial decodes the position, health, state, class/spec/username, and visual_state fields.
func decodePlayerSpatial(buf []byte, off int, p DecodedPlayer) (DecodedPlayer, int, bool) {
	if off+27 > len(buf) { // minimum: u16 + 3*f32 + f32 + f32 + f32 + u8 = 27
		return p, off, false
	}
	p.PeerID = getU16(buf[off:])
	off += 2
	p.PosX = getF32(buf[off:])
	off += 4
	p.PosY = getF32(buf[off:])
	off += 4
	p.PosZ = getF32(buf[off:])
	off += 4
	p.RotY = getF32(buf[off:])
	off += 4
	p.Health = getF32(buf[off:])
	off += 4
	p.MaxHealth = getF32(buf[off:])
	off += 4
	p.State = buf[off]
	off++

	var ok bool
	p.Class, off, ok = decodeStr8(buf, off)
	if !ok {
		return p, off, false
	}
	p.SpecName, off, ok = decodeStr8(buf, off)
	if !ok {
		return p, off, false
	}
	p.Username, off, ok = decodeStr8(buf, off)
	if !ok {
		return p, off, false
	}

	// visual_state: u8
	if off >= len(buf) {
		return p, off, false
	}
	p.VisualState = buf[off]
	off++
	return p, off, true
}

// decodeStr8 reads a length-prefixed (1-byte) string from buf at off.
// Returns the string, new offset, and success.
func decodeStr8(buf []byte, off int) (string, int, bool) {
	if off >= len(buf) {
		return "", off, false
	}
	sLen := int(buf[off])
	off++
	if off+sLen > len(buf) {
		return "", off, false
	}
	s := string(buf[off : off+sLen])
	return s, off + sLen, true
}

// decodePlayerResources decodes aim_pitch, buff flags, config, resources, gunner state, and flux pools.
func decodePlayerResources(buf []byte, off int, p DecodedPlayer) (DecodedPlayer, int, bool) {
	var ok bool
	p, off, ok = decodePlayerStats(buf, off, p)
	if !ok {
		return p, off, false
	}
	p, off, ok = decodePlayerFluxPools(buf, off, p)
	if !ok {
		return p, off, false
	}
	return p, off, true
}

// decodePlayerStats decodes aim_pitch, flags, config, stamina/shield/munitions/resonance/flux,
// mastery stacks, gunner assault state, and speed multiplier.
func decodePlayerStats(buf []byte, off int, p DecodedPlayer) (DecodedPlayer, int, bool) {
	// aim_pitch, buff_flags, config, stamina, shield_hp, munitions, resonance, flux, maxflux, onslaught_stacks
	if off+31 > len(buf) {
		return p, off, false
	}
	p.AimPitch = getF32(buf[off:])
	off += 4
	p.BuffFlags = buf[off]
	off++
	p.Config = buf[off]
	off++
	p.Stamina = getF32(buf[off:])
	off += 4
	p.ShieldHP = getF32(buf[off:])
	off += 4
	p.Munitions = getF32(buf[off:])
	off += 4
	p.Resonance = getF32(buf[off:])
	off += 4
	p.Flux = getF32(buf[off:])
	off += 4
	p.MaxFlux = getF32(buf[off:])
	off += 4
	p.OnslaughtStacks = buf[off]
	off++
	// Gunner Assault state (7 bytes)
	if off+7 > len(buf) {
		return p, off, false
	}
	p.Magazine = buf[off]
	p.MagMax = buf[off+1]
	p.StabilityQ = buf[off+2]
	p.SteadinessQ = buf[off+3]
	p.PressureStacks = buf[off+4]
	p.EnhancedLoaded = buf[off+5]
	p.AssaultFlags = buf[off+6]
	off += 7
	// Speed multiplier (1 byte)
	if off >= len(buf) {
		return p, off, false
	}
	p.SpeedMultQ = buf[off]
	off++
	return p, off, true
}

// decodePlayerFluxPools decodes the variable-length flux commitment pool array.
func decodePlayerFluxPools(buf []byte, off int, p DecodedPlayer) (DecodedPlayer, int, bool) {
	if off >= len(buf) {
		return p, off, false
	}
	poolCount := int(buf[off])
	off++
	if poolCount > 0 {
		if off+poolCount*8 > len(buf) {
			return p, off, false
		}
		p.FluxPools = make([]DecodedFluxPool, poolCount)
		for pi := range poolCount {
			p.FluxPools[pi].Current = getF32(buf[off:])
			off += 4
			p.FluxPools[pi].Max = getF32(buf[off:])
			off += 4
		}
	}
	return p, off, true
}

// decodeEnemy parses one enemy entry starting at buf[off].
// Returns the decoded enemy, new offset, and success.
func decodeEnemy(buf []byte, off int) (DecodedEnemy, int, bool) {
	var e DecodedEnemy
	if off+23 > len(buf) {
		return e, off, false
	}
	e.Alive = buf[off] == 1
	off++
	e.EnemyID = getU16(buf[off:])
	off += 2
	e.PosX = getF32(buf[off:])
	off += 4
	e.PosY = getF32(buf[off:])
	off += 4
	e.PosZ = getF32(buf[off:])
	off += 4
	e.RotY = getF32(buf[off:])
	off += 4
	e.Health = getF32(buf[off:])
	off += 4
	e.State = buf[off]
	off++
	e.Phase = buf[off]
	off++
	e.MaxHealth = getF32(buf[off:])
	off += 4
	// def_name: str8
	if off >= len(buf) {
		return e, off, false
	}
	sLen := int(buf[off])
	off++
	if off+sLen > len(buf) {
		return e, off, false
	}
	e.DefName = string(buf[off : off+sLen])
	off += sLen
	// ranged_target(3*f32) + charge_dir(3*f32) + melee_cone_angle + melee_range
	if off+32 > len(buf) {
		return e, off, false
	}
	off += 24 // skip ranged_target + charge_dir
	off += 4  // melee_cone_angle
	off += 4  // melee_range
	return e, off, true
}

// decodeProjectile parses one projectile entry starting at buf[off].
// Returns the decoded projectile, new offset, and success.
func decodeProjectile(buf []byte, off int) (DecodedProjectile, int, bool) {
	var p DecodedProjectile
	if off+37 > len(buf) { // u32 + 6*f32 + f32 + f32 + u8 = 37
		return p, off, false
	}
	p.ID = getU32(buf[off:])
	off += 4
	p.PosX = getF32(buf[off:])
	off += 4
	p.PosY = getF32(buf[off:])
	off += 4
	p.PosZ = getF32(buf[off:])
	off += 4
	p.DirX = getF32(buf[off:])
	off += 4
	p.DirY = getF32(buf[off:])
	off += 4
	p.DirZ = getF32(buf[off:])
	off += 4
	p.Speed = getF32(buf[off:])
	off += 4
	p.AngularVelocity = getF32(buf[off:])
	off += 4
	tagLen := int(buf[off])
	off++
	if off+tagLen > len(buf) {
		return p, off, false
	}
	p.VisualTag = string(buf[off : off+tagLen])
	off += tagLen
	return p, off, true
}

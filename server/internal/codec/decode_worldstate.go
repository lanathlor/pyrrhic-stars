package codec

// DecodedWorldState holds the parsed contents of a binary WorldState frame.
type DecodedWorldState struct {
	Tick        uint32
	Players     []DecodedPlayer
	Enemies     []DecodedEnemy
	Projectiles []DecodedProjectile
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
	Flux                 float32
	OnslaughtStacks      uint8
	// Gunner Assault state
	Magazine, MagMax, StabilityQ, SteadinessQ, PressureStacks, EnhancedLoaded, AssaultFlags uint8
	SpeedMultQ uint8 // quantized 0-255 → 0.0-1.0
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
		if off+27 > len(buf) { // minimum: u16 + 3*f32 + f32 + f32 + f32 + u8 = 27
			return ws, false
		}
		p := &ws.Players[i]
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
		// class: str8
		if off >= len(buf) {
			return ws, false
		}
		sLen := int(buf[off])
		off++
		if off+sLen > len(buf) {
			return ws, false
		}
		p.Class = string(buf[off : off+sLen])
		off += sLen
		// spec: str8
		if off >= len(buf) {
			return ws, false
		}
		sLen = int(buf[off])
		off++
		if off+sLen > len(buf) {
			return ws, false
		}
		p.SpecName = string(buf[off : off+sLen])
		off += sLen
		// username: str8
		if off >= len(buf) {
			return ws, false
		}
		sLen = int(buf[off])
		off++
		if off+sLen > len(buf) {
			return ws, false
		}
		p.Username = string(buf[off : off+sLen])
		off += sLen
		// visual_state: u8
		if off >= len(buf) {
			return ws, false
		}
		p.VisualState = buf[off]
		off++
		// aim_pitch, buff_flags, config, stamina, shield_hp, munitions, resonance, onslaught_stacks
		if off+27 > len(buf) {
			return ws, false
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
		p.OnslaughtStacks = buf[off]
		off++
		// Gunner Assault state (7 bytes)
		if off+7 > len(buf) {
			return ws, false
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
			return ws, false
		}
		p.SpeedMultQ = buf[off]
		off++
	}

	// Enemies
	if off >= len(buf) {
		return ws, false
	}
	enemyCount := int(buf[off])
	off++
	ws.Enemies = make([]DecodedEnemy, enemyCount)
	for i := range enemyCount {
		if off+23 > len(buf) {
			return ws, false
		}
		e := &ws.Enemies[i]
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
			return ws, false
		}
		sLen := int(buf[off])
		off++
		if off+sLen > len(buf) {
			return ws, false
		}
		e.DefName = string(buf[off : off+sLen])
		off += sLen
		// ranged_target(3*f32) + charge_dir(3*f32) + melee_cone_angle + melee_range
		if off+32 > len(buf) {
			return ws, false
		}
		off += 24 // skip ranged_target + charge_dir
		off += 4  // melee_cone_angle
		off += 4  // melee_range
	}

	// Projectiles
	if off >= len(buf) {
		return ws, false
	}
	projCount := int(buf[off])
	off++
	ws.Projectiles = make([]DecodedProjectile, projCount)
	for i := range projCount {
		if off+37 > len(buf) { // u32 + 6*f32 + f32 + f32 + u8 = 37
			return ws, false
		}
		p := &ws.Projectiles[i]
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
			return ws, false
		}
		p.VisualTag = string(buf[off : off+tagLen])
		off += tagLen
	}

	return ws, true
}

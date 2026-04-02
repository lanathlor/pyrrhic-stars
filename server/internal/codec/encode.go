package codec

import "codex-online/server/internal/entity"

// EncodeWorldState serializes the arena tick snapshot (players, enemy, projectiles).
func EncodeWorldState(tick uint32, players []*entity.Player, enemy *entity.Enemy, projectiles []*entity.Projectile) []byte {
	buf := make([]byte, 0, 512)

	buf = appendU32(buf, tick)

	buf = append(buf, byte(len(players)))
	for _, p := range players {
		buf = appendU16(buf, p.PeerID)
		buf = appendF32(buf, p.Position.X)
		buf = appendF32(buf, p.Position.Y)
		buf = appendF32(buf, p.Position.Z)
		buf = appendF32(buf, p.RotationY)
		buf = appendF32(buf, p.Health)
		buf = append(buf, byte(p.State))
		buf = appendStr8(buf, p.ClassName)
		buf = appendStr8(buf, p.Username)
		buf = appendStr8(buf, p.AnimName)
		buf = appendF32(buf, p.AnimSpeed)
		buf = appendF32(buf, p.AimPitch)
	}

	if enemy != nil {
		if enemy.Alive {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}
		buf = appendU16(buf, enemy.ID)
		buf = appendF32(buf, enemy.Position.X)
		buf = appendF32(buf, enemy.Position.Y)
		buf = appendF32(buf, enemy.Position.Z)
		buf = appendF32(buf, enemy.RotationY)
		buf = appendF32(buf, enemy.Health)
		buf = append(buf, byte(enemy.State))
		buf = append(buf, byte(enemy.Phase))
		buf = appendF32(buf, enemy.RangedTargetPos.X)
		buf = appendF32(buf, enemy.RangedTargetPos.Y)
		buf = appendF32(buf, enemy.RangedTargetPos.Z)
		buf = appendF32(buf, enemy.ChargeDirection.X)
		buf = appendF32(buf, enemy.ChargeDirection.Y)
		buf = appendF32(buf, enemy.ChargeDirection.Z)
	} else {
		// Nil enemy (hub zones): 49 zero bytes — inert dead enemy at origin.
		buf = append(buf, make([]byte, 49)...)
	}

	buf = append(buf, byte(len(projectiles)))
	for _, proj := range projectiles {
		buf = appendU32(buf, proj.ID)
		buf = appendF32(buf, proj.Position.X)
		buf = appendF32(buf, proj.Position.Y)
		buf = appendF32(buf, proj.Position.Z)
		buf = appendF32(buf, proj.Direction.X)
		buf = appendF32(buf, proj.Direction.Y)
		buf = appendF32(buf, proj.Direction.Z)
	}

	return buf
}

// EncodeLobbyState serializes the lobby player list.
func EncodeLobbyState(players []LobbyPlayerInfo) []byte {
	buf := make([]byte, 0, 256)
	buf = append(buf, byte(len(players)))
	for _, p := range players {
		buf = appendU16(buf, p.PeerID)
		buf = appendStr8(buf, p.ClassName)
		if p.Ready {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}
	}
	return buf
}

// EncodeDamageEvent serializes a single damage event.
// Takes primitive fields to avoid a codec→combat import cycle.
func EncodeDamageEvent(targetPeerID, sourcePeerID uint16, amount, hitX, hitY, hitZ float32, sourceType uint8) []byte {
	buf := make([]byte, 0, 21)
	buf = appendU16(buf, targetPeerID)
	buf = appendU16(buf, sourcePeerID)
	buf = appendF32(buf, amount)
	buf = appendF32(buf, hitX)
	buf = appendF32(buf, hitY)
	buf = appendF32(buf, hitZ)
	buf = append(buf, sourceType)
	return buf
}

// EncodeGameFlow serializes a game flow event (fight start, boss dead, etc.).
func EncodeGameFlow(flowType uint8, text string) []byte {
	buf := []byte{flowType}
	textBytes := []byte(text)
	buf = append(buf, byte(len(textBytes)))
	buf = append(buf, textBytes...)
	return buf
}

// EncodePlayerInput serializes a client→server movement packet.
// Used by test clients to build OpPlayerInput payloads.
func EncodePlayerInput(posX, posY, posZ, rotY float32, tick uint32, animName string, animSpeed, aimPitch float32) []byte {
	buf := make([]byte, 0, 30)
	buf = appendF32(buf, posX)
	buf = appendF32(buf, posY)
	buf = appendF32(buf, posZ)
	buf = appendF32(buf, rotY)
	buf = appendU32(buf, tick)
	buf = appendStr8(buf, animName)
	buf = appendF32(buf, animSpeed)
	buf = appendF32(buf, aimPitch)
	return buf
}

// EncodeAbilityInput serializes a client→server ability activation packet.
// Used by test clients to build OpAbilityInput payloads.
func EncodeAbilityInput(action uint8, aimPitch float32) []byte {
	buf := []byte{action}
	return appendF32(buf, aimPitch)
}

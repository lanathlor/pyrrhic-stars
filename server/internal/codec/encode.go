package codec

import (
	"encoding/binary"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/item"
)

var encodeByte = make([]byte, 0, 65536)

// EncodeWorldState serializes the tick snapshot (players, enemies, projectiles, npcs).
func EncodeWorldState(tick uint32, players map[uint16]*entity.Player, enemies []*entity.Enemy, projectiles []*entity.Projectile, npcs ...[]*entity.NPC) []byte {
	var npcList []*entity.NPC
	if len(npcs) > 0 {
		npcList = npcs[0]
	}
	encodeByte = encodeByte[:0]
	return AppendEncodeWorldState(encodeByte, tick, players, enemies, projectiles, npcList)
}

// AppendEncodeWorldState serializes the tick snapshot into buf, growing it if necessary.
// Pass a pooled buffer to avoid per-call allocations in hot paths.
func AppendEncodeWorldState(buf []byte, tick uint32, players map[uint16]*entity.Player, enemies []*entity.Enemy, projectiles []*entity.Projectile, npcs []*entity.NPC) []byte {
	// Estimate needed capacity and grow if needed.
	// Per player: ~50 bytes. Per enemy: ~60 bytes. Per projectile: ~28 bytes.
	estCap := 512 + len(players)*50 + len(enemies)*60 + len(projectiles)*28
	if cap(buf) < estCap {
		newCap := cap(buf) * 2
		if newCap < estCap {
			newCap = estCap
		}
		newBuf := make([]byte, len(buf), newCap)
		copy(newBuf, buf)
		buf = newBuf
	}

	// Tick number
	buf = appendU32(buf, tick)

	buf = append(buf, byte(len(players)))
	for _, p := range players {
		buf = appendU16(buf, p.ID)
		buf = appendF32(buf, p.Position.X)
		buf = appendF32(buf, p.Position.Y)
		buf = appendF32(buf, p.Position.Z)
		buf = appendF32(buf, p.RotationY)
		buf = appendF32(buf, p.Health)
		buf = appendF32(buf, p.MaxHealth)
		buf = append(buf, byte(p.State))
		buf = appendStr8(buf, p.ClassName())
		buf = appendStr8(buf, p.Username)
		buf = append(buf, p.VisualState)
		buf = appendF32(buf, p.AimPitch)
		var flags uint8
		if p.HasBuff("overclock") {
			flags |= 0x01
		}
		if p.HasBuff("rechamber_buff") {
			flags |= 0x02
		}
		flags |= (p.GetAbilityPhase("rechamber") & 0x03) << 2
		if p.HasBuff("blade_swirl") {
			flags |= 0x10
		}
		if p.HasBuff("guard") || p.HasBuff("vg_parry") || p.HasBuff("vg_block") {
			flags |= 0x20
		}
		buf = append(buf, flags)
		buf = append(buf, byte(p.Config))
		buf = appendF32(buf, p.GetResource("stamina"))
		buf = appendF32(buf, p.GetResource("shield"))
		buf = appendF32(buf, p.GetResource("munitions"))
		buf = appendF32(buf, p.GetResource("resonance"))
	}

	buf = append(buf, byte(len(enemies)))
	for _, e := range enemies {
		if e.Alive {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}
		buf = appendU16(buf, e.ID)
		buf = appendF32(buf, e.Position.X)
		buf = appendF32(buf, e.Position.Y)
		buf = appendF32(buf, e.Position.Z)
		buf = appendF32(buf, e.RotationY)
		buf = appendF32(buf, e.Health)
		buf = append(buf, byte(e.State))
		buf = append(buf, byte(e.Phase))
		buf = appendF32(buf, e.MaxHealth)
		buf = appendStr8(buf, e.DefName)
		buf = appendF32(buf, e.RangedTargetPos.X)
		buf = appendF32(buf, e.RangedTargetPos.Y)
		buf = appendF32(buf, e.RangedTargetPos.Z)
		buf = appendF32(buf, e.ChargeDirection.X)
		buf = appendF32(buf, e.ChargeDirection.Y)
		buf = appendF32(buf, e.ChargeDirection.Z)
		buf = appendF32(buf, e.MeleeConeAngle)
		buf = appendF32(buf, e.MeleeRange)
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
		buf = appendF32(buf, proj.Speed)
		buf = appendF32(buf, proj.AngularVelocity)
		tag := proj.VisualTag
		if len(tag) > 255 {
			tag = tag[:255]
		}
		buf = append(buf, byte(len(tag)))
		buf = append(buf, tag...)
	}

	npcList := npcs
	buf = append(buf, byte(len(npcList)))
	for _, n := range npcList {
		buf = appendU16(buf, n.ID)
		buf = appendF32(buf, n.Position.X)
		buf = appendF32(buf, n.Position.Y)
		buf = appendF32(buf, n.Position.Z)
		buf = appendF32(buf, n.RotationY)
		buf = append(buf, byte(n.State))
		buf = appendStr8(buf, n.DefName)
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
		buf = appendStr8(buf, p.Username)
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
	return AppendEncodeDamageEvent(buf, targetPeerID, sourcePeerID, amount, hitX, hitY, hitZ, sourceType)
}

// AppendEncodeDamageEvent appends a damage event to buf.
// Pass a pooled buffer to avoid per-call allocations.
func AppendEncodeDamageEvent(buf []byte, targetPeerID, sourcePeerID uint16, amount, hitX, hitY, hitZ float32, sourceType uint8) []byte {
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
func EncodePlayerInput(senderBuffer []byte, posX, posY, posZ, rotY float32, tick uint32, visualState uint8, aimPitch float32) []byte {
	senderBuffer = appendF32(senderBuffer, posX)
	senderBuffer = appendF32(senderBuffer, posY)
	senderBuffer = appendF32(senderBuffer, posZ)
	senderBuffer = appendF32(senderBuffer, rotY)
	senderBuffer = appendU32(senderBuffer, tick)
	senderBuffer = append(senderBuffer, visualState)
	senderBuffer = appendF32(senderBuffer, aimPitch)
	return senderBuffer
}

// EncodeAbilityInput serializes a client→server ability activation packet.
// Used by test clients to build OpAbilityInput payloads.
func EncodeAbilityInput(action uint8, aimPitch float32, rotY ...float32) []byte {
	buf := []byte{action}
	buf = appendF32(buf, aimPitch)
	if len(rotY) > 0 {
		buf = appendF32(buf, rotY[0])
	} else {
		buf = appendF32(buf, 0)
	}
	return buf
}

// EncodeInteractInput serializes a client→server interact packet.
// Used by test clients to build OpInteractInput payloads.
func EncodeInteractInput(action uint8, className string) []byte {
	buf := []byte{action}
	nameBytes := []byte(className)
	buf = append(buf, byte(len(nameBytes)))
	buf = append(buf, nameBytes...)
	return buf
}

// EncodeRespawnRequest serializes a client→server respawn request.
// Used by test clients to build OpRespawnRequest payloads.
func EncodeRespawnRequest(respawnType uint8) []byte {
	return []byte{respawnType}
}

// EncodePeerID serializes a peer ID as 2 bytes big-endian.
func EncodePeerID(id uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, id)
	return b
}

// EncodeCharacterState builds the payload for OpCharacterState.
// Format: [charID:u32 LE][classLen:u8][class:...][nameLen:u8][name:...]
//
//	[posX:f32 LE][posY:f32 LE][posZ:f32 LE][rotY:f32 LE]
func EncodeCharacterState(c CharacterInfo) []byte {
	classBytes := []byte(c.ClassName)
	nameBytes := []byte(c.Name)
	buf := make([]byte, 0, 4+1+len(classBytes)+1+len(nameBytes)+16)

	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, c.ID)
	buf = append(buf, b4...)

	buf = append(buf, byte(len(classBytes)))
	buf = append(buf, classBytes...)
	buf = append(buf, byte(len(nameBytes)))
	buf = append(buf, nameBytes...)

	for _, f := range [4]float32{c.PosX, c.PosY, c.PosZ, c.RotY} {
		buf = appendF32(buf, f)
	}

	return buf
}

// EncodeCharacterList builds the payload for OpCharacterList.
// Format: [usernameLen:u8][username:...]
//
//	[count:u8] per char: [charID:u32 LE][classLen:u8][class:...][nameLen:u8][name:...]
//	                     [posX:f32 LE][posY:f32 LE][posZ:f32 LE][rotY:f32 LE]
//	[lastCharID:u32 LE]
func EncodeCharacterList(username string, chars []CharacterInfo, lastCharID uint32) []byte {
	buf := make([]byte, 0, 256)

	usernameBytes := []byte(username)
	buf = append(buf, byte(len(usernameBytes)))
	buf = append(buf, usernameBytes...)

	buf = append(buf, byte(len(chars)))
	for _, c := range chars {
		b4 := make([]byte, 4)
		binary.LittleEndian.PutUint32(b4, c.ID)
		buf = append(buf, b4...)

		classBytes := []byte(c.ClassName)
		buf = append(buf, byte(len(classBytes)))
		buf = append(buf, classBytes...)

		nameBytes := []byte(c.Name)
		buf = append(buf, byte(len(nameBytes)))
		buf = append(buf, nameBytes...)

		for _, f := range [4]float32{c.PosX, c.PosY, c.PosZ, c.RotY} {
			buf = appendF32(buf, f)
		}
	}

	lastID := make([]byte, 4)
	binary.LittleEndian.PutUint32(lastID, lastCharID)
	buf = append(buf, lastID...)

	return buf
}

// EncodeCharacterError builds the payload for OpCharacterError.
// Format: [code:u8][msgLen:u8][msg:...]
func EncodeCharacterError(code uint8, msg string) []byte {
	msgBytes := []byte(msg)
	buf := make([]byte, 0, 2+len(msgBytes))
	buf = append(buf, code)
	buf = append(buf, byte(len(msgBytes)))
	buf = append(buf, msgBytes...)
	return buf
}

// EncodeGroupState builds the payload for OpGroupState.
// Format: [groupID:u32 LE][leaderPeerID:u16 LE][count:u8]
//
//	per member: [peerID:u16 LE][nameLen:u8][name:...]
func EncodeGroupState(groupID uint32, leaderPeerID uint16, members []GroupMemberInfo) []byte {
	buf := make([]byte, 0, 128)
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, groupID)
	buf = append(buf, b4...)

	b2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(b2, leaderPeerID)
	buf = append(buf, b2...)

	buf = append(buf, byte(len(members)))
	for _, m := range members {
		b2 := make([]byte, 2)
		binary.LittleEndian.PutUint16(b2, m.PeerID)
		buf = append(buf, b2...)
		nameBytes := []byte(m.Username)
		buf = append(buf, byte(len(nameBytes)))
		buf = append(buf, nameBytes...)
	}
	return buf
}

// EncodeGroupError builds the payload for OpGroupError.
// Format: [code:u8(1)][msgLen:u8][msg:...]
func EncodeGroupError(errMsg string) []byte {
	buf := []byte{1} // error code 1 = generic
	msgBytes := []byte(errMsg)
	buf = append(buf, byte(len(msgBytes)))
	buf = append(buf, msgBytes...)
	return buf
}

// EncodeGroupInviteRecv builds the payload for OpGroupInviteRecv.
// Format: [groupID:u32 LE][nameLen:u8][leaderName:...]
func EncodeGroupInviteRecv(groupID uint32, leaderName string) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, groupID)
	nameBytes := []byte(leaderName)
	buf = append(buf, byte(len(nameBytes)))
	buf = append(buf, nameBytes...)
	return buf
}

// EncodeEmptyGroupState builds the payload for an empty OpGroupState (not in a group).
// Format: [groupID:u32(0)][leaderPeerID:u16(0)][count:u8(0)]
func EncodeEmptyGroupState() []byte {
	return make([]byte, 7) // 4 bytes group_id(0) + 2 bytes leader(0) + 1 byte count(0)
}

// InventoryItemInfo carries item data for inventory encoding (decoupled from persistence).
type InventoryItemInfo struct {
	ItemID    uint32
	SlotID    uint8
	DefID     string
	Name      string
	ILvl      uint16
	StatLines []InventoryStatLine
}

// InventoryStatLine carries a single stat line for encoding.
type InventoryStatLine struct {
	Stat  uint8
	Value float32
}

// EncodeInventoryState builds the payload for OpInventoryState.
// Format: [equip_count:u8] per: [slotID:u8][itemID:u32][defID:str8][ilvl:u16][name:str8][stat_count:u8][stat_id:u8 + value:f32]...
//
//	[bag_count:u8]   per: [slotID:u8][itemID:u32][defID:str8][ilvl:u16][name:str8][stat_count:u8][stat_id:u8 + value:f32]...
//	[6x computed_stat:f32] (hull, output, plating, tempo, identity, mastery)
func EncodeInventoryState(equipped []InventoryItemInfo, bag []InventoryItemInfo, stats item.Stats) []byte {
	buf := make([]byte, 0, 512)

	// Equipped items.
	buf = append(buf, byte(len(equipped)))
	for _, it := range equipped {
		buf = encodeInventoryItem(buf, it)
	}

	// Bag items.
	buf = append(buf, byte(len(bag)))
	for _, it := range bag {
		buf = encodeInventoryItem(buf, it)
	}

	// Computed stats (6 floats).
	buf = appendF32(buf, stats.Hull)
	buf = appendF32(buf, stats.Output)
	buf = appendF32(buf, stats.Plating)
	buf = appendF32(buf, stats.Tempo)
	buf = appendF32(buf, stats.Identity)
	buf = appendF32(buf, stats.Mastery)

	return buf
}

func encodeInventoryItem(buf []byte, it InventoryItemInfo) []byte {
	buf = append(buf, it.SlotID)
	buf = appendU32(buf, it.ItemID)
	buf = appendStr8(buf, it.DefID)
	buf = appendU16(buf, it.ILvl)
	buf = appendStr8(buf, it.Name)
	buf = append(buf, byte(len(it.StatLines)))
	for _, sl := range it.StatLines {
		buf = append(buf, sl.Stat)
		buf = appendF32(buf, sl.Value)
	}
	return buf
}

// DecodeEquipItem decodes the payload for OpEquipItem.
// Returns itemID and slotID.
func DecodeEquipItem(payload []byte) (itemID uint32, slotID uint8, ok bool) {
	if len(payload) < 5 {
		return 0, 0, false
	}
	itemID = binary.LittleEndian.Uint32(payload[0:4])
	slotID = payload[4]
	return itemID, slotID, true
}

// DecodeUnequipItem decodes the payload for OpUnequipItem.
// Returns slotID.
func DecodeUnequipItem(payload []byte) (slotID uint8, ok bool) {
	if len(payload) < 1 {
		return 0, false
	}
	return payload[0], true
}

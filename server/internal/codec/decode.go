package codec

// DecodePlayerInput parses a client movement packet.
// Returns ok=false if the payload is too short.
func DecodePlayerInput(payload []byte) (msg PlayerInputMsg, ok bool) {
	if len(payload) < 16 {
		return msg, false
	}
	msg.PosX = getF32(payload[0:4])
	msg.PosY = getF32(payload[4:8])
	msg.PosZ = getF32(payload[8:12])
	msg.RotY = getF32(payload[12:16])
	if len(payload) >= 20 {
		msg.Tick = getU32(payload[16:20])
	}
	off := 20
	if off < len(payload) {
		msg.VisualState = payload[off]
		off++
		if off+4 <= len(payload) {
			msg.AimPitch = getF32(payload[off : off+4])
		}
	}
	return msg, true
}

// DecodeAbilityInput parses an ability activation packet. Returns nil if too short.
func DecodeAbilityInput(payload []byte) *AbilityInputMsg {
	if len(payload) < 1 {
		return nil
	}
	msg := &AbilityInputMsg{
		Action: payload[0],
	}
	if len(payload) >= 5 {
		msg.AimPitch = getF32(payload[1:5])
	}
	if len(payload) >= 9 {
		msg.RotY = getF32(payload[5:9])
	}
	if len(payload) >= 11 {
		msg.TargetPeerID = getU16(payload[9:11])
	}
	return msg
}

// DecodeInteractInput parses a lobby/interact packet. Returns nil if too short.
func DecodeInteractInput(payload []byte) (msg InteractInputMsg, ok bool) {
	if len(payload) < 1 {
		return msg, false
	}
	msg.Action = payload[0]
	if len(payload) >= 3 {
		nameLen := int(payload[1])
		if len(payload) >= 2+nameLen {
			msg.ClassName = unsafeString(payload[2 : 2+nameLen])
		}
	}
	return msg, true
}

// DecodeRespawnRequest parses a respawn request. Returns the respawn type and ok=true,
// or 0,false if the payload is too short.
func DecodeRespawnRequest(payload []byte) (uint8, bool) {
	if len(payload) < 1 {
		return 0, false
	}
	return payload[0], true
}

// DecodeSetLoadout parses a client loadout change.
// Wire format: [slot0:str8][slot1:str8]...[slot5:str8]
func DecodeSetLoadout(payload []byte) ([6]string, bool) {
	var slots [6]string
	off := 0
	for i := range 6 {
		if off >= len(payload) {
			return slots, false
		}
		sLen := int(payload[off])
		off++
		if off+sLen > len(payload) {
			return slots, false
		}
		slots[i] = string(payload[off : off+sLen])
		off += sLen
	}
	return slots, true
}

// FluxCommitEntry holds one school's commitment percentage.
type FluxCommitEntry struct {
	School     string
	Percentage uint8 // 0–100
}

// DecodeFluxCommitment parses a client flux commitment change.
// Wire format: [count:u8][per entry: school:str8 + pct:u8]
func DecodeFluxCommitment(payload []byte) ([]FluxCommitEntry, bool) {
	if len(payload) < 1 {
		return nil, false
	}
	count := int(payload[0])
	off := 1
	entries := make([]FluxCommitEntry, 0, count)
	for range count {
		if off >= len(payload) {
			return nil, false
		}
		sLen := int(payload[off])
		off++
		if off+sLen > len(payload) {
			return nil, false
		}
		school := string(payload[off : off+sLen])
		off += sLen
		if off >= len(payload) {
			return nil, false
		}
		pct := payload[off]
		off++
		entries = append(entries, FluxCommitEntry{School: school, Percentage: pct})
	}
	return entries, true
}

// EncodeFluxCommitState serializes flux commitment state.
// Wire format: [count:u8][per entry: school:str8 + pct:u8]
func EncodeFluxCommitState(entries []FluxCommitEntry) []byte {
	buf := make([]byte, 0, 64)
	buf = append(buf, byte(len(entries)))
	for _, e := range entries {
		buf = appendStr8(buf, e.School)
		buf = append(buf, e.Percentage)
	}
	return buf
}

// DecodeSavePreset parses a save-preset request.
// Wire format: [name:str8][slot0:str8]...[slot5:str8][commitment:str8]
func DecodeSavePreset(payload []byte) (name string, slots [6]string, commitment string, ok bool) {
	off := 0
	// name
	if off >= len(payload) {
		return "", slots, "", false
	}
	nLen := int(payload[off])
	off++
	if off+nLen > len(payload) {
		return "", slots, "", false
	}
	name = string(payload[off : off+nLen])
	off += nLen
	// 6 slots
	for i := range 6 {
		if off >= len(payload) {
			return "", slots, "", false
		}
		sLen := int(payload[off])
		off++
		if off+sLen > len(payload) {
			return "", slots, "", false
		}
		slots[i] = string(payload[off : off+sLen])
		off += sLen
	}
	// commitment
	if off >= len(payload) {
		return "", slots, "", false
	}
	cLen := int(payload[off])
	off++
	if off+cLen > len(payload) {
		return "", slots, "", false
	}
	commitment = string(payload[off : off+cLen])
	return name, slots, commitment, true
}

// DecodeDeletePreset parses a delete-preset request.
// Wire format: [preset_id:u32 LE]
func DecodeDeletePreset(payload []byte) (uint32, bool) {
	if len(payload) < 4 {
		return 0, false
	}
	return getU32(payload[0:4]), true
}

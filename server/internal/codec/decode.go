package codec

// DecodePlayerInput parses a client movement packet. Returns nil if too short.
func DecodePlayerInput(payload []byte) *PlayerInputMsg {
	if len(payload) < 16 {
		return nil
	}
	msg := &PlayerInputMsg{
		PosX: getF32(payload[0:4]),
		PosY: getF32(payload[4:8]),
		PosZ: getF32(payload[8:12]),
		RotY: getF32(payload[12:16]),
	}
	if len(payload) >= 20 {
		msg.Tick = getU32(payload[16:20])
	}
	off := 20
	if off < len(payload) {
		animLen := int(payload[off])
		off++
		if off+animLen <= len(payload) {
			msg.AnimName = string(payload[off : off+animLen])
			off += animLen
		}
		if off+4 <= len(payload) {
			msg.AnimSpeed = getF32(payload[off : off+4])
			off += 4
		}
		if off+4 <= len(payload) {
			msg.AimPitch = getF32(payload[off : off+4])
		}
	}
	return msg
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
	return msg
}

// DecodeInteractInput parses a lobby/interact packet. Returns nil if too short.
func DecodeInteractInput(payload []byte) *InteractInputMsg {
	if len(payload) < 1 {
		return nil
	}
	msg := &InteractInputMsg{
		Action: payload[0],
	}
	if len(payload) >= 3 {
		nameLen := int(payload[1])
		if len(payload) >= 2+nameLen {
			msg.ClassName = string(payload[2 : 2+nameLen])
		}
	}
	return msg
}

// DecodeRespawnRequest parses a respawn request. Returns the respawn type and ok=true,
// or 0,false if the payload is too short.
func DecodeRespawnRequest(payload []byte) (uint8, bool) {
	if len(payload) < 1 {
		return 0, false
	}
	return payload[0], true
}

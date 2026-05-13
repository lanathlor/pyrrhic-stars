package codec

import (
	"encoding/binary"
	"math"
)

// EncodeDebugInfo builds the OpDebugInfo payload: [str8: def_name][uint8: count][str8: ability_id]...
func EncodeDebugInfo(defName string, abilityIDs []string) []byte {
	buf := appendStr8(nil, defName)
	buf = append(buf, byte(len(abilityIDs)))
	for _, id := range abilityIDs {
		buf = appendStr8(buf, id)
	}
	return buf
}

// DecodeDebugStr8 reads a str8 (length-prefixed string) from the payload.
// Used by force-cast and repeat-ability opcodes.
func DecodeDebugStr8(payload []byte) (string, bool) {
	if len(payload) < 1 {
		return "", false
	}
	n := int(payload[0])
	if len(payload) < 1+n {
		return "", false
	}
	return string(payload[1 : 1+n]), true
}

// DecodeDebugPhase reads a uint8 phase number from the payload.
func DecodeDebugPhase(payload []byte) (uint8, bool) {
	if len(payload) < 1 {
		return 0, false
	}
	return payload[0], true
}

// DecodeDebugGodMode reads a bool (0=off, 1=on) from the payload.
func DecodeDebugGodMode(payload []byte) (bool, bool) {
	if len(payload) < 1 {
		return false, false
	}
	return payload[0] != 0, true
}

// DecodeDebugTimeScale reads a float32 time scale from the payload.
func DecodeDebugTimeScale(payload []byte) (float32, bool) {
	if len(payload) < 4 {
		return 0, false
	}
	bits := binary.LittleEndian.Uint32(payload[0:4])
	return math.Float32frombits(bits), true
}

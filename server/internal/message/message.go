package message

import (
	"encoding/binary"
	"errors"
)

// Opcodes are uint16, big-endian. Grouped by range:
//   0x0001–0x000F  game state sync (high frequency)
//   0x0010–0x001F  lobby
//   0x0020–0x002F  game flow
//   0xFF00–0xFFFF  zone management (server-handled)
const (
	// Game sync — broadcast excluding sender (unreliable-style).
	OpPlayerSync      uint16 = 0x0001
	OpEnemySync       uint16 = 0x0002
	OpDamage          uint16 = 0x0003
	OpNetFlash        uint16 = 0x0004
	OpProjectileSpawn uint16 = 0x0005

	// Lobby — broadcast including sender (call_local).
	OpClassSelect uint16 = 0x0010
	OpReadyState  uint16 = 0x0011
	OpPlayerInfo  uint16 = 0x0012

	// Game flow — broadcast including sender (call_local).
	OpSpawnPlayers uint16 = 0x0020
	OpStartFight   uint16 = 0x0021
	OpShowResult   uint16 = 0x0022
	OpResetReady   uint16 = 0x0023

	// Zone management — server-handled, never relayed.
	OpJoinZone         uint16 = 0xFF00
	OpZoneJoined       uint16 = 0xFF01
	OpPeerConnected    uint16 = 0xFF02
	OpPeerDisconnected uint16 = 0xFF03
)

// HeaderSize is the fixed-size message header: 2 bytes opcode + 2 bytes sender ID.
const HeaderSize = 4

var errMessageTooShort = errors.New("message too short")

// Encode builds a wire message: [opcode:2][senderID:2][payload...].
func Encode(opcode, senderID uint16, payload []byte) []byte {
	msg := make([]byte, HeaderSize+len(payload))
	binary.BigEndian.PutUint16(msg[0:2], opcode)
	binary.BigEndian.PutUint16(msg[2:4], senderID)
	if len(payload) > 0 {
		copy(msg[HeaderSize:], payload)
	}
	return msg
}

// Decode splits a wire message into opcode, sender ID, and payload.
func Decode(data []byte) (opcode, senderID uint16, payload []byte, err error) {
	if len(data) < HeaderSize {
		return 0, 0, nil, errMessageTooShort
	}
	opcode = binary.BigEndian.Uint16(data[0:2])
	senderID = binary.BigEndian.Uint16(data[2:4])
	payload = data[HeaderSize:]
	return opcode, senderID, payload, nil
}

// BroadcastExcludeSender returns true for opcodes where the sender should NOT
// receive their own message back (position/state sync, visual effects).
func BroadcastExcludeSender(opcode uint16) bool {
	switch opcode {
	case OpPlayerSync, OpEnemySync, OpNetFlash:
		return true
	default:
		return false
	}
}

// IsServerHandled returns true for opcodes that the server processes directly
// and does not relay to other clients.
func IsServerHandled(opcode uint16) bool {
	return opcode >= 0xFF00
}

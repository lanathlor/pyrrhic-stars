package message

import (
	"encoding/binary"
	"errors"
)

// Opcodes are uint16, big-endian. Grouped by range:
//
//	0x0001–0x000F  legacy game state sync (deprecated — kept for compat during migration)
//	0x0010–0x001F  legacy lobby (deprecated)
//	0x0020–0x002F  legacy game flow (deprecated)
//	0x0030–0x003F  client → server inputs
//	0x0040–0x004F  server → client authoritative state
//	0xFF00–0xFFFF  zone management (server-handled)
const (
	// Legacy relay opcodes (will be removed after full migration).
	OpPlayerSync      uint16 = 0x0001
	OpEnemySync       uint16 = 0x0002
	OpDamage          uint16 = 0x0003
	OpNetFlash        uint16 = 0x0004
	OpProjectileSpawn uint16 = 0x0005

	OpClassSelect uint16 = 0x0010
	OpReadyState  uint16 = 0x0011
	OpPlayerInfo  uint16 = 0x0012

	OpSpawnPlayers uint16 = 0x0020
	OpStartFight   uint16 = 0x0021
	OpShowResult   uint16 = 0x0022
	OpResetReady   uint16 = 0x0023

	// Client → Server inputs.
	OpPlayerInput    uint16 = 0x0030 // movement + continuous actions
	OpAbilityInput   uint16 = 0x0031 // discrete ability activation
	OpInteractInput  uint16 = 0x0032 // lobby actions (class select, ready)
	OpRespawnRequest uint16 = 0x0033 // death respawn (arena or hub)

	// Server → Client authoritative state.
	OpWorldState     uint16 = 0x0040 // full entity snapshot per tick
	OpEntitySpawn    uint16 = 0x0041 // new entity appeared
	OpEntityDespawn  uint16 = 0x0042 // entity removed
	OpDamageEvent    uint16 = 0x0043 // visual damage event (for effects)
	OpGameFlowEvent  uint16 = 0x0044 // fight start, result, phase transition
	OpLobbyState     uint16 = 0x0045 // lobby player list, ready states
	OpInputAck       uint16 = 0x0046 // acknowledges client input tick

	// Social / group — client → server (gateway-handled).
	OpGroupCreate      uint16 = 0x0050
	OpGroupInvite      uint16 = 0x0051
	OpGroupInviteReply uint16 = 0x0052
	OpGroupLeave       uint16 = 0x0053
	OpGroupKick        uint16 = 0x0054
	OpEnterPortal      uint16 = 0x0055

	// Social / group — server → client.
	OpGroupState       uint16 = 0x0060
	OpGroupInviteRecv  uint16 = 0x0061
	OpGroupError       uint16 = 0x0062
	OpHubState         uint16 = 0x0063
	OpPlayerNames      uint16 = 0x0064

	// Zone management — server-handled, never relayed.
	OpJoinZone         uint16 = 0xFF00
	OpZoneJoined       uint16 = 0xFF01
	OpPeerConnected    uint16 = 0xFF02
	OpPeerDisconnected uint16 = 0xFF03
	OpSetUsername      uint16 = 0xFF04
	OpRequestZoneTransfer uint16 = 0xFF05
	OpZoneTransfer     uint16 = 0xFF06
	OpCharacterState   uint16 = 0xFF07 // server → client: saved character data after auth
	OpCharacterList    uint16 = 0xFF08 // server → client: all characters after auth
	OpSelectCharacter  uint16 = 0xFF09 // client → server: pick character to play
	OpCreateCharacter  uint16 = 0xFF0A // client → server: create new character
	OpCharacterError   uint16 = 0xFF0B // server → client: character operation error
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

// AppendEncode writes a wire message into buf, growing it if necessary.
// Pass a pooled buffer to avoid per-call allocations in hot paths.
func AppendEncode(buf []byte, opcode, senderID uint16, payload []byte) []byte {
	// Build header inline
	header := [HeaderSize]byte{
		byte(opcode >> 8), byte(opcode),
		byte(senderID >> 8), byte(senderID),
	}
	// Ensure capacity: current len + header + payload
	needed := len(buf) + HeaderSize + len(payload)
	if cap(buf) < needed {
		// Grow: at least double, or exact if large
		newCap := cap(buf) * 2
		if newCap < needed {
			newCap = needed
		}
		newBuf := make([]byte, len(buf), newCap)
		copy(newBuf, buf)
		buf = newBuf
	}
	// Extend with header then payload
	buf = append(buf, header[:]...)
	buf = append(buf, payload...)
	return buf
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
	return opcode >= 0xFF00 || IsGroupRelated(opcode)
}

// IsGroupRelated returns true for group/social opcodes (0x0050–0x005F).
func IsGroupRelated(opcode uint16) bool {
	return opcode >= 0x0050 && opcode <= 0x005F
}

// IsClientInput returns true for opcodes that are client-to-server input messages.
// These are routed to the zone simulation, not relayed.
func IsClientInput(opcode uint16) bool {
	return opcode >= 0x0030 && opcode <= 0x003F
}

// Game flow event types sent within OpGameFlowEvent payload.
const (
	FlowSpawnPlayers   uint8 = 1
	FlowFightStart     uint8 = 2
	FlowShowResult     uint8 = 3
	FlowPhaseTransition uint8 = 4
	FlowReturnLobby    uint8 = 5
	FlowReturnHub      uint8 = 6
	FlowBossDead       uint8 = 7
	FlowAllDead        uint8 = 8
	FlowBossActivated  uint8 = 9
	FlowBossReset      uint8 = 10
)

// Interact input action types sent within OpInteractInput payload.
const (
	InteractClassSelect uint8 = 0
	InteractReadyToggle uint8 = 1
	InteractExitPortal  uint8 = 2
)

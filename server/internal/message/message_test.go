package message

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		opcode   uint16
		senderID uint16
		payload  []byte
	}{
		{"empty payload", OpPlayerSync, 1, nil},
		{"with payload", OpDamage, 42, []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		{"zone opcode", OpJoinZone, 0, []byte("arena")},
		{"max values", 0xFFFF, 0xFFFF, []byte{0xFF}},
		{"zero values", 0, 0, nil},
		{"large payload", OpEnemySync, 3, bytes.Repeat([]byte{0x42}, 256)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := Encode(tt.opcode, tt.senderID, tt.payload)

			if len(encoded) != HeaderSize+len(tt.payload) {
				t.Fatalf("encoded length = %d, want %d", len(encoded), HeaderSize+len(tt.payload))
			}

			opcode, senderID, payload, err := Decode(encoded)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}
			if opcode != tt.opcode {
				t.Errorf("opcode = 0x%04X, want 0x%04X", opcode, tt.opcode)
			}
			if senderID != tt.senderID {
				t.Errorf("senderID = %d, want %d", senderID, tt.senderID)
			}
			if !bytes.Equal(payload, tt.payload) {
				t.Errorf("payload = %v, want %v", payload, tt.payload)
			}
		})
	}
}

func TestDecodeTooShort(t *testing.T) {
	for _, data := range [][]byte{nil, {}, {0x00}, {0x00, 0x01}, {0x00, 0x01, 0x02}} {
		_, _, _, err := Decode(data)
		if err == nil {
			t.Errorf("Decode(%v) should fail for data shorter than %d bytes", data, HeaderSize)
		}
	}
}

func TestBroadcastExcludeSender(t *testing.T) {
	excluded := []uint16{OpPlayerSync, OpEnemySync, OpNetFlash}
	for _, op := range excluded {
		if !BroadcastExcludeSender(op) {
			t.Errorf("BroadcastExcludeSender(0x%04X) = false, want true", op)
		}
	}

	included := []uint16{OpDamage, OpProjectileSpawn, OpClassSelect, OpReadyState,
		OpPlayerInfo, OpSpawnPlayers, OpStartFight, OpShowResult, OpResetReady}
	for _, op := range included {
		if BroadcastExcludeSender(op) {
			t.Errorf("BroadcastExcludeSender(0x%04X) = true, want false", op)
		}
	}
}

func TestIsServerHandled(t *testing.T) {
	serverOps := []uint16{OpJoinZone, OpZoneJoined, OpPeerConnected, OpPeerDisconnected}
	for _, op := range serverOps {
		if !IsServerHandled(op) {
			t.Errorf("IsServerHandled(0x%04X) = false, want true", op)
		}
	}

	gameOps := []uint16{OpPlayerSync, OpDamage, OpClassSelect, OpSpawnPlayers}
	for _, op := range gameOps {
		if IsServerHandled(op) {
			t.Errorf("IsServerHandled(0x%04X) = true, want false", op)
		}
	}
}

func TestIsClientInput(t *testing.T) {
	clientInputs := []uint16{OpPlayerInput, OpAbilityInput, OpInteractInput, OpRespawnRequest}
	for _, op := range clientInputs {
		if !IsClientInput(op) {
			t.Errorf("IsClientInput(0x%04X) = false, want true", op)
		}
	}

	nonInputs := []uint16{OpWorldState, OpDamageEvent, OpJoinZone, OpGroupCreate, OpPlayerSync}
	for _, op := range nonInputs {
		if IsClientInput(op) {
			t.Errorf("IsClientInput(0x%04X) = true, want false", op)
		}
	}
}

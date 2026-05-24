package codec

import (
	"encoding/binary"
	"math"
	"unsafe"
)

// --- Decode output types (zone input handlers consume these) ---

// PlayerInputMsg is the decoded client movement packet.
type PlayerInputMsg struct {
	PosX, PosY, PosZ, RotY float32
	Tick                   uint32
	VisualState            uint8
	AimPitch               float32
}

// AbilityInputMsg is the decoded ability activation packet.
type AbilityInputMsg struct {
	Action       uint8
	AimPitch     float32
	RotY         float32
	TargetPeerID uint16 // ally target for heals (0 = no ally target)
}

// InteractInputMsg is the decoded lobby/interact packet.
type InteractInputMsg struct {
	Action    uint8
	ClassName string
}

// --- Encode input types (zone builds these, codec serializes) ---

// LobbyPlayerInfo carries per-player lobby data for encoding.
type LobbyPlayerInfo struct {
	PeerID    uint16
	ClassName string
	SpecName  string
	Username  string
	Ready     bool
}

// CharacterInfo carries character data for encoding (decoupled from persistence).
type CharacterInfo struct {
	ID                     uint32
	ClassName              string
	Name                   string
	PosX, PosY, PosZ, RotY float32
}

// GroupMemberInfo carries per-member data for group state encoding.
type GroupMemberInfo struct {
	PeerID   uint16
	Username string
}

// AbilityCatalogEntry is a single ability for wire transmission.
type AbilityCatalogEntry struct {
	ID          string
	Name        string
	School      string
	AbilityType string
	Delivery    string
	FluxCost    string
	Description string
	Cooldown    float32
	Implemented bool
	Affinity    string // "primary", "secondary", "off"

	// Exact stats from AbilityDef (0 = not applicable).
	FluxAmount   float32
	BaseHeal     float32
	BaseDamage   float32
	Range        float32
	GCD          float32
	CommitTime   float32
	ZoneRadius   float32
	ZoneDuration float32
	ZoneHealTick float32
	Sustain      bool
}

// PresetInfo holds a loadout preset for wire encoding.
type PresetInfo struct {
	ID         uint32
	Name       string
	Slots      [6]string
	Commitment string
}

// --- Private wire helpers ---
// These helpers write primitives directly into buf without allocating.
// Each calls append to ensure capacity (which may trigger a growth), then
// writes the bytes in-place. After the first growth, subsequent calls reuse
// the extra capacity — no per-call heap allocation.

func appendF32(buf []byte, v float32) []byte {
	buf = append(buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(buf[len(buf)-4:], math.Float32bits(v))
	return buf
}

func appendU16(buf []byte, v uint16) []byte {
	buf = append(buf, 0, 0)
	binary.LittleEndian.PutUint16(buf[len(buf)-2:], v)
	return buf
}

func appendU32(buf []byte, v uint32) []byte {
	buf = append(buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(buf[len(buf)-4:], v)
	return buf
}

func appendStr8(buf []byte, s string) []byte {
	// string→[]byte conversion is unavoidable; we only do one allocation here.
	b := []byte(s)
	buf = append(buf, byte(len(b)))
	return append(buf, b...)
}

func appendStr16(buf []byte, s string) []byte {
	b := []byte(s)
	buf = append(buf, 0, 0)
	binary.LittleEndian.PutUint16(buf[len(buf)-2:], uint16(len(b)))
	return append(buf, b...)
}

func getF32(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}

func getU16(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

func getU32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

// unsafeString returns a string that shares the underlying bytes of b.
// The caller must ensure b is not modified while the string is in use.
func unsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

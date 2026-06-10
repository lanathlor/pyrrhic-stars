package overflux

import (
	"encoding/binary"
	"fmt"
)

// ConditionID is a string key identifying an overflux condition.
type ConditionID string

const (
	CondEnemyHP  ConditionID = "enemy_hp"
	CondTempered ConditionID = "tempered" // boss: different BT, cooldown overrides, new abilities
	CondFrenzied ConditionID = "frenzied" // mobs: different BT, cooldown overrides, new abilities
	CondVolatile ConditionID = "volatile" // boss: pattern/damage replacement on abilities
)

// ConditionDef describes a single toggleable condition with ranked severity.
type ConditionDef struct {
	ID           ConditionID
	Name         string // display name (e.g. "Fortified")
	Description  string
	MaxRank      int // 1-5
	ScorePerRank int // overflux points added per rank
}

// ActiveCondition is one enabled condition with its chosen rank.
type ActiveCondition struct {
	ID   ConditionID
	Rank int // 1-5 (0 = off, should not appear in active list)
}

// State holds the active overflux conditions for a zone instance.
type State struct {
	Conditions []ActiveCondition
	TotalScore int
}

// Registry is the global list of available overflux conditions.
var Registry = []ConditionDef{
	{ID: CondEnemyHP, Name: "Fortified", Description: "Increases enemy max health", MaxRank: 5, ScorePerRank: 4},
	{ID: CondTempered, Name: "Tempered", Description: "Boss uses a smarter behavior tree with new abilities", MaxRank: 1, ScorePerRank: 10},
	{ID: CondFrenzied, Name: "Frenzied", Description: "Mobs use a more aggressive behavior tree with new abilities", MaxRank: 1, ScorePerRank: 10},
	{ID: CondVolatile, Name: "Volatile", Description: "Boss ability patterns are denser and more complex", MaxRank: 1, ScorePerRank: 10},
}

// HPMultiplier returns the enemy HP multiplier from active conditions.
// Each rank of CondEnemyHP adds 20% (rank 1 = 1.2x, rank 5 = 2.0x).
func (s *State) HPMultiplier() float32 {
	if s == nil {
		return 1.0
	}
	for _, c := range s.Conditions {
		if c.ID == CondEnemyHP && c.Rank > 0 {
			return 1.0 + 0.2*float32(c.Rank)
		}
	}
	return 1.0
}

// DamageMultiplier returns the enemy damage multiplier from active conditions.
// Each rank of CondEnemyHP adds 15% (rank 1 = 1.15x, rank 5 = 1.75x).
// Prevents HP-only scaling from making fights easier by diluting DPS pressure.
func (s *State) DamageMultiplier() float32 {
	if s == nil {
		return 1.0
	}
	for _, c := range s.Conditions {
		if c.ID == CondEnemyHP && c.Rank > 0 {
			return 1.0 + 0.15*float32(c.Rank)
		}
	}
	return 1.0
}

// HasCondition returns true if the given condition is active (rank > 0).
func (s *State) HasCondition(id ConditionID) bool {
	if s == nil {
		return false
	}
	for _, c := range s.Conditions {
		if c.ID == id && c.Rank > 0 {
			return true
		}
	}
	return false
}

// ComputeScore calculates total overflux from a set of active conditions.
func ComputeScore(conditions []ActiveCondition) int {
	score := 0
	for _, c := range conditions {
		def := lookupDef(c.ID)
		if def != nil && c.Rank > 0 {
			score += def.ScorePerRank * c.Rank
		}
	}
	return score
}

func lookupDef(id ConditionID) *ConditionDef {
	for i := range Registry {
		if Registry[i].ID == id {
			return &Registry[i]
		}
	}
	return nil
}

// NewState builds a validated State from client-submitted conditions.
// Invalid condition IDs or out-of-range ranks are silently dropped.
func NewState(conditions []ActiveCondition) *State {
	var valid []ActiveCondition
	for _, c := range conditions {
		def := lookupDef(c.ID)
		if def == nil || c.Rank < 1 || c.Rank > def.MaxRank {
			continue
		}
		valid = append(valid, c)
	}
	if len(valid) == 0 {
		return &State{}
	}
	return &State{
		Conditions: valid,
		TotalScore: ComputeScore(valid),
	}
}

// ---------------------------------------------------------------------------
// Wire codec
// ---------------------------------------------------------------------------

// DecodeConditions parses conditions from OpEnterPortal payload.
// Wire format: [count:u8][per: id_len:u8 + id:bytes + rank:u8]
// Returns nil slice (not error) for empty payload (backward compat).
func DecodeConditions(payload []byte) ([]ActiveCondition, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	count := int(payload[0])
	off := 1
	out := make([]ActiveCondition, 0, count)
	for i := range count {
		if off >= len(payload) {
			return nil, fmt.Errorf("overflux: truncated at condition %d", i)
		}
		idLen := int(payload[off])
		off++
		if off+idLen >= len(payload) {
			return nil, fmt.Errorf("overflux: truncated id at condition %d", i)
		}
		id := ConditionID(payload[off : off+idLen])
		off += idLen
		rank := int(payload[off])
		off++
		out = append(out, ActiveCondition{ID: id, Rank: rank})
	}
	return out, nil
}

// EncodeState serializes for OpOverfluxState.
// Wire format: [total_score:u16 LE][count:u8][per: id_len:u8 + id:bytes + rank:u8]
func EncodeState(s *State) []byte {
	if s == nil {
		return []byte{0, 0, 0} // score=0, count=0
	}
	size := 3 // u16 score + u8 count
	for _, c := range s.Conditions {
		size += 1 + len(c.ID) + 1 // id_len + id + rank
	}
	buf := make([]byte, size)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(s.TotalScore))
	buf[2] = byte(len(s.Conditions))
	off := 3
	for _, c := range s.Conditions {
		buf[off] = byte(len(c.ID))
		off++
		copy(buf[off:], c.ID)
		off += len(c.ID)
		buf[off] = byte(c.Rank)
		off++
	}
	return buf
}

// EncodeJoinPrompt serializes for OpInstanceJoinPrompt.
// Wire format: [zone:str8][leader:str8][total_score:u16 LE][count:u8][per: id_len:u8 + id:bytes + rank:u8]
func EncodeJoinPrompt(zoneName, leaderName string, s *State) []byte {
	stateBytes := EncodeState(s)
	size := 1 + len(zoneName) + 1 + len(leaderName) + len(stateBytes)
	buf := make([]byte, size)
	off := 0
	buf[off] = byte(len(zoneName))
	off++
	copy(buf[off:], zoneName)
	off += len(zoneName)
	buf[off] = byte(len(leaderName))
	off++
	copy(buf[off:], leaderName)
	off += len(leaderName)
	copy(buf[off:], stateBytes)
	return buf
}

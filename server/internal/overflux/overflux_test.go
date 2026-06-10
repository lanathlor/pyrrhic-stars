package overflux

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// ---------------------------------------------------------------------------
// HPMultiplier
// ---------------------------------------------------------------------------

func TestHPMultiplier_NilState(t *testing.T) {
	var s *State
	if got := s.HPMultiplier(); got != 1.0 {
		t.Errorf("nil state: want 1.0, got %v", got)
	}
}

func TestHPMultiplier(t *testing.T) {
	tests := []struct {
		name       string
		conditions []ActiveCondition
		want       float32
	}{
		{
			name:       "empty conditions returns 1.0",
			conditions: nil,
			want:       1.0,
		},
		{
			name:       "rank 1 returns 1.2",
			conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 1}},
			want:       1.2,
		},
		{
			name:       "rank 5 returns 2.0",
			conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 5}},
			want:       2.0,
		},
		{
			name:       "rank 3 returns 1.6",
			conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 3}},
			want:       1.6,
		},
		{
			name:       "non-HP condition returns 1.0",
			conditions: []ActiveCondition{{ID: "something_else", Rank: 3}},
			want:       1.0,
		},
		{
			name:       "rank 0 is ignored returns 1.0",
			conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 0}},
			want:       1.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &State{Conditions: tc.conditions}
			if got := s.HPMultiplier(); got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ComputeScore
// ---------------------------------------------------------------------------

func TestComputeScore(t *testing.T) {
	tests := []struct {
		name       string
		conditions []ActiveCondition
		want       int
	}{
		{
			name:       "empty returns 0",
			conditions: nil,
			want:       0,
		},
		{
			name:       "single condition rank 1",
			conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 1}},
			want:       5, // ScorePerRank(5) * rank(1)
		},
		{
			name:       "single condition rank 5",
			conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 5}},
			want:       25, // ScorePerRank(5) * rank(5)
		},
		{
			name: "multiple conditions with same id",
			conditions: []ActiveCondition{
				{ID: CondEnemyHP, Rank: 2},
				{ID: CondEnemyHP, Rank: 3},
			},
			want: 25, // (5*2) + (5*3)
		},
		{
			name:       "unknown condition ID returns 0",
			conditions: []ActiveCondition{{ID: "nonexistent", Rank: 5}},
			want:       0,
		},
		{
			name: "mix of known and unknown",
			conditions: []ActiveCondition{
				{ID: CondEnemyHP, Rank: 2},
				{ID: "nonexistent", Rank: 5},
			},
			want: 10, // only the known condition counts (5*2)
		},
		{
			name:       "rank 0 skipped",
			conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 0}},
			want:       0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ComputeScore(tc.conditions); got != tc.want {
				t.Errorf("want %d, got %d", tc.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewState
// ---------------------------------------------------------------------------

func TestNewState(t *testing.T) {
	t.Run("empty input returns empty state with score 0", func(t *testing.T) {
		s := NewState(nil)
		if s == nil {
			t.Fatal("want non-nil *State")
		}
		if len(s.Conditions) != 0 {
			t.Errorf("want empty conditions, got %v", s.Conditions)
		}
		if s.TotalScore != 0 {
			t.Errorf("want score 0, got %d", s.TotalScore)
		}
	})

	t.Run("valid condition passes through", func(t *testing.T) {
		s := NewState([]ActiveCondition{{ID: CondEnemyHP, Rank: 3}})
		if len(s.Conditions) != 1 {
			t.Fatalf("want 1 condition, got %d", len(s.Conditions))
		}
		if s.Conditions[0].ID != CondEnemyHP || s.Conditions[0].Rank != 3 {
			t.Errorf("unexpected condition: %+v", s.Conditions[0])
		}
		if s.TotalScore != 15 {
			t.Errorf("want score 15, got %d", s.TotalScore)
		}
	})

	t.Run("invalid ID is dropped", func(t *testing.T) {
		s := NewState([]ActiveCondition{{ID: "bogus_id", Rank: 2}})
		if len(s.Conditions) != 0 {
			t.Errorf("want 0 conditions, got %d", len(s.Conditions))
		}
		if s.TotalScore != 0 {
			t.Errorf("want score 0, got %d", s.TotalScore)
		}
	})

	t.Run("rank 0 is dropped", func(t *testing.T) {
		s := NewState([]ActiveCondition{{ID: CondEnemyHP, Rank: 0}})
		if len(s.Conditions) != 0 {
			t.Errorf("want 0 conditions, got %d", len(s.Conditions))
		}
	})

	t.Run("rank above MaxRank is dropped", func(t *testing.T) {
		// CondEnemyHP has MaxRank=5; rank 6 should be rejected.
		s := NewState([]ActiveCondition{{ID: CondEnemyHP, Rank: 6}})
		if len(s.Conditions) != 0 {
			t.Errorf("want 0 conditions for rank 6, got %d", len(s.Conditions))
		}
	})

	t.Run("mixed valid and invalid drops only invalid", func(t *testing.T) {
		input := []ActiveCondition{
			{ID: CondEnemyHP, Rank: 2},
			{ID: "bad", Rank: 1},
			{ID: CondEnemyHP, Rank: 0},
			{ID: CondEnemyHP, Rank: 6},
		}
		s := NewState(input)
		if len(s.Conditions) != 1 {
			t.Fatalf("want 1 valid condition, got %d", len(s.Conditions))
		}
		if s.Conditions[0].Rank != 2 {
			t.Errorf("want rank 2, got %d", s.Conditions[0].Rank)
		}
		if s.TotalScore != 10 {
			t.Errorf("want score 10, got %d", s.TotalScore)
		}
	})
}

// ---------------------------------------------------------------------------
// DecodeConditions
// ---------------------------------------------------------------------------

func TestDecodeConditions_EmptyPayload(t *testing.T) {
	out, err := DecodeConditions(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Errorf("want nil slice, got %v", out)
	}

	out, err = DecodeConditions([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Errorf("want nil slice for empty slice, got %v", out)
	}
}

func TestDecodeConditions_RoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		conditions []ActiveCondition
	}{
		{
			name:       "single condition",
			conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 3}},
		},
		{
			name: "multiple conditions",
			conditions: []ActiveCondition{
				{ID: CondEnemyHP, Rank: 1},
				{ID: "other_id", Rank: 2},
			},
		},
		{
			name:       "count zero",
			conditions: []ActiveCondition{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Manually encode in wire format: [count:u8][id_len:u8 + id:bytes + rank:u8]
			payload := encodeConditionsForTest(tc.conditions)
			got, err := DecodeConditions(payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.conditions) {
				t.Fatalf("want %d conditions, got %d", len(tc.conditions), len(got))
			}
			for i, want := range tc.conditions {
				if got[i].ID != want.ID || got[i].Rank != want.Rank {
					t.Errorf("condition[%d]: want %+v, got %+v", i, want, got[i])
				}
			}
		})
	}
}

// encodeConditionsForTest produces the wire format consumed by DecodeConditions.
func encodeConditionsForTest(conditions []ActiveCondition) []byte {
	if len(conditions) == 0 {
		// count=0 payload
		return []byte{0}
	}
	var buf bytes.Buffer
	buf.WriteByte(byte(len(conditions)))
	for _, c := range conditions {
		buf.WriteByte(byte(len(c.ID)))
		buf.WriteString(string(c.ID))
		buf.WriteByte(byte(c.Rank))
	}
	return buf.Bytes()
}

func TestDecodeConditions_Errors(t *testing.T) {
	t.Run("truncated at condition start", func(t *testing.T) {
		// count=1 but no bytes follow
		_, err := DecodeConditions([]byte{1})
		if err == nil {
			t.Error("want error for truncated payload, got nil")
		}
	})

	t.Run("truncated ID (no rank byte)", func(t *testing.T) {
		// count=1, id_len=3, id="abc" but no rank byte
		// off+idLen >= len(payload) triggers truncated id error when rank is missing
		payload := []byte{1, 3, 'a', 'b', 'c'} // len=5; off=1, idLen=3, check: 2+3>=5 true
		_, err := DecodeConditions(payload)
		if err == nil {
			t.Error("want error for truncated id (missing rank), got nil")
		}
	})

	t.Run("truncated ID bytes", func(t *testing.T) {
		// count=1, id_len=8 but only 2 bytes of id follow (no rank)
		payload := []byte{1, 8, 'a', 'b'}
		_, err := DecodeConditions(payload)
		if err == nil {
			t.Error("want error for truncated id bytes, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// EncodeState
// ---------------------------------------------------------------------------

func TestEncodeState_NilState(t *testing.T) {
	b := EncodeState(nil)
	// expect [0x00, 0x00, 0x00] — score=0 LE u16 + count=0
	want := []byte{0, 0, 0}
	if !bytes.Equal(b, want) {
		t.Errorf("nil state: want %v, got %v", want, b)
	}
}

func TestEncodeState_EmptyConditions(t *testing.T) {
	s := &State{}
	b := EncodeState(s)
	want := []byte{0, 0, 0}
	if !bytes.Equal(b, want) {
		t.Errorf("empty state: want %v, got %v", want, b)
	}
}

func TestEncodeState_ByteFormat(t *testing.T) {
	s := &State{
		Conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 2}},
		TotalScore: 8,
	}
	b := EncodeState(s)

	// Verify total_score u16 LE
	score := binary.LittleEndian.Uint16(b[0:2])
	if score != 8 {
		t.Errorf("score: want 8, got %d", score)
	}

	// Verify count byte
	if b[2] != 1 {
		t.Errorf("count: want 1, got %d", b[2])
	}

	// Verify id_len
	idLen := int(b[3])
	if idLen != len(CondEnemyHP) {
		t.Errorf("id_len: want %d, got %d", len(CondEnemyHP), idLen)
	}

	// Verify id bytes
	id := string(b[4 : 4+idLen])
	if id != string(CondEnemyHP) {
		t.Errorf("id: want %q, got %q", CondEnemyHP, id)
	}

	// Verify rank byte
	rank := b[4+idLen]
	if rank != 2 {
		t.Errorf("rank: want 2, got %d", rank)
	}

	// Verify total length: 2 (score) + 1 (count) + 1 (id_len) + len(id) + 1 (rank)
	wantLen := 3 + 1 + len(CondEnemyHP) + 1
	if len(b) != wantLen {
		t.Errorf("len: want %d, got %d", wantLen, len(b))
	}
}

func TestEncodeState_RoundTrip(t *testing.T) {
	original := []ActiveCondition{{ID: CondEnemyHP, Rank: 4}}
	s := NewState(original)

	encoded := EncodeState(s)

	// Skip the 3-byte header (score + count) and decode the conditions portion.
	// Re-use DecodeConditions on the conditions-only slice by prepending a count byte.
	condPayload := make([]byte, 1+len(encoded)-3)
	condPayload[0] = encoded[2] // count
	copy(condPayload[1:], encoded[3:])

	decoded, err := DecodeConditions(condPayload)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if len(decoded) != len(s.Conditions) {
		t.Fatalf("want %d conditions, got %d", len(s.Conditions), len(decoded))
	}
	for i, want := range s.Conditions {
		if decoded[i].ID != want.ID || decoded[i].Rank != want.Rank {
			t.Errorf("condition[%d]: want %+v, got %+v", i, want, decoded[i])
		}
	}
}

// ---------------------------------------------------------------------------
// EncodeJoinPrompt
// ---------------------------------------------------------------------------

func TestEncodeJoinPrompt(t *testing.T) {
	zoneName := "arena-1"
	leaderName := "Alice"
	s := &State{
		Conditions: []ActiveCondition{{ID: CondEnemyHP, Rank: 2}},
		TotalScore: 8,
	}

	b := EncodeJoinPrompt(zoneName, leaderName, s)

	off := 0

	// zone str8
	if int(b[off]) != len(zoneName) {
		t.Fatalf("zone len: want %d, got %d", len(zoneName), b[off])
	}
	off++
	if string(b[off:off+len(zoneName)]) != zoneName {
		t.Errorf("zone: want %q, got %q", zoneName, string(b[off:off+len(zoneName)]))
	}
	off += len(zoneName)

	// leader str8
	if int(b[off]) != len(leaderName) {
		t.Fatalf("leader len: want %d, got %d", len(leaderName), b[off])
	}
	off++
	if string(b[off:off+len(leaderName)]) != leaderName {
		t.Errorf("leader: want %q, got %q", leaderName, string(b[off:off+len(leaderName)]))
	}
	off += len(leaderName)

	// remaining bytes should match EncodeState output exactly
	stateBytes := EncodeState(s)
	if !bytes.Equal(b[off:], stateBytes) {
		t.Errorf("state bytes: want %v, got %v", stateBytes, b[off:])
	}

	// total length check
	wantLen := 1 + len(zoneName) + 1 + len(leaderName) + len(stateBytes)
	if len(b) != wantLen {
		t.Errorf("total len: want %d, got %d", wantLen, len(b))
	}
}

func TestEncodeJoinPrompt_NilState(t *testing.T) {
	b := EncodeJoinPrompt("z", "l", nil)

	off := 0
	off += 1 + int(b[0])   // skip zone str8
	off += 1 + int(b[off]) // skip leader str8

	// remainder should be the nil EncodeState output
	stateBytes := EncodeState(nil)
	if !bytes.Equal(b[off:], stateBytes) {
		t.Errorf("nil state bytes: want %v, got %v", stateBytes, b[off:])
	}
}

// ---------------------------------------------------------------------------
// MaxScore
// ---------------------------------------------------------------------------

func TestMaxScore(t *testing.T) {
	// Registry: EnemyHP(5*5) + Tempered(20*1) + Frenzied(10*1) + Volatile(20*1) + WoundedPrey(10*5)
	// = 25 + 20 + 10 + 20 + 50 = 125
	want := 125
	if got := MaxScore(); got != want {
		t.Errorf("MaxScore() = %d, want %d", got, want)
	}
}

// ---------------------------------------------------------------------------
// DamageMultiplier
// ---------------------------------------------------------------------------

func TestDamageMultiplier_NilState(t *testing.T) {
	var s *State
	if got := s.DamageMultiplier(); got != 1.0 {
		t.Errorf("nil state: want 1.0, got %v", got)
	}
}

func TestDamageMultiplier(t *testing.T) {
	tests := []struct {
		name       string
		conditions []ActiveCondition
		want       float32
	}{
		{"empty", nil, 1.0},
		{"rank 1", []ActiveCondition{{ID: CondEnemyHP, Rank: 1}}, 1.15},
		{"rank 3", []ActiveCondition{{ID: CondEnemyHP, Rank: 3}}, 1.45},
		{"rank 5", []ActiveCondition{{ID: CondEnemyHP, Rank: 5}}, 1.75},
		{"non-HP condition", []ActiveCondition{{ID: CondTempered, Rank: 1}}, 1.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &State{Conditions: tc.conditions}
			got := s.DamageMultiplier()
			if got < tc.want-0.001 || got > tc.want+0.001 {
				t.Errorf("DamageMultiplier() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HasCondition
// ---------------------------------------------------------------------------

func TestHasCondition(t *testing.T) {
	tests := []struct {
		name  string
		state *State
		query ConditionID
		want  bool
	}{
		{"nil state", nil, CondTempered, false},
		{"empty conditions", &State{}, CondTempered, false},
		{"condition present", &State{Conditions: []ActiveCondition{{ID: CondTempered, Rank: 1}}}, CondTempered, true},
		{"different condition", &State{Conditions: []ActiveCondition{{ID: CondVolatile, Rank: 1}}}, CondTempered, false},
		{"rank 0 treated as inactive", &State{Conditions: []ActiveCondition{{ID: CondTempered, Rank: 0}}}, CondTempered, false},
		{"multiple conditions", &State{Conditions: []ActiveCondition{{ID: CondTempered, Rank: 1}, {ID: CondVolatile, Rank: 1}}}, CondVolatile, true},
		{"frenzied present", &State{Conditions: []ActiveCondition{{ID: CondFrenzied, Rank: 1}}}, CondFrenzied, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.state.HasCondition(tc.query); got != tc.want {
				t.Errorf("HasCondition(%q) = %v, want %v", tc.query, got, tc.want)
			}
		})
	}
}

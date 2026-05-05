package combatlog

// ReplayRecorder captures binary WorldState frames during an encounter for
// later playback. Each frame is the raw output of codec.AppendEncodeWorldState,
// which the client can decode with NetSerializer.decode_world_state().
type ReplayRecorder struct {
	frames [][]byte
}

// NewReplayRecorder creates a recorder pre-allocated for ~5 minutes at 20fps.
func NewReplayRecorder() *ReplayRecorder {
	return &ReplayRecorder{
		frames: make([][]byte, 0, 6000),
	}
}

// AppendFrame stores a copy of the encoded WorldState binary for this tick.
// The caller may reuse the source buffer after this call.
func (r *ReplayRecorder) AppendFrame(encoded []byte) {
	frame := make([]byte, len(encoded))
	copy(frame, encoded)
	r.frames = append(r.frames, frame)
}

// Frames returns the accumulated frame data.
func (r *ReplayRecorder) Frames() [][]byte {
	return r.frames
}

// FrameCount returns the number of recorded frames.
func (r *ReplayRecorder) FrameCount() int {
	return len(r.frames)
}

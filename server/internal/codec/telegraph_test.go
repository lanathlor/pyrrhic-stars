package codec

import (
	"encoding/binary"
	"math"
	"testing"
)

// telegraphReader decodes the wire format produced by AppendTelegraphs.
type telegraphReader struct {
	b   []byte
	off int
}

func (r *telegraphReader) u8() uint8 { v := r.b[r.off]; r.off++; return v }
func (r *telegraphReader) u32() uint32 {
	v := binary.LittleEndian.Uint32(r.b[r.off:])
	r.off += 4
	return v
}
func (r *telegraphReader) f32() float32 { return math.Float32frombits(r.u32()) }

func (r *telegraphReader) desc() TelegraphDesc {
	t := TelegraphDesc{ID: r.u32(), Shape: r.u8(), Category: r.u8(), StartTick: r.u32(), ExecuteTick: r.u32()}
	switch t.Shape {
	case TelegraphShapeCircle:
		t.CX, t.CZ, t.Radius = r.f32(), r.f32(), r.f32()
	case TelegraphShapeCone:
		t.CX, t.CZ, t.Facing, t.HalfAngle, t.Range = r.f32(), r.f32(), r.f32(), r.f32(), r.f32()
	case TelegraphShapeLine:
		t.CX, t.CZ, t.DirX, t.DirZ, t.Length, t.Width = r.f32(), r.f32(), r.f32(), r.f32(), r.f32(), r.f32()
	case TelegraphShapeMulti:
		t.Radius = r.f32()
		n := int(r.u8())
		for range n {
			t.Centers = append(t.Centers, [2]float32{r.f32(), r.f32()})
		}
	}
	return t
}

func TestAppendTelegraphs_RoundTrip(t *testing.T) {
	in := []TelegraphDesc{
		{ID: 1000, Shape: TelegraphShapeCircle, Category: TelegraphCatUnavoidable, StartTick: 10, ExecuteTick: 30, CX: 1.5, CZ: -2.5, Radius: 6.5},
		{ID: 1001, Shape: TelegraphShapeCone, Category: TelegraphCatParryable, StartTick: 5, ExecuteTick: 25, CX: 0, CZ: 0, Facing: 1.2, HalfAngle: 1.57, Range: 3},
		{ID: 1002, Shape: TelegraphShapeLine, Category: TelegraphCatUnavoidable, StartTick: 0, ExecuteTick: 20, CX: 2, CZ: 2, DirX: 1, DirZ: 0, Length: 15, Width: 4},
		{ID: 1003, Shape: TelegraphShapeMulti, Category: TelegraphCatUnavoidable, StartTick: 7, ExecuteTick: 27, Radius: 9, Centers: [][2]float32{{-8, -6}, {8, -6}, {0, 10}}},
	}

	buf := AppendTelegraphs(nil, in)

	r := &telegraphReader{b: buf}
	count := int(r.u8())
	if count != len(in) {
		t.Fatalf("count = %d, want %d", count, len(in))
	}
	for i := range in {
		got := r.desc()
		want := in[i]
		if got.ID != want.ID || got.Shape != want.Shape || got.Category != want.Category ||
			got.StartTick != want.StartTick || got.ExecuteTick != want.ExecuteTick {
			t.Errorf("telegraph %d header mismatch: got %+v want %+v", i, got, want)
		}
		switch want.Shape {
		case TelegraphShapeCircle:
			if got.CX != want.CX || got.CZ != want.CZ || got.Radius != want.Radius {
				t.Errorf("circle %d geometry mismatch: got %+v", i, got)
			}
		case TelegraphShapeCone:
			if got.Facing != want.Facing || got.HalfAngle != want.HalfAngle || got.Range != want.Range {
				t.Errorf("cone %d geometry mismatch: got %+v", i, got)
			}
		case TelegraphShapeLine:
			if got.DirX != want.DirX || got.Length != want.Length || got.Width != want.Width {
				t.Errorf("line %d geometry mismatch: got %+v", i, got)
			}
		case TelegraphShapeMulti:
			if got.Radius != want.Radius || len(got.Centers) != len(want.Centers) {
				t.Fatalf("multi %d geometry mismatch: got %+v", i, got)
			}
			for j := range want.Centers {
				if got.Centers[j] != want.Centers[j] {
					t.Errorf("multi %d center %d = %v, want %v", i, j, got.Centers[j], want.Centers[j])
				}
			}
		}
	}
	// Every byte consumed.
	if r.off != len(buf) {
		t.Errorf("consumed %d of %d bytes", r.off, len(buf))
	}
}

func TestAppendTelegraphs_Empty(t *testing.T) {
	buf := AppendTelegraphs(nil, nil)
	if len(buf) != 1 || buf[0] != 0 {
		t.Fatalf("empty telegraphs should encode as a single 0 byte, got %v", buf)
	}
}

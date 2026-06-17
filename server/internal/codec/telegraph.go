package codec

// Telegraph wire encoding. The server is the source of truth for telegraphs
// (the ground danger/heal indicators). Each tick it emits a stateless array of
// active telegraph descriptors, appended to the world-state packet after the
// NPC array. The client is a dumb renderer: it draws exactly what is sent and
// derives the fill animation from StartTick/ExecuteTick against the packet tick.

// Telegraph shapes.
const (
	TelegraphShapeCircle uint8 = 0 // ring centered at (CX,CZ), radius Radius
	TelegraphShapeCone   uint8 = 1 // wedge from apex (CX,CZ), Facing, HalfAngle, Range
	TelegraphShapeLine   uint8 = 2 // rectangle from (CX,CZ) along (DirX,DirZ), Length x Width
	TelegraphShapeMulti  uint8 = 3 // N rings of Radius at each Centers entry
)

// Telegraph categories (drive client color/semantics).
const (
	TelegraphCatUnavoidable uint8 = 0 // must move out of the zone
	TelegraphCatParryable   uint8 = 1 // can be parried
	TelegraphCatBlockable   uint8 = 2 // can be blocked
	TelegraphCatHeal        uint8 = 3 // beneficial (heal zone)
)

// TelegraphDesc is one telegraph descriptor for wire encoding. Geometry fields
// are interpreted by Shape; all positions are on the XZ plane (floor y).
type TelegraphDesc struct {
	ID          uint32 // stable across ticks (one telegraph per enemy)
	Shape       uint8
	Category    uint8
	StartTick   uint32 // absolute tick where fill begins (progress 0)
	ExecuteTick uint32 // absolute tick where the ability lands (progress 1)

	CX, CZ float32 // circle center / cone apex / line start
	Radius float32 // circle / multi ring radius

	Facing    float32 // cone facing angle (radians)
	HalfAngle float32 // cone half-angle (radians)
	Range     float32 // cone range

	DirX, DirZ float32 // line direction (unit, XZ)
	Length     float32 // line length
	Width      float32 // line width

	Centers [][2]float32 // multi ring centers
}

// AppendTelegraphs appends [count:u8] followed by each descriptor to buf.
func AppendTelegraphs(buf []byte, tgs []TelegraphDesc) []byte {
	if len(tgs) > 255 {
		tgs = tgs[:255]
	}
	buf = append(buf, byte(len(tgs)))
	for i := range tgs {
		t := &tgs[i]
		buf = appendU32(buf, t.ID)
		buf = append(buf, t.Shape, t.Category)
		buf = appendU32(buf, t.StartTick)
		buf = appendU32(buf, t.ExecuteTick)
		switch t.Shape {
		case TelegraphShapeCircle:
			buf = appendF32(buf, t.CX)
			buf = appendF32(buf, t.CZ)
			buf = appendF32(buf, t.Radius)
		case TelegraphShapeCone:
			buf = appendF32(buf, t.CX)
			buf = appendF32(buf, t.CZ)
			buf = appendF32(buf, t.Facing)
			buf = appendF32(buf, t.HalfAngle)
			buf = appendF32(buf, t.Range)
		case TelegraphShapeLine:
			buf = appendF32(buf, t.CX)
			buf = appendF32(buf, t.CZ)
			buf = appendF32(buf, t.DirX)
			buf = appendF32(buf, t.DirZ)
			buf = appendF32(buf, t.Length)
			buf = appendF32(buf, t.Width)
		case TelegraphShapeMulti:
			buf = appendF32(buf, t.Radius)
			centers := t.Centers
			if len(centers) > 255 {
				centers = centers[:255]
			}
			buf = append(buf, byte(len(centers)))
			for _, c := range centers {
				buf = appendF32(buf, c[0])
				buf = appendF32(buf, c[1])
			}
		}
	}
	return buf
}

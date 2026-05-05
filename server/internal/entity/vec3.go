package entity

import "math"

// Vec3 is a minimal 3D vector for server-side game math.
type Vec3 struct {
	X, Y, Z float32
}

func (v Vec3) Add(o Vec3) Vec3      { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3      { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Scale(s float32) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3) Neg() Vec3            { return Vec3{-v.X, -v.Y, -v.Z} }
func (v Vec3) Dot(o Vec3) float32   { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }

func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		v.Y*o.Z - v.Z*o.Y,
		v.Z*o.X - v.X*o.Z,
		v.X*o.Y - v.Y*o.X,
	}
}

func (v Vec3) LengthSq() float32 { return v.Dot(v) }

func (v Vec3) Length() float32 { return float32(math.Sqrt(float64(v.LengthSq()))) }

func (v Vec3) Normalized() Vec3 {
	l := v.Length()
	if l < 1e-7 {
		return Vec3{}
	}
	return v.Scale(1.0 / l)
}

func (v Vec3) DistanceTo(o Vec3) float32 { return v.Sub(o).Length() }

func (v Vec3) DistanceToSq(o Vec3) float32 { return v.Sub(o).LengthSq() }

// Flat zeros out the Y component (for horizontal-plane calculations).
func (v Vec3) Flat() Vec3 { return Vec3{v.X, 0, v.Z} }

// Lerp linearly interpolates between v and o by t.
func (v Vec3) Lerp(o Vec3, t float32) Vec3 {
	return Vec3{
		v.X + (o.X-v.X)*t,
		v.Y + (o.Y-v.Y)*t,
		v.Z + (o.Z-v.Z)*t,
	}
}

// AngleTo returns the unsigned angle in radians between v and o.
func (v Vec3) AngleTo(o Vec3) float32 {
	d := v.Dot(o)
	lv := v.Length()
	lo := o.Length()
	denom := lv * lo
	if denom < 1e-7 {
		return 0
	}
	cosA := d / denom
	// Clamp for numerical safety.
	if cosA > 1.0 {
		cosA = 1.0
	} else if cosA < -1.0 {
		cosA = -1.0
	}
	return float32(math.Acos(float64(cosA)))
}

// MoveToward moves v toward o by at most maxDelta.
func MoveToward(current, target, maxDelta float32) float32 {
	diff := target - current
	if diff > maxDelta {
		return current + maxDelta
	}
	if diff < -maxDelta {
		return current - maxDelta
	}
	return target
}

// Clamp restricts v between lo and hi.
func Clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// LerpAngle interpolates between two angles (radians), handling wrap-around.
func LerpAngle(from, to, t float32) float32 {
	diff := float32(math.Remainder(float64(to-from), 2*math.Pi))
	return from + diff*t
}

// RotateY returns a direction vector rotated by yaw (radians) around the Y axis.
// Input (0,0,-1) at yaw=0 gives forward in Godot convention.
func RotateY(dir Vec3, yaw float32) Vec3 {
	s := float32(math.Sin(float64(yaw)))
	c := float32(math.Cos(float64(yaw)))
	return Vec3{
		dir.X*c + dir.Z*s,
		dir.Y,
		-dir.X*s + dir.Z*c,
	}
}

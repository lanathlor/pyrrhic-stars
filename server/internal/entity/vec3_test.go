package entity

import (
	"math"
	"testing"
)

func TestSub(t *testing.T) {
	a := Vec3{X: 3, Y: 5, Z: 7}
	b := Vec3{X: 1, Y: 2, Z: 3}
	got := a.Sub(b)
	want := Vec3{X: 2, Y: 3, Z: 4}
	if got != want {
		t.Errorf("Sub = %v, want %v", got, want)
	}
}

func TestNeg(t *testing.T) {
	v := Vec3{X: 1, Y: -2, Z: 3}
	got := v.Neg()
	want := Vec3{X: -1, Y: 2, Z: -3}
	if got != want {
		t.Errorf("Neg = %v, want %v", got, want)
	}
}

func TestCross(t *testing.T) {
	x := Vec3{X: 1}
	y := Vec3{Y: 1}
	got := x.Cross(y)
	// X cross Y = Z
	if got.Z < 0.99 || got.Z > 1.01 {
		t.Errorf("Cross(X,Y) = %v, want (0,0,1)", got)
	}

	// Y cross X = -Z
	got2 := y.Cross(x)
	if got2.Z > -0.99 || got2.Z < -1.01 {
		t.Errorf("Cross(Y,X) = %v, want (0,0,-1)", got2)
	}
}

func TestDistanceTo(t *testing.T) {
	a := Vec3{X: 0, Y: 0, Z: 0}
	b := Vec3{X: 3, Y: 4, Z: 0}
	got := a.DistanceTo(b)
	if got < 4.99 || got > 5.01 {
		t.Errorf("DistanceTo = %f, want 5", got)
	}
}

func TestDistanceToSq(t *testing.T) {
	a := Vec3{X: 0, Y: 0, Z: 0}
	b := Vec3{X: 3, Y: 4, Z: 0}
	got := a.DistanceToSq(b)
	if got < 24.99 || got > 25.01 {
		t.Errorf("DistanceToSq = %f, want 25", got)
	}
}

func TestFlat(t *testing.T) {
	v := Vec3{X: 3, Y: 5, Z: 7}
	got := v.Flat()
	want := Vec3{X: 3, Y: 0, Z: 7}
	if got != want {
		t.Errorf("Flat = %v, want %v", got, want)
	}
}

func TestLerp(t *testing.T) {
	a := Vec3{X: 0, Y: 0, Z: 0}
	b := Vec3{X: 10, Y: 20, Z: 30}

	got0 := a.Lerp(b, 0)
	if got0 != a {
		t.Errorf("Lerp(0) = %v, want %v", got0, a)
	}
	got1 := a.Lerp(b, 1)
	if got1 != b {
		t.Errorf("Lerp(1) = %v, want %v", got1, b)
	}
	gotHalf := a.Lerp(b, 0.5)
	want := Vec3{X: 5, Y: 10, Z: 15}
	if gotHalf != want {
		t.Errorf("Lerp(0.5) = %v, want %v", gotHalf, want)
	}
}

func TestAngleTo(t *testing.T) {
	x := Vec3{X: 1}
	y := Vec3{Y: 1}
	got := x.AngleTo(y)
	want := float32(math.Pi / 2)
	if diff := got - want; diff > 0.01 || diff < -0.01 {
		t.Errorf("AngleTo(X,Y) = %f, want %f", got, want)
	}

	// Parallel vectors → angle 0
	got2 := x.AngleTo(x)
	if got2 > 0.01 {
		t.Errorf("AngleTo(X,X) = %f, want 0", got2)
	}

	// Zero vector → 0
	got3 := x.AngleTo(Vec3{})
	if got3 != 0 {
		t.Errorf("AngleTo(X,zero) = %f, want 0", got3)
	}
}

func TestMoveToward(t *testing.T) {
	tests := []struct {
		current, target, delta, want float32
	}{
		{0, 10, 3, 3},   // move toward
		{0, 10, 20, 10}, // overshoot clamps
		{5, 0, 2, 3},    // move backward
		{5, 0, 10, 0},   // overshoot backward
		{5, 5, 1, 5},    // already there
	}
	for _, tc := range tests {
		got := MoveToward(tc.current, tc.target, tc.delta)
		if got != tc.want {
			t.Errorf("MoveToward(%f, %f, %f) = %f, want %f", tc.current, tc.target, tc.delta, got, tc.want)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		v, lo, hi, want float32
	}{
		{5, 0, 10, 5},   // in range
		{-1, 0, 10, 0},  // below
		{15, 0, 10, 10}, // above
		{0, 0, 0, 0},    // equal bounds
	}
	for _, tc := range tests {
		got := Clamp(tc.v, tc.lo, tc.hi)
		if got != tc.want {
			t.Errorf("Clamp(%f, %f, %f) = %f, want %f", tc.v, tc.lo, tc.hi, got, tc.want)
		}
	}
}

func TestLerpAngle(t *testing.T) {
	// t=0 → returns from
	got0 := LerpAngle(1.0, 2.0, 0)
	if got0 != 1.0 {
		t.Errorf("LerpAngle(1, 2, 0) = %f, want 1.0", got0)
	}

	// t=1 → returns to (via shortest path)
	got1 := LerpAngle(0, 1.0, 1.0)
	if diff := got1 - 1.0; diff > 0.01 || diff < -0.01 {
		t.Errorf("LerpAngle(0, 1, 1) = %f, want 1.0", got1)
	}

	// Midpoint of small arc
	got2 := LerpAngle(0, 1.0, 0.5)
	if diff := got2 - 0.5; diff > 0.01 || diff < -0.01 {
		t.Errorf("LerpAngle(0, 1, 0.5) = %f, want 0.5", got2)
	}
}

func TestRotateY(t *testing.T) {
	// Forward (0,0,-1) at yaw=0 should stay (0,0,-1)
	fwd := Vec3{X: 0, Y: 0, Z: -1}
	got := RotateY(fwd, 0)
	if got.Z > -0.99 {
		t.Errorf("RotateY(fwd, 0) = %v, want (0,0,-1)", got)
	}

	// Forward (0,0,-1) at yaw=pi/2 should give (-1,0,0) or close
	got2 := RotateY(fwd, float32(math.Pi/2))
	if got2.X > -0.99 || got2.X < -1.01 {
		t.Errorf("RotateY(fwd, pi/2).X = %f, want -1", got2.X)
	}
	if got2.Z > 0.01 || got2.Z < -0.01 {
		t.Errorf("RotateY(fwd, pi/2).Z = %f, want 0", got2.Z)
	}

	// Y component should be preserved
	up := Vec3{X: 0, Y: 5, Z: -1}
	got3 := RotateY(up, float32(math.Pi))
	if got3.Y != 5 {
		t.Errorf("RotateY preserved Y = %f, want 5", got3.Y)
	}
}

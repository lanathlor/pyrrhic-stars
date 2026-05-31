package level

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestPointInConvexPolyXZ(t *testing.T) {
	// Simple triangle on XZ plane
	tri := []entity.Vec3{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 5, Y: 0, Z: 10},
	}
	tests := []struct {
		name   string
		x, z   float32
		expect bool
	}{
		{"center", 5, 3, true},
		{"near vertex", 1, 1, true},
		{"outside left", -1, 5, false},
		{"outside right", 11, 5, false},
		{"outside top", 5, 11, false},
		{"outside bottom", 5, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pointInConvexPolyXZ(tt.x, tt.z, tri)
			if got != tt.expect {
				t.Errorf("pointInConvexPolyXZ(%v, %v) = %v, want %v", tt.x, tt.z, got, tt.expect)
			}
		})
	}
}

func TestPointInConvexPolyXZ_Quad(t *testing.T) {
	quad := []entity.Vec3{
		{X: -5, Y: 0, Z: -5},
		{X: 5, Y: 0, Z: -5},
		{X: 5, Y: 0, Z: 5},
		{X: -5, Y: 0, Z: 5},
	}
	tests := []struct {
		name   string
		x, z   float32
		expect bool
	}{
		{"center", 0, 0, true},
		{"corner", 4, 4, true},
		{"outside", 6, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pointInConvexPolyXZ(tt.x, tt.z, quad)
			if got != tt.expect {
				t.Errorf("pointInConvexPolyXZ(%v, %v) = %v, want %v", tt.x, tt.z, got, tt.expect)
			}
		})
	}
}

func TestSampleY_FlatFloor(t *testing.T) {
	verts := []entity.Vec3{
		{X: -10, Y: 0, Z: -10},
		{X: 10, Y: 0, Z: -10},
		{X: 10, Y: 0, Z: 10},
		{X: -10, Y: 0, Z: 10},
	}
	nm := buildNavmesh(verts, [][]int{{0, 1, 2, 3}})
	if len(nm.Polys) != 1 {
		t.Fatalf("expected 1 polygon, got %d", len(nm.Polys))
	}

	y, ok := nm.SampleY(0, 0, 0)
	if !ok {
		t.Fatal("SampleY should find floor at origin")
	}
	if math.Abs(float64(y)) > 0.01 {
		t.Errorf("expected Y=0, got %v", y)
	}

	// Off-mesh point
	_, ok = nm.SampleY(15, 0, 0)
	if ok {
		t.Error("SampleY should return false for off-mesh point")
	}
}

func TestSampleY_Ramp(t *testing.T) {
	// A tilted quad: Y=0 at Z=-5, Y=5 at Z=5 (45-degree ramp along Z)
	verts := []entity.Vec3{
		{X: -5, Y: 0, Z: -5},
		{X: 5, Y: 0, Z: -5},
		{X: 5, Y: 5, Z: 5},
		{X: -5, Y: 5, Z: 5},
	}
	nm := buildNavmesh(verts, [][]int{{0, 1, 2, 3}})
	if len(nm.Polys) != 1 {
		t.Fatalf("expected 1 polygon, got %d", len(nm.Polys))
	}

	tests := []struct {
		name      string
		x, z      float32
		expectedY float32
	}{
		{"bottom", 0, -5, 0},
		{"middle", 0, 0, 2.5},
		{"top", 0, 5, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			y, ok := nm.SampleY(tt.x, tt.z, tt.expectedY)
			if !ok {
				t.Fatalf("SampleY should find ramp at (%v, %v)", tt.x, tt.z)
			}
			if math.Abs(float64(y-tt.expectedY)) > 0.1 {
				t.Errorf("expected Y=%v, got %v", tt.expectedY, y)
			}
		})
	}
}

func TestSampleY_TwoFloors(t *testing.T) {
	// Two overlapping quads: ground floor at Y=0, upper floor at Y=5
	verts := []entity.Vec3{
		// Ground floor
		{X: -10, Y: 0, Z: -10},
		{X: 10, Y: 0, Z: -10},
		{X: 10, Y: 0, Z: 10},
		{X: -10, Y: 0, Z: 10},
		// Upper floor
		{X: -10, Y: 5, Z: -10},
		{X: 10, Y: 5, Z: -10},
		{X: 10, Y: 5, Z: 10},
		{X: -10, Y: 5, Z: 10},
	}
	nm := buildNavmesh(verts, [][]int{
		{0, 1, 2, 3},
		{4, 5, 6, 7},
	})
	if len(nm.Polys) != 2 {
		t.Fatalf("expected 2 polygons, got %d", len(nm.Polys))
	}

	// nearY=0 should pick ground floor
	y, ok := nm.SampleY(0, 0, 0)
	if !ok {
		t.Fatal("SampleY should find floor")
	}
	if math.Abs(float64(y)) > 0.01 {
		t.Errorf("nearY=0: expected Y=0, got %v", y)
	}

	// nearY=5 should pick upper floor
	y, ok = nm.SampleY(0, 0, 5)
	if !ok {
		t.Fatal("SampleY should find floor")
	}
	if math.Abs(float64(y-5)) > 0.01 {
		t.Errorf("nearY=5: expected Y=5, got %v", y)
	}

	// nearY=3 should pick nearest (ground=3 away, upper=2 away → upper)
	y, ok = nm.SampleY(0, 0, 3)
	if !ok {
		t.Fatal("SampleY should find floor")
	}
	if math.Abs(float64(y-5)) > 0.01 {
		t.Errorf("nearY=3: expected Y=5 (closer), got %v", y)
	}
}

func TestSampleY_OffMesh(t *testing.T) {
	verts := []entity.Vec3{
		{X: 0, Y: 0, Z: 0},
		{X: 5, Y: 0, Z: 0},
		{X: 5, Y: 0, Z: 5},
	}
	nm := buildNavmesh(verts, [][]int{{0, 1, 2}})

	_, ok := nm.SampleY(10, 10, 0)
	if ok {
		t.Error("SampleY should return false for off-mesh point")
	}
}

func TestBuildNavmesh_DegeneratePolygon(t *testing.T) {
	// Degenerate polygon with collinear vertices
	verts := []entity.Vec3{
		{X: 0, Y: 0, Z: 0},
		{X: 5, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0}, // collinear
	}
	nm := buildNavmesh(verts, [][]int{{0, 1, 2}})
	// Should skip degenerate polygon (zero-area cross product)
	if len(nm.Polys) != 0 {
		t.Errorf("expected 0 polygons for degenerate input, got %d", len(nm.Polys))
	}
}

func TestBuildNavmesh_TooFewVertices(t *testing.T) {
	verts := []entity.Vec3{
		{X: 0, Y: 0, Z: 0},
		{X: 5, Y: 0, Z: 0},
	}
	nm := buildNavmesh(verts, [][]int{{0, 1}})
	if len(nm.Polys) != 0 {
		t.Errorf("expected 0 polygons for 2-vertex polygon, got %d", len(nm.Polys))
	}
}

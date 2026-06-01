package level

import (
	"math"

	"codex-online/server/internal/entity"
)

// NavPoly is a convex walkable polygon from the baked navmesh.
type NavPoly struct {
	Vertices   []entity.Vec3 // polygon vertices in winding order
	Normal     entity.Vec3   // plane normal (typically ~(0,1,0) for floors)
	D          float32       // plane equation: Normal.Dot(point) = D
	MinX, MaxX float32       // AABB for fast rejection
	MinZ, MaxZ float32
}

// gridCell holds indices into Navmesh.Polys for polygons overlapping this cell.
type gridCell struct {
	polyIndices []int
}

const gridCellSize = 4.0 // meters per cell

// Navmesh holds the baked navigation mesh for a zone.
type Navmesh struct {
	Polys []NavPoly

	// Spatial grid for O(1) polygon lookup
	grid        []gridCell
	gridCols    int
	gridRows    int
	gridOriginX float32
	gridOriginZ float32
}

// buildNavmesh constructs a Navmesh from raw vertex and polygon index data.
func buildNavmesh(verts []entity.Vec3, polygons [][]int) *Navmesh {
	nm := &Navmesh{
		Polys: make([]NavPoly, 0, len(polygons)),
	}
	for _, indices := range polygons {
		if len(indices) < 3 {
			continue
		}
		if poly, ok := buildNavPoly(verts, indices); ok {
			nm.Polys = append(nm.Polys, poly)
		}
	}
	nm.buildGrid()
	return nm
}

func buildNavPoly(verts []entity.Vec3, indices []int) (NavPoly, bool) {
	poly := NavPoly{
		Vertices: make([]entity.Vec3, len(indices)),
		MinX:     float32(math.MaxFloat32),
		MaxX:     -float32(math.MaxFloat32),
		MinZ:     float32(math.MaxFloat32),
		MaxZ:     -float32(math.MaxFloat32),
	}
	for i, idx := range indices {
		if idx < 0 || idx >= len(verts) {
			continue
		}
		v := verts[idx]
		poly.Vertices[i] = v
		if v.X < poly.MinX {
			poly.MinX = v.X
		}
		if v.X > poly.MaxX {
			poly.MaxX = v.X
		}
		if v.Z < poly.MinZ {
			poly.MinZ = v.Z
		}
		if v.Z > poly.MaxZ {
			poly.MaxZ = v.Z
		}
	}

	// Compute plane equation from first 3 vertices
	v0 := poly.Vertices[0]
	v1 := poly.Vertices[1]
	v2 := poly.Vertices[2]
	edge1 := v1.Sub(v0)
	edge2 := v2.Sub(v0)
	normal := edge1.Cross(edge2)
	nLen := normal.Length()
	if nLen < 1e-6 {
		return poly, false // degenerate polygon
	}
	poly.Normal = entity.Vec3{X: normal.X / nLen, Y: normal.Y / nLen, Z: normal.Z / nLen}
	poly.D = poly.Normal.Dot(v0)

	// Ensure normal points upward (walkable surface)
	if poly.Normal.Y < 0 {
		poly.Normal = entity.Vec3{X: -poly.Normal.X, Y: -poly.Normal.Y, Z: -poly.Normal.Z}
		poly.D = -poly.D
		for i, j := 0, len(poly.Vertices)-1; i < j; i, j = i+1, j-1 {
			poly.Vertices[i], poly.Vertices[j] = poly.Vertices[j], poly.Vertices[i]
		}
	}
	return poly, true
}

// buildGrid creates a spatial hash grid for fast polygon lookup.
func (nm *Navmesh) buildGrid() {
	if len(nm.Polys) == 0 {
		return
	}

	// Find world bounds from all polygons
	minX := float32(math.MaxFloat32)
	maxX := -float32(math.MaxFloat32)
	minZ := float32(math.MaxFloat32)
	maxZ := -float32(math.MaxFloat32)
	for i := range nm.Polys {
		p := &nm.Polys[i]
		if p.MinX < minX {
			minX = p.MinX
		}
		if p.MaxX > maxX {
			maxX = p.MaxX
		}
		if p.MinZ < minZ {
			minZ = p.MinZ
		}
		if p.MaxZ > maxZ {
			maxZ = p.MaxZ
		}
	}

	nm.gridOriginX = minX
	nm.gridOriginZ = minZ
	nm.gridCols = int(math.Ceil(float64(maxX-minX)/gridCellSize)) + 1
	nm.gridRows = int(math.Ceil(float64(maxZ-minZ)/gridCellSize)) + 1
	nm.grid = make([]gridCell, nm.gridCols*nm.gridRows)

	// Insert each polygon into all cells its AABB overlaps
	for pi := range nm.Polys {
		p := &nm.Polys[pi]
		colMin := int((p.MinX - minX) / gridCellSize)
		colMax := int((p.MaxX - minX) / gridCellSize)
		rowMin := int((p.MinZ - minZ) / gridCellSize)
		rowMax := int((p.MaxZ - minZ) / gridCellSize)
		if colMax >= nm.gridCols {
			colMax = nm.gridCols - 1
		}
		if rowMax >= nm.gridRows {
			rowMax = nm.gridRows - 1
		}
		for r := rowMin; r <= rowMax; r++ {
			for c := colMin; c <= colMax; c++ {
				idx := r*nm.gridCols + c
				nm.grid[idx].polyIndices = append(nm.grid[idx].polyIndices, pi)
			}
		}
	}
}

// maxSampleYDelta is the maximum vertical distance between nearY and a navmesh
// polygon result. Polygons further than this are rejected (e.g. wall tops baked
// into the navmesh by mistake).
const maxSampleYDelta = 4.0

// SampleY finds the polygon under (x, z) closest to nearY and returns its Y.
// Returns (y, true) if found, (0, false) if no polygon contains (x, z).
func (nm *Navmesh) SampleY(x, z, nearY float32) (float32, bool) {
	polys := nm.polysAt(x, z)

	bestY := float32(0)
	bestDist := float32(math.MaxFloat32)
	found := false
	for _, pi := range polys {
		p := &nm.Polys[pi]
		// Fast AABB reject
		if x < p.MinX || x > p.MaxX || z < p.MinZ || z > p.MaxZ {
			continue
		}
		// Point-in-polygon test (XZ projection)
		if !pointInConvexPolyXZ(x, z, p.Vertices) {
			continue
		}
		// Near-vertical polygon: not a walkable floor
		if p.Normal.Y < 0.1 {
			continue
		}
		// Compute Y from plane equation: N.X*x + N.Y*y + N.Z*z = D
		y := (p.D - p.Normal.X*x - p.Normal.Z*z) / p.Normal.Y
		dist := y - nearY
		if dist < 0 {
			dist = -dist
		}
		if dist > maxSampleYDelta {
			continue
		}
		if !found || dist < bestDist {
			bestY = y
			bestDist = dist
			found = true
		}
	}
	return bestY, found
}

// polysAt returns the polygon indices for the grid cell containing (x, z).
// Returns nil if out of bounds.
func (nm *Navmesh) polysAt(x, z float32) []int {
	if nm.grid == nil {
		return nil
	}
	col := int((x - nm.gridOriginX) / gridCellSize)
	row := int((z - nm.gridOriginZ) / gridCellSize)
	if col < 0 || col >= nm.gridCols || row < 0 || row >= nm.gridRows {
		return nil
	}
	return nm.grid[row*nm.gridCols+col].polyIndices
}

// pointInConvexPolyXZ tests if (px, pz) is inside a convex polygon
// projected onto the XZ plane using cross-product sign consistency.
func pointInConvexPolyXZ(px, pz float32, verts []entity.Vec3) bool {
	n := len(verts)
	if n < 3 {
		return false
	}
	var positive, negative bool
	for i := range n {
		j := (i + 1) % n
		ex := verts[j].X - verts[i].X
		ez := verts[j].Z - verts[i].Z
		dx := px - verts[i].X
		dz := pz - verts[i].Z
		cross := ex*dz - ez*dx
		if cross > 0 {
			positive = true
		} else if cross < 0 {
			negative = true
		}
		if positive && negative {
			return false
		}
	}
	return true
}

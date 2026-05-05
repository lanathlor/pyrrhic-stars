package entity

// NPCState represents the NPC behavior state.
type NPCState uint8

const (
	NPCIdle NPCState = iota
	NPCWalk
)

// NPC represents a server-controlled non-player character in the hub.
// NPCs walk between waypoints and occasionally pause to idle.
type NPC struct {
	ID        uint16
	Position  Vec3
	RotationY float32
	State     NPCState

	// Definition name (for client-side visual selection)
	DefName string

	// Movement
	Speed       float32
	Waypoints   []Vec3
	WaypointIdx int // current target waypoint

	// Idle timer: when > 0 the NPC stands still
	IdleTimer    float32
	IdleDuration float32 // how long to idle at each waypoint
}

// NewNPC creates a hub NPC at the first waypoint.
func NewNPC(id uint16, defName string, speed float32, idleDuration float32, waypoints []Vec3) *NPC {
	pos := Vec3{}
	if len(waypoints) > 0 {
		pos = waypoints[0]
	}
	return &NPC{
		ID:           id,
		DefName:      defName,
		Position:     pos,
		State:        NPCIdle,
		Speed:        speed,
		Waypoints:    waypoints,
		WaypointIdx:  0,
		IdleTimer:    idleDuration, // start idle
		IdleDuration: idleDuration,
	}
}

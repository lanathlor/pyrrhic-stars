package session

import (
	"sync"

	"codex-online/server/internal/network"
)

// Session represents a connected player across zone transfers.
// Fields are written by the owning connection goroutine and read by others
// (e.g., ResolveZonePeer, PersistFlushTargets). Use Mu to synchronize access
// to mutable fields (ZoneID, ZoneType, PeerID, Class, Spec, CharID, CharName).
type Session struct {
	Mu       sync.RWMutex `json:"-"` // protects mutable fields below
	ID       uint32       // permanent global ID (assigned once at connect)
	UserUUID string       // persistent user account identity
	Username string
	Conn     *network.Client
	ZoneID   string // current zone
	ZoneType uint8  // current zone type (0=OpenWorld, 1=Instanced)
	PeerID   uint16 // current zone peer ID
	Class    string // selected class
	Spec     string // selected spec within class
	CharID   uint   // selected character ID (persistence primary key)
	CharName string // character display name (shown overhead)
}

// PersistFlushTarget holds the fields needed to persist an open-world player's position.
type PersistFlushTarget struct {
	UserUUID string
	CharID   uint
	PeerID   uint16
}

package session

import "codex-online/server/internal/network"

// Session represents a connected player across zone transfers.
type Session struct {
	ID       uint32 // permanent global ID (assigned once at connect)
	UserUUID string // persistent user account identity
	Username string
	Conn     *network.Client
	ZoneID   string // current zone
	PeerID   uint16 // current zone peer ID
	Class    string // selected class
	Spec     string // selected spec within class
	CharID   uint   // selected character ID (persistence primary key)
	CharName string // character display name (shown overhead)
}

// HubFlushTarget holds the fields needed to save a hub player's position.
type HubFlushTarget struct {
	UserUUID string
	CharID   uint
	PeerID   uint16
}

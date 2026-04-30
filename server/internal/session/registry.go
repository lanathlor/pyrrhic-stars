package session

import (
	"sync"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/network"
	"codex-online/server/internal/zone"
)

// Registry manages active player sessions with thread-safe access.
type Registry struct {
	sessions map[uint32]*Session
	connMap  map[*network.Client]uint32
	nextID   uint32
	mu       sync.Mutex
}

// NewRegistry creates an empty session registry.
func NewRegistry() *Registry {
	return &Registry{
		sessions: make(map[uint32]*Session),
		connMap:  make(map[*network.Client]uint32),
		nextID:   1,
	}
}

// Register creates a new session for the given client connection.
func (r *Registry) Register(client *network.Client) *Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextID
	r.nextID++
	sess := &Session{
		ID:    id,
		Conn:  client,
		Class: entity.ClassGunner,
	}
	r.sessions[id] = sess
	r.connMap[client] = id
	return sess
}

// Get returns the session for the given client, or nil.
func (r *Registry) Get(client *network.Client) *Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.connMap[client]
	if !ok {
		return nil
	}
	return r.sessions[id]
}

// GetByID returns the session for the given global player ID, or nil.
func (r *Registry) GetByID(playerID uint32) *Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[playerID]
}

// Remove cleans up a disconnected player's session and returns it.
func (r *Registry) Remove(client *network.Client) *Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.connMap[client]
	if !ok {
		return nil
	}
	sess := r.sessions[id]
	delete(r.connMap, client)
	delete(r.sessions, id)
	return sess
}

// ResolveZonePeer finds the global player ID for a zone peer ID.
// Returns 0 if not found.
func (r *Registry) ResolveZonePeer(zoneID string, peerID uint16) uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sess := range r.sessions {
		if sess.ZoneID == zoneID && sess.PeerID == peerID {
			return sess.ID
		}
	}
	return 0
}

// HubFlushTargets returns a snapshot of all sessions currently in the hub
// zone that have a persisted character, suitable for position flushing.
func (r *Registry) HubFlushTargets() []HubFlushTarget {
	r.mu.Lock()
	defer r.mu.Unlock()
	var targets []HubFlushTarget
	for _, sess := range r.sessions {
		if sess.PlayerUUID != "" && sess.ZoneID == zone.ZoneHub && sess.CharID != 0 {
			targets = append(targets, HubFlushTarget{
				PlayerUUID: sess.PlayerUUID,
				CharID:     sess.CharID,
				PeerID:     sess.PeerID,
			})
		}
	}
	return targets
}

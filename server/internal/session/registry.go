package session

import (
	"sync"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/network"
)

// zonePeerKey is a composite key for the zone peer index.
type zonePeerKey struct {
	ZoneID string
	PeerID uint16
}

// Registry manages active player sessions with thread-safe access.
type Registry struct {
	sessions  map[uint32]*Session
	connMap   map[*network.Client]uint32
	zonePeers map[zonePeerKey]uint32 // (zoneID, peerID) -> global session ID
	nextID    uint32
	mu        sync.Mutex
}

// NewRegistry creates an empty session registry.
func NewRegistry() *Registry {
	return &Registry{
		sessions:  make(map[uint32]*Session),
		connMap:   make(map[*network.Client]uint32),
		zonePeers: make(map[zonePeerKey]uint32),
		nextID:    1,
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
	if sess.ZoneID != "" {
		delete(r.zonePeers, zonePeerKey{sess.ZoneID, sess.PeerID})
	}
	return sess
}

// IndexZonePeer updates the reverse lookup index for fast ResolveZonePeer.
// Call this after writing sess.ZoneID/PeerID to keep the index in sync.
func (r *Registry) IndexZonePeer(sessID uint32, zoneID string, peerID uint16) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Remove any old entry for this session by scanning (rare, only on zone transfer)
	for k, v := range r.zonePeers {
		if v == sessID {
			delete(r.zonePeers, k)
			break
		}
	}
	if zoneID != "" {
		r.zonePeers[zonePeerKey{zoneID, peerID}] = sessID
	}
}

// ResolveZonePeer finds the global player ID for a zone peer ID.
// Returns 0 if not found.
func (r *Registry) ResolveZonePeer(zoneID string, peerID uint16) uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.zonePeers[zonePeerKey{zoneID, peerID}]
}

// PersistFlushTargets returns a snapshot of all sessions currently in an
// open-world zone that have a persisted character, suitable for position flushing.
func (r *Registry) PersistFlushTargets() []PersistFlushTarget {
	r.mu.Lock()
	defer r.mu.Unlock()
	var targets []PersistFlushTarget
	for _, sess := range r.sessions {
		sess.Mu.RLock()
		match := sess.UserUUID != "" && sess.ZoneType == 0 && sess.ZoneID != "" && sess.CharID != 0
		if match {
			targets = append(targets, PersistFlushTarget{
				UserUUID: sess.UserUUID,
				CharID:   sess.CharID,
				PeerID:   sess.PeerID,
			})
		}
		sess.Mu.RUnlock()
	}
	return targets
}

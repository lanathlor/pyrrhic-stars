package group

import (
	"errors"
	"sync"
	"time"
)

const MaxGroupSize = 5

var (
	ErrAlreadyInGroup = errors.New("player already in a group")
	ErrGroupFull      = errors.New("group is full")
	ErrNotLeader      = errors.New("only the leader can do this")
	ErrNotInGroup     = errors.New("player is not in a group")
	ErrInviteSelf     = errors.New("cannot invite yourself")
	ErrInvitePending  = errors.New("player already has a pending invite")
	ErrInviteExpired  = errors.New("invite expired or not found")
	ErrGroupNotFound  = errors.New("group not found")
)

// Group represents a player party.
type Group struct {
	ID       uint32
	LeaderID uint32   // global player ID
	Members  []uint32 // global player IDs (includes leader)
}

// PendingInvite tracks an outstanding group invitation.
type PendingInvite struct {
	GroupID   uint32
	InviterID uint32
	InviteeID uint32
	ExpiresAt time.Time
}

// Manager coordinates group creation, invites, and membership.
type Manager struct {
	groups      map[uint32]*Group
	playerGroup map[uint32]uint32         // playerID → groupID
	invites     map[uint32]*PendingInvite // inviteeID → pending invite
	nextID      uint32
	mu          sync.Mutex
}

// NewManager creates a group manager.
func NewManager() *Manager {
	return &Manager{
		groups:      make(map[uint32]*Group),
		playerGroup: make(map[uint32]uint32),
		invites:     make(map[uint32]*PendingInvite),
		nextID:      1,
	}
}

// CreateGroup creates a new group with the given player as leader.
func (m *Manager) CreateGroup(leaderID uint32) (*Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.playerGroup[leaderID]; ok {
		return nil, ErrAlreadyInGroup
	}
	g := &Group{
		ID:       m.nextID,
		LeaderID: leaderID,
		Members:  []uint32{leaderID},
	}
	m.nextID++
	m.groups[g.ID] = g
	m.playerGroup[leaderID] = g.ID
	return g, nil
}

// InvitePlayer creates a pending invite from the group leader to a target player.
func (m *Manager) InvitePlayer(inviterID, inviteeID uint32) (*PendingInvite, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inviterID == inviteeID {
		return nil, ErrInviteSelf
	}
	groupID, ok := m.playerGroup[inviterID]
	if !ok {
		return nil, ErrNotInGroup
	}
	g := m.groups[groupID]
	if g.LeaderID != inviterID {
		return nil, ErrNotLeader
	}
	if len(g.Members) >= MaxGroupSize {
		return nil, ErrGroupFull
	}
	if _, ok := m.playerGroup[inviteeID]; ok {
		return nil, ErrAlreadyInGroup
	}
	// Clean expired invite
	if existing, ok := m.invites[inviteeID]; ok {
		if time.Now().After(existing.ExpiresAt) {
			delete(m.invites, inviteeID)
		} else {
			return nil, ErrInvitePending
		}
	}
	invite := &PendingInvite{
		GroupID:   groupID,
		InviterID: inviterID,
		InviteeID: inviteeID,
		ExpiresAt: time.Now().Add(30 * time.Second),
	}
	m.invites[inviteeID] = invite
	return invite, nil
}

// AcceptInvite accepts a pending invite and adds the player to the group.
func (m *Manager) AcceptInvite(inviteeID, groupID uint32) (*Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	invite, ok := m.invites[inviteeID]
	if !ok || invite.GroupID != groupID {
		return nil, ErrInviteExpired
	}
	if time.Now().After(invite.ExpiresAt) {
		delete(m.invites, inviteeID)
		return nil, ErrInviteExpired
	}
	delete(m.invites, inviteeID)

	g, ok := m.groups[groupID]
	if !ok {
		return nil, ErrGroupNotFound
	}
	if len(g.Members) >= MaxGroupSize {
		return nil, ErrGroupFull
	}
	if _, ok := m.playerGroup[inviteeID]; ok {
		return nil, ErrAlreadyInGroup
	}
	g.Members = append(g.Members, inviteeID)
	m.playerGroup[inviteeID] = g.ID
	return g, nil
}

// DeclineInvite removes a pending invite.
func (m *Manager) DeclineInvite(inviteeID, groupID uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if invite, ok := m.invites[inviteeID]; ok && invite.GroupID == groupID {
		delete(m.invites, inviteeID)
	}
}

// LeaveGroup removes a player from their group. Returns the updated group
// (nil if disbanded) and whether the group was disbanded.
func (m *Manager) LeaveGroup(playerID uint32) (*Group, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	groupID, ok := m.playerGroup[playerID]
	if !ok {
		return nil, false
	}
	g := m.groups[groupID]
	delete(m.playerGroup, playerID)

	// Remove player from members
	for i, id := range g.Members {
		if id == playerID {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			break
		}
	}

	// If group is now empty, disband
	if len(g.Members) == 0 {
		delete(m.groups, groupID)
		return nil, true
	}

	// If leader left, promote next member
	if g.LeaderID == playerID {
		g.LeaderID = g.Members[0]
	}

	return g, false
}

// KickPlayer removes a player from the group (leader only).
func (m *Manager) KickPlayer(leaderID, targetID uint32) (*Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	groupID, ok := m.playerGroup[leaderID]
	if !ok {
		return nil, ErrNotInGroup
	}
	g := m.groups[groupID]
	if g.LeaderID != leaderID {
		return nil, ErrNotLeader
	}
	if leaderID == targetID {
		return nil, errors.New("cannot kick yourself, use leave")
	}
	targetGroupID, ok := m.playerGroup[targetID]
	if !ok || targetGroupID != groupID {
		return nil, errors.New("target is not in your group")
	}
	delete(m.playerGroup, targetID)
	for i, id := range g.Members {
		if id == targetID {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			break
		}
	}
	return g, nil
}

// GetGroup returns the group for a player, or nil if not in a group.
func (m *Manager) GetGroup(playerID uint32) *Group {
	m.mu.Lock()
	defer m.mu.Unlock()
	groupID, ok := m.playerGroup[playerID]
	if !ok {
		return nil
	}
	return m.groups[groupID]
}

// GetGroupMembers returns the member IDs of a player's group.
func (m *Manager) GetGroupMembers(playerID uint32) []uint32 {
	g := m.GetGroup(playerID)
	if g == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	members := make([]uint32, len(g.Members))
	copy(members, g.Members)
	return members
}

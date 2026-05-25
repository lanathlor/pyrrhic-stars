package group

import (
	"errors"
	"testing"
)

func TestCreateGroup(t *testing.T) {
	m := NewManager()
	g, err := m.CreateGroup(1)
	if err != nil {
		t.Fatal(err)
	}
	if g.ID == 0 {
		t.Error("expected non-zero group ID")
	}
	if g.LeaderID != 1 {
		t.Errorf("expected leader 1, got %d", g.LeaderID)
	}
	if len(g.Members) != 1 || g.Members[0] != 1 {
		t.Errorf("expected members [1], got %v", g.Members)
	}
}

func TestCreateGroupAlreadyInGroup(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	_, err := m.CreateGroup(1)
	if !errors.Is(err, ErrAlreadyInGroup) {
		t.Errorf("expected ErrAlreadyInGroup, got %v", err)
	}
}

func TestInviteAndAccept(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	invite, err := m.InvitePlayer(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if invite.GroupID == 0 {
		t.Error("expected non-zero group ID in invite")
	}
	g, err := m.AcceptInvite(2, invite.GroupID)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(g.Members))
	}
}

func TestInviteAndDecline(t *testing.T) {
	m := NewManager()
	g, _ := m.CreateGroup(1)
	_, _ = m.InvitePlayer(1, 2)
	m.DeclineInvite(2, g.ID)
	_, err := m.AcceptInvite(2, g.ID)
	if !errors.Is(err, ErrInviteExpired) {
		t.Errorf("expected ErrInviteExpired after decline, got %v", err)
	}
}

func TestInviteNotLeader(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	inv, _ := m.InvitePlayer(1, 2)
	_, _ = m.AcceptInvite(2, inv.GroupID)
	_, err := m.InvitePlayer(2, 3) // player 2 is not leader
	if !errors.Is(err, ErrNotLeader) {
		t.Errorf("expected ErrNotLeader, got %v", err)
	}
}

func TestLeaveGroup(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	inv, _ := m.InvitePlayer(1, 2)
	_, _ = m.AcceptInvite(2, inv.GroupID)

	// Non-leader leaves
	g, disbanded := m.LeaveGroup(2)
	if disbanded {
		t.Error("group should not be disbanded")
	}
	if len(g.Members) != 1 {
		t.Errorf("expected 1 member, got %d", len(g.Members))
	}
}

func TestLeaderLeavePromotesNext(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	inv, _ := m.InvitePlayer(1, 2)
	_, _ = m.AcceptInvite(2, inv.GroupID)

	// Leader leaves
	g, disbanded := m.LeaveGroup(1)
	if disbanded {
		t.Error("group should not be disbanded")
	}
	if g.LeaderID != 2 {
		t.Errorf("expected new leader 2, got %d", g.LeaderID)
	}
}

func TestLastMemberLeaveDisbands(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	_, disbanded := m.LeaveGroup(1)
	if !disbanded {
		t.Error("group should be disbanded")
	}
}

func TestGroupFull(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	for i := uint32(2); i <= MaxGroupSize; i++ {
		inv, _ := m.InvitePlayer(1, i)
		_, _ = m.AcceptInvite(i, inv.GroupID)
	}
	_, err := m.InvitePlayer(1, MaxGroupSize+1)
	if !errors.Is(err, ErrGroupFull) {
		t.Errorf("expected ErrGroupFull, got %v", err)
	}
}

func TestKickPlayer(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	inv, _ := m.InvitePlayer(1, 2)
	_, _ = m.AcceptInvite(2, inv.GroupID)

	g, err := m.KickPlayer(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Members) != 1 {
		t.Errorf("expected 1 member after kick, got %d", len(g.Members))
	}
	// Verify kicked player has no group
	if m.GetGroup(2) != nil {
		t.Error("kicked player should not have a group")
	}
}

func TestGetGroup(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	g := m.GetGroup(1)
	if g == nil {
		t.Error("expected group, got nil")
	}
	if m.GetGroup(99) != nil {
		t.Error("expected nil for non-member")
	}
}

func TestGetGroupMembers(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	inv, _ := m.InvitePlayer(1, 2)
	_, _ = m.AcceptInvite(2, inv.GroupID)

	members := m.GetGroupMembers(1)
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	// Should contain both player IDs
	has1, has2 := false, false
	for _, id := range members {
		if id == 1 {
			has1 = true
		}
		if id == 2 {
			has2 = true
		}
	}
	if !has1 || !has2 {
		t.Errorf("members = %v, want [1, 2]", members)
	}
}

func TestGetGroupMembersNoGroup(t *testing.T) {
	m := NewManager()
	members := m.GetGroupMembers(99)
	if members != nil {
		t.Errorf("expected nil for non-member, got %v", members)
	}
}

func TestGetGroupMembersIsCopy(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateGroup(1)
	members := m.GetGroupMembers(1)
	members[0] = 999 // mutate the returned slice
	original := m.GetGroupMembers(1)
	if original[0] == 999 {
		t.Error("GetGroupMembers should return a copy, not the original slice")
	}
}

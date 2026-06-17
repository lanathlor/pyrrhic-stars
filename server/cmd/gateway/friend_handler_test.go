package main

import (
	"testing"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/session"
)

// realRepoGateway builds a gateway backed by a real in-memory sqlite repo so the
// friend persistence paths exercise actual queries.
func realRepoGateway(t *testing.T) *gateway {
	t.Helper()
	repo, err := persistence.NewGormRepo("sqlite", "")
	if err != nil {
		t.Fatalf("NewGormRepo: %v", err)
	}
	return newTestGateway(repo)
}

// registerOnline registers a session in the gateway with identity fields set and
// returns it together with its spy. zoneID lets tests place peers in different zones.
func registerOnline(t *testing.T, gw *gateway, uuid, username, charName, zoneID string) (*session.Session, *network.TestSpy) {
	t.Helper()
	conn, spy := network.NewTestClient()
	sess := gw.sessions.Register(conn)
	sess.UserUUID = uuid
	sess.Username = username
	sess.CharName = charName
	sess.Class = entity.ClassGunner
	sess.ZoneID = zoneID
	return sess, spy
}

func TestGroupInviteByNameCrossZone(t *testing.T) {
	gw := realRepoGateway(t)
	leader, _ := registerOnline(t, gw, "uuid-leader", "Leader", "LeaderChar", "hub")
	// Member is in a different zone (an arena instance) to prove cross-zone reach.
	member, memberSpy := registerOnline(t, gw, "uuid-member", "Member", "MemberChar", "arena_g1")

	if _, err := gw.groups.CreateGroup(leader.ID); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	// Invite by character name (type 1).
	payload := append([]byte{1, byte(len("MemberChar"))}, []byte("MemberChar")...)
	gw.handleGroupInviteByName(leader, payload)

	msgs := drainSpy(memberSpy)
	if findMessage(msgs, message.OpGroupInviteRecv) == nil {
		t.Fatalf("member did not receive OpGroupInviteRecv; got %d msgs", len(msgs))
	}
	_ = member
}

func TestGroupInviteByNameOfflineTarget(t *testing.T) {
	gw := realRepoGateway(t)
	leader, leaderSpy := registerOnline(t, gw, "uuid-leader", "Leader", "LeaderChar", "hub")
	if _, err := gw.groups.CreateGroup(leader.ID); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	payload := append([]byte{1, byte(len("Ghost"))}, []byte("Ghost")...)
	gw.handleGroupInviteByName(leader, payload)

	msgs := drainSpy(leaderSpy)
	if findMessage(msgs, message.OpGroupError) == nil {
		t.Fatal("expected OpGroupError for offline/unknown target")
	}
}

func TestFriendRequestOfflineReplayedOnConnect(t *testing.T) {
	gw := realRepoGateway(t)
	repo := gw.container.Repo

	// Seed two accounts, each with a character (so name resolution works).
	mk := func(uuid, username, charName string) {
		if err := repo.UpsertUser(uuid, username); err != nil {
			t.Fatalf("UpsertUser: %v", err)
		}
		c := &persistence.Character{UserID: uuid, ClassName: entity.ClassGunner, Name: charName}
		if err := repo.CreateCharacter(c); err != nil {
			t.Fatalf("CreateCharacter: %v", err)
		}
	}
	mk("uuid-alice", "Alice", "AliceChar")
	mk("uuid-bob", "Bob", "BobChar")

	// Alice (online) sends a request to Bob (offline).
	alice, _ := registerOnline(t, gw, "uuid-alice", "Alice", "AliceChar", "hub")
	reqPayload := append([]byte{1, byte(len("BobChar"))}, []byte("BobChar")...)
	gw.handleFriendRequest(alice, reqPayload)

	// Bob connects: pending request must be replayed.
	bob, bobSpy := registerOnline(t, gw, "uuid-bob", "Bob", "BobChar", "hub")
	gw.deliverPendingFriendRequests(bob)

	msgs := drainSpy(bobSpy)
	rec := findMessage(msgs, message.OpFriendRequestRecv)
	if rec == nil {
		t.Fatal("Bob did not receive replayed OpFriendRequestRecv")
	}

	// Bob accepts; both should then list each other as a friend.
	_, _, payload, _ := message.Decode(rec)
	requesterID, _ := readStr8Payload(payload, 0)
	respondPayload := append([]byte{1}, encodeStr8(requesterID)...)
	gw.handleFriendRespond(bob, respondPayload)

	if friends, _ := gw.friends.List("uuid-bob"); len(friends) != 1 {
		t.Errorf("bob has %d friends after accept, want 1", len(friends))
	}
	if friends, _ := gw.friends.List("uuid-alice"); len(friends) != 1 {
		t.Errorf("alice has %d friends after accept, want 1", len(friends))
	}
}

func TestFriendStatusBroadcastOnConnect(t *testing.T) {
	gw := realRepoGateway(t)
	repo := gw.container.Repo
	if err := repo.UpsertUser("uuid-a", "A"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	if err := repo.UpsertUser("uuid-b", "B"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	if err := repo.CreateFriendship("uuid-a", "uuid-b"); err != nil {
		t.Fatalf("CreateFriendship: %v", err)
	}
	if err := repo.AcceptFriendship("uuid-a", "uuid-b"); err != nil {
		t.Fatalf("AcceptFriendship: %v", err)
	}

	// A is online; B coming online should notify A.
	_, aSpy := registerOnline(t, gw, "uuid-a", "A", "AChar", "hub")
	registerOnline(t, gw, "uuid-b", "B", "BChar", "hub")
	gw.notifyFriendsStatus("uuid-b", true)

	msgs := drainSpy(aSpy)
	if findMessage(msgs, message.OpFriendStatus) == nil {
		t.Fatal("A did not receive OpFriendStatus when B came online")
	}
}

// encodeStr8 builds a [len:u8][bytes] string for test payloads.
func encodeStr8(s string) []byte {
	return append([]byte{byte(len(s))}, []byte(s)...)
}

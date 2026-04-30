package main

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"strings"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/session"
	"codex-online/server/internal/zone"
)

func handleServerMessage(gw *gateway, sess *session.Session, opcode uint16, payload []byte) {
	if message.IsGroupRelated(opcode) {
		gw.handleGroupMessage(sess, opcode, payload)
		return
	}

	switch opcode {
	case message.OpSetUsername:
		if len(payload) < 1 {
			return
		}
		nameLen := int(payload[0])
		if len(payload) < 1+nameLen {
			return
		}
		name := strings.TrimSpace(string(payload[1 : 1+nameLen]))
		if name == "" {
			name = fmt.Sprintf("Player_%d", sess.ID)
		}
		if len(name) > 20 {
			name = name[:20]
		}
		sess.Username = name
		slog.Info("username set", "player_id", sess.ID, "username", name)

	case message.OpSelectCharacter:
		if len(payload) < 4 {
			return
		}
		charID := uint(binary.LittleEndian.Uint32(payload[0:4]))

		char, err := gw.container.Repo.GetCharacterByID(charID)
		if err != nil || char == nil || char.PlayerID != sess.PlayerUUID {
			sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(5, "Character not found")))
			return
		}

		sess.CharID = char.ID
		sess.Class = char.ClassName
		sess.CharName = char.Name
		sess.Conn.Send(message.Encode(message.OpCharacterState, 0, codec.EncodeCharacterState(charToCodec(char))))

		gw.joinHubAfterCharSelect(sess, char)

	case message.OpCreateCharacter:
		gw.handleCreateCharacter(sess, payload)

	case message.OpJoinZone:
		zoneID := string(payload)
		if zoneID == "" {
			zoneID = zone.ZoneHub
		}

		zoneType := zone.ZoneTypeHub
		if strings.HasPrefix(zoneID, "arena") {
			zoneType = zone.ZoneTypeArena
		}

		zi := gw.getOrCreateZone(zoneID, zoneType)

		zi.mu.Lock()
		peerID := zi.nextID
		zi.nextID++
		zi.mu.Unlock()

		sess.PeerID = peerID
		sess.ZoneID = zoneID

		displayName := sess.CharName
		if displayName == "" {
			displayName = sess.Username
		}
		if displayName == "" {
			displayName = fmt.Sprintf("Player_%d", sess.ID)
			sess.Username = displayName
		}

		zi.zone.AddClient(&zone.Client{
			PeerID:   peerID,
			Username: displayName,
			Send:     sess.Conn.Send,
		})

		// Restore saved position for hub zone.
		if zoneType == zone.ZoneTypeHub && sess.CharID != 0 {
			if ch, _ := gw.container.Repo.GetCharacterByID(sess.CharID); ch != nil && (ch.PosX != 0 || ch.PosY != 0 || ch.PosZ != 0) {
				zi.zone.SetPlayerPosition(peerID, entity.Vec3{
					X: float32(ch.PosX),
					Y: float32(ch.PosY),
					Z: float32(ch.PosZ),
				}, float32(ch.RotY))
			}
		}

		for _, existingID := range zi.zone.GetPeerIDs() {
			if existingID == peerID {
				continue
			}
			sess.Conn.Send(message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(existingID)))
		}

		resp := make([]byte, 3)
		binary.BigEndian.PutUint16(resp[0:2], peerID)
		resp[2] = 0
		sess.Conn.Send(message.Encode(message.OpZoneJoined, 0, resp))
		slog.Info("peer joined zone", "zone_id", zoneID, "peer_id", peerID, "username", displayName)

		peerMsg := message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(peerID))
		zi.zone.Broadcast(peerMsg, peerID)

	default:
		slog.Warn("unknown server opcode", "opcode", opcode)
	}
}

// handleCreateCharacter processes OpCreateCharacter.
// Payload: [classLen:u8][class:...][nameLen:u8][name:...]
func (g *gateway) handleCreateCharacter(sess *session.Session, payload []byte) {
	if len(payload) < 2 {
		return
	}

	classLen := int(payload[0])
	if len(payload) < 1+classLen+1 {
		return
	}
	className := string(payload[1 : 1+classLen])

	nameLen := int(payload[1+classLen])
	if len(payload) < 1+classLen+1+nameLen {
		return
	}
	charName := strings.TrimSpace(string(payload[1+classLen+1 : 1+classLen+1+nameLen]))

	validClasses := map[string]bool{"gunner": true, "vanguard": true, "blade_dancer": true}
	if !validClasses[className] {
		sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(4, "Invalid class")))
		return
	}

	if len(charName) < 2 || len(charName) > 20 {
		sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(3, "Name must be 2-20 characters")))
		return
	}
	for _, r := range charName {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != ' ' && r != '-' && r != '_' {
			sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(3, "Name must be alphanumeric (spaces, hyphens, underscores allowed)")))
			return
		}
	}

	count, err := g.container.Repo.CountCharacters(sess.PlayerUUID)
	if err != nil {
		slog.Error("count characters", "error", err)
		sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(2, "Failed to check limit")))
		return
	}
	if count >= 100 {
		sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(2, "Character limit reached")))
		return
	}

	taken, err := g.container.Repo.IsCharacterNameTaken(charName)
	if err != nil {
		slog.Error("check name taken", "error", err)
		sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(1, "Name already taken")))
		return
	}
	if taken {
		sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(1, "Name already taken")))
		return
	}

	char := &persistence.Character{
		PlayerID:  sess.PlayerUUID,
		ClassName: className,
		Name:      charName,
	}
	if err := g.container.Repo.CreateCharacter(char); err != nil {
		slog.Error("create character", "error", err)
		sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(1, "Name already taken")))
		return
	}

	sess.CharID = char.ID
	sess.Class = char.ClassName
	sess.CharName = char.Name
	sess.Conn.Send(message.Encode(message.OpCharacterState, 0, codec.EncodeCharacterState(charToCodec(char))))

	g.joinHubAfterCharSelect(sess, char)
}

// joinHubAfterCharSelect handles the shared logic for joining the hub zone
// after a character is selected or created.
func (g *gateway) joinHubAfterCharSelect(sess *session.Session, char *persistence.Character) {
	zoneID := zone.ZoneHub
	zi := g.getOrCreateZone(zoneID, zone.ZoneTypeHub)

	zi.mu.Lock()
	peerID := zi.nextID
	zi.nextID++
	zi.mu.Unlock()

	sess.PeerID = peerID
	sess.ZoneID = zoneID

	displayName := sess.CharName
	if displayName == "" {
		displayName = sess.Username
	}
	if displayName == "" {
		displayName = fmt.Sprintf("Player_%d", sess.ID)
	}

	zi.zone.AddClient(&zone.Client{
		PeerID:   peerID,
		Username: displayName,
		Send:     sess.Conn.Send,
	})

	if char != nil && (char.PosX != 0 || char.PosY != 0 || char.PosZ != 0) {
		zi.zone.SetPlayerPosition(peerID, entity.Vec3{
			X: float32(char.PosX),
			Y: float32(char.PosY),
			Z: float32(char.PosZ),
		}, float32(char.RotY))
	}

	if sess.Class != "" && sess.Class != "gunner" {
		zi.zone.QueueInput(peerID, message.OpInteractInput, codec.EncodeInteractInput(message.InteractClassSelect, sess.Class))
	}

	for _, existingID := range zi.zone.GetPeerIDs() {
		if existingID == peerID {
			continue
		}
		sess.Conn.Send(message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(existingID)))
	}

	resp := make([]byte, 3)
	binary.BigEndian.PutUint16(resp[0:2], peerID)
	resp[2] = 0
	sess.Conn.Send(message.Encode(message.OpZoneJoined, 0, resp))
	slog.Info("character selected, joined hub", "player_id", sess.ID, "char_id", sess.CharID, "class", sess.Class, "peer_id", peerID)

	peerMsg := message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(peerID))
	zi.zone.Broadcast(peerMsg, peerID)
}

// charToCodec converts a persistence.Character to a codec.CharacterInfo.
func charToCodec(c *persistence.Character) codec.CharacterInfo {
	return codec.CharacterInfo{
		ID:        uint32(c.ID),
		ClassName: c.ClassName,
		Name:      c.Name,
		PosX:      float32(c.PosX),
		PosY:      float32(c.PosY),
		PosZ:      float32(c.PosZ),
		RotY:      float32(c.RotY),
	}
}

// encodeCharacterListMsg builds a full OpCharacterList wire message from persistence characters.
func encodeCharacterListMsg(username string, chars []*persistence.Character) []byte {
	infos := make([]codec.CharacterInfo, len(chars))
	for i, c := range chars {
		infos[i] = charToCodec(c)
	}
	var lastCharID uint32
	if len(chars) > 0 {
		lastCharID = uint32(chars[0].ID)
	}
	return message.Encode(message.OpCharacterList, 0, codec.EncodeCharacterList(username, infos, lastCharID))
}


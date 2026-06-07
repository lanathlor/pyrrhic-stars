package main

import (
	"encoding/binary"
	"errors"
	"log/slog"

	"codex-online/server/internal/character"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/message"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/session"
)

// handleCharacterMessage processes character CRUD opcodes.
func (g *gateway) handleCharacterMessage(sess *session.Session, opcode uint16, payload []byte) {
	switch opcode {
	case message.OpSelectCharacter:
		if len(payload) < 4 {
			return
		}
		charID := uint(binary.LittleEndian.Uint32(payload[0:4]))

		char, err := g.characters.Select(charID, sess.UserUUID)
		if err != nil {
			sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(5, "Character not found")))
			return
		}

		applyCharacterSession(sess, char)
		g.joinHubAfterCharSelect(sess)

	case message.OpCreateCharacter:
		g.handleCreateCharacter(sess, payload)

	default:
		slog.Warn("unknown character opcode", "opcode", opcode)
	}
}

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
	charName := string(payload[1+classLen+1 : 1+classLen+1+nameLen])

	char, err := g.characters.Create(sess.UserUUID, className, charName)
	if err != nil {
		code, msg := characterErrorResponse(err)
		sess.Conn.Send(message.Encode(message.OpCharacterError, 0, codec.EncodeCharacterError(code, msg)))
		return
	}
	if err := g.inventory.SpawnStarterGear(char.ID); err != nil {
		slog.Error("spawn starter gear", "char_id", char.ID, "error", err)
	}
	applyCharacterSession(sess, char)
	g.joinHubAfterCharSelect(sess)
}

func applyCharacterSession(sess *session.Session, char *persistence.Character) {
	sess.Mu.Lock()
	sess.CharID = char.ID
	sess.Class = char.ClassName
	sess.Spec = char.SpecID
	sess.CharName = char.Name
	sess.Mu.Unlock()
	sess.Conn.Send(message.Encode(message.OpCharacterState, 0, codec.EncodeCharacterState(charToCodec(char))))
}

// characterErrorResponse maps service errors to wire error codes and messages.
func characterErrorResponse(err error) (uint8, string) {
	switch {
	case errors.Is(err, character.ErrInvalidClass):
		return 4, "Invalid class"
	case errors.Is(err, character.ErrNameLength):
		return 3, "Name must be 2-20 characters"
	case errors.Is(err, character.ErrNameChars):
		return 3, "Name must be alphanumeric (spaces, hyphens, underscores allowed)"
	case errors.Is(err, character.ErrLimitReached):
		return 2, "Character limit reached"
	case errors.Is(err, character.ErrNameTaken):
		return 1, "Name already taken"
	default:
		return 1, "Character creation failed"
	}
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

// encodeCharacterStateMsg builds a full OpCharacterState wire message from a persistence character.
func encodeCharacterStateMsg(c *persistence.Character) []byte {
	return message.Encode(message.OpCharacterState, 0, codec.EncodeCharacterState(charToCodec(c)))
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

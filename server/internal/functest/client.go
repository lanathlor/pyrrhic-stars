// Package functest provides a functional test client that connects to a real
// running gateway over WebSocket and speaks the binary wire protocol.
//
// Unlike integration tests (which spin up in-process gateways with Go fallback
// level data), functional tests hit the real Docker-hosted server and exercise
// the full stack: JSON level loading, zone tick loop, binary serialization.
//
// Usage:
//
//	func TestHubSpawn(t *testing.T) {
//	    c := functest.Dial(t, functest.DefaultAddr)
//	    defer c.Close()
//	    c.JoinZone("hub")
//	    ws := c.WaitWorldState(2 * time.Second)
//	    me := ws.Player(c.PeerID)
//	    require(t, me != nil, "local player not in world state")
//	    assertNear(t, me.PosY, -199.9, 1.0, "spawn Y")
//	}
package functest

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/message"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// DefaultAddr is the gateway address when running via docker compose.
const DefaultAddr = "ws://localhost:7777/ws"

// Client is a functional test WebSocket client.
type Client struct {
	t      *testing.T
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc

	PeerID   uint16
	Username string
	UUID     string

	mu   sync.Mutex
	msgs []RawMsg
	cond *sync.Cond
}

// RawMsg is a decoded wire message.
type RawMsg struct {
	Opcode   uint16
	SenderID uint16
	Payload  []byte
}

// Dial connects to a running gateway. It generates a random UUID and sets the
// given username (or "FuncTest" if empty). The test is failed if the connection
// cannot be established.
// TryDial attempts to connect without failing the test. Returns an error if unreachable.
func TryDial(addr, username string) (*Client, error) {
	url := fmt.Sprintf("%s?uuid=%s&username=%s", addr, uuid.New().String(), username)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	conn.SetReadLimit(1 << 20)
	c := &Client{
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
	c.cond = sync.NewCond(&c.mu)
	go c.readLoop()
	return c, nil
}

func Dial(t *testing.T, addr string, username ...string) *Client {
	t.Helper()
	name := "FuncTest"
	if len(username) > 0 && username[0] != "" {
		name = username[0]
	}
	return DialWithUUID(t, addr, uuid.New().String(), name)
}

// DialWithUUID connects to a running gateway using a specific UUID.
// This allows reconnecting as the same player to test persistence.
func DialWithUUID(t *testing.T, addr, playerUUID, username string) *Client {
	t.Helper()

	url := fmt.Sprintf("%s?uuid=%s&username=%s", addr, playerUUID, username)

	ctx, cancel := context.WithCancel(context.Background())
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		cancel()
		t.Fatalf("functest.Dial: %v (is the gateway running?)", err)
	}
	// Allow large messages (world state can be big).
	conn.SetReadLimit(1 << 20)

	c := &Client{
		t:        t,
		conn:     conn,
		ctx:      ctx,
		cancel:   cancel,
		Username: username,
		UUID:     playerUUID,
	}
	c.cond = sync.NewCond(&c.mu)
	go c.readLoop()

	t.Cleanup(func() { c.Close() })
	return c
}

// Close shuts down the connection.
func (c *Client) Close() {
	_ = c.conn.Close(websocket.StatusNormalClosure, "test done")
	c.cancel()
}

// readLoop receives messages and appends them to the buffer.
func (c *Client) readLoop() {
	for {
		_, data, err := c.conn.Read(c.ctx)
		if err != nil {
			return
		}
		opcode, senderID, payload, err := message.Decode(data)
		if err != nil {
			continue
		}
		c.mu.Lock()
		c.msgs = append(c.msgs, RawMsg{
			Opcode:   opcode,
			SenderID: senderID,
			Payload:  append([]byte(nil), payload...),
		})
		c.cond.Broadcast()
		c.mu.Unlock()
	}
}

// send writes an encoded message on the wire.
func (c *Client) send(data []byte) {
	c.t.Helper()
	if err := c.conn.Write(c.ctx, websocket.MessageBinary, data); err != nil {
		c.t.Fatalf("functest send: %v", err)
	}
}

// SendAbility sends an OpAbilityInput with the given action ID.
func (c *Client) SendAbility(action uint8) {
	c.t.Helper()
	payload := codec.EncodeAbilityInput(action, 0.0)
	c.send(message.Encode(message.OpAbilityInput, c.PeerID, payload))
}

// JoinZone sends OpJoinZone, waits for OpZoneJoined, and stores the PeerID.
func (c *Client) JoinZone(zoneID string) {
	c.t.Helper()
	c.send(message.Encode(message.OpJoinZone, 0, []byte(zoneID)))
	msg := c.WaitFor(message.OpZoneJoined, 5*time.Second)
	if len(msg.Payload) < 3 {
		c.t.Fatalf("ZoneJoined payload too short: %d bytes", len(msg.Payload))
	}
	c.PeerID = binary.BigEndian.Uint16(msg.Payload[0:2])
}

// SelectCharacter sends OpSelectCharacter with a character ID, waits for
// OpCharacterState confirmation and OpZoneJoined, then stores the PeerID.
func (c *Client) SelectCharacter(charID uint32) CharacterState {
	c.t.Helper()
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, charID)
	c.send(message.Encode(message.OpSelectCharacter, 0, payload))

	cs := c.WaitCharacterState(5 * time.Second)
	msg := c.WaitFor(message.OpZoneJoined, 5*time.Second)
	if len(msg.Payload) < 3 {
		c.t.Fatalf("ZoneJoined payload too short: %d bytes", len(msg.Payload))
	}
	c.PeerID = binary.BigEndian.Uint16(msg.Payload[0:2])
	return cs
}

// CreateCharacter sends OpCreateCharacter, waits for OpCharacterState + OpZoneJoined.
// Returns the confirmed CharacterState. Fails the test on timeout.
func (c *Client) CreateCharacter(className, charName string) CharacterState {
	c.t.Helper()
	classBytes := []byte(className)
	nameBytes := []byte(charName)
	payload := []byte{byte(len(classBytes))}
	payload = append(payload, classBytes...)
	payload = append(payload, byte(len(nameBytes)))
	payload = append(payload, nameBytes...)
	c.send(message.Encode(message.OpCreateCharacter, 0, payload))

	cs := c.WaitCharacterState(5 * time.Second)
	msg := c.WaitFor(message.OpZoneJoined, 5*time.Second)
	if len(msg.Payload) < 3 {
		c.t.Fatalf("ZoneJoined payload too short: %d bytes", len(msg.Payload))
	}
	c.PeerID = binary.BigEndian.Uint16(msg.Payload[0:2])
	return cs
}

// SendCreateCharacter sends OpCreateCharacter without waiting (for error tests).
func (c *Client) SendCreateCharacter(className, charName string) {
	c.t.Helper()
	classBytes := []byte(className)
	nameBytes := []byte(charName)
	payload := []byte{byte(len(classBytes))}
	payload = append(payload, classBytes...)
	payload = append(payload, byte(len(nameBytes)))
	payload = append(payload, nameBytes...)
	c.send(message.Encode(message.OpCreateCharacter, 0, payload))
}

// CharacterError is a decoded OpCharacterError.
type CharacterError struct {
	Code    uint8
	Message string
}

// WaitCharacterError waits for an OpCharacterError and decodes it.
func (c *Client) WaitCharacterError(timeout time.Duration) CharacterError {
	c.t.Helper()
	msg := c.WaitFor(message.OpCharacterError, timeout)
	ce := CharacterError{}
	if len(msg.Payload) >= 2 {
		ce.Code = msg.Payload[0]
		msgLen := int(msg.Payload[1])
		if len(msg.Payload) >= 2+msgLen {
			ce.Message = string(msg.Payload[2 : 2+msgLen])
		}
	}
	return ce
}

// WaitCharacterList waits for an OpCharacterList and decodes it.
func (c *Client) WaitCharacterList(timeout time.Duration) CharacterList {
	c.t.Helper()
	msg := c.WaitFor(message.OpCharacterList, timeout)
	return decodeCharacterList(msg.Payload)
}

// CharacterList is a decoded OpCharacterList message.
type CharacterList struct {
	Username   string
	Characters []CharacterListEntry
	LastCharID uint32
}

// CharacterListEntry is a single character in the list.
type CharacterListEntry struct {
	CharID    uint32
	ClassName string
	Name      string
	PosX      float32
	PosY      float32
	PosZ      float32
	RotY      float32
}

func decodeCharacterList(data []byte) CharacterList {
	cl := CharacterList{}
	if len(data) < 1 {
		return cl
	}
	off := 0

	// Username prefix.
	uLen := int(data[off])
	off++
	if uLen > 0 {
		cl.Username = string(data[off : off+uLen])
		off += uLen
	}

	count := int(data[off])
	off++

	for i := 0; i < count; i++ {
		e := CharacterListEntry{}
		e.CharID = leU32(data[off:])
		off += 4
		classLen := int(data[off])
		off++
		e.ClassName = string(data[off : off+classLen])
		off += classLen
		nameLen := int(data[off])
		off++
		e.Name = string(data[off : off+nameLen])
		off += nameLen
		e.PosX = leF32(data[off:])
		off += 4
		e.PosY = leF32(data[off:])
		off += 4
		e.PosZ = leF32(data[off:])
		off += 4
		e.RotY = leF32(data[off:])
		off += 4
		cl.Characters = append(cl.Characters, e)
	}

	if off+4 <= len(data) {
		cl.LastCharID = leU32(data[off:])
	}
	return cl
}

// SendPlayerInput sends a movement packet.
func (c *Client) SendPlayerInput(posX, posY, posZ, rotY float32, tick uint32) {
	c.t.Helper()
	buf := encodePlayerInput(posX, posY, posZ, rotY, tick, 0, 0)
	c.send(message.Encode(message.OpPlayerInput, 0, buf))
}

// SendInteract sends an interact input (ready toggle, class select, etc.).
func (c *Client) SendInteract(action uint8, className ...string) {
	c.t.Helper()
	payload := []byte{action}
	if len(className) > 0 {
		name := []byte(className[0])
		payload = append(payload, byte(len(name)))
		payload = append(payload, name...)
	}
	c.send(message.Encode(message.OpInteractInput, 0, payload))
}

// ReadyUp sends the ready toggle interact.
func (c *Client) ReadyUp() {
	c.t.Helper()
	c.SendInteract(message.InteractReadyToggle)
}

// ---------------------------------------------------------------------------
// Message waiting
// ---------------------------------------------------------------------------

// WaitFor blocks until a message with the given opcode arrives.
func (c *Client) WaitFor(opcode uint16, timeout time.Duration) RawMsg {
	c.t.Helper()
	deadline := time.Now().Add(timeout)

	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		for i, m := range c.msgs {
			if m.Opcode == opcode {
				c.msgs = append(c.msgs[:i], c.msgs[i+1:]...)
				return m
			}
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			c.t.Fatalf("timeout waiting for opcode 0x%04X (%d buffered)", opcode, len(c.msgs))
		}
		done := make(chan struct{})
		go func() {
			timer := time.NewTimer(remaining)
			defer timer.Stop()
			select {
			case <-timer.C:
				c.cond.Broadcast()
			case <-done:
			}
		}()
		c.cond.Wait()
		close(done)
	}
}

// Drain discards all buffered messages.
func (c *Client) Drain() {
	c.mu.Lock()
	c.msgs = c.msgs[:0]
	c.mu.Unlock()
}

// ---------------------------------------------------------------------------
// WorldState parsing
// ---------------------------------------------------------------------------

// WorldState is a decoded OpWorldState snapshot.
type WorldState struct {
	Tick    uint32
	Players []PlayerState
	Enemies []EnemyState
	NPCs    []NPCState
}

// PlayerState is a single player from a WorldState.
type PlayerState struct {
	PeerID      uint16
	PosX        float32
	PosY        float32
	PosZ        float32
	RotY        float32
	Health      float32
	State       uint8
	ClassName   string
	Username    string
	VisualState uint8
	Stamina     float32
}

// EnemyState is a single enemy from a WorldState.
type EnemyState struct {
	ID      uint16
	Alive   bool
	PosX    float32
	PosY    float32
	PosZ    float32
	RotY    float32
	Health  float32
	State   uint8
	Phase   uint8
	DefName string
}

// NPCState is a single NPC from a WorldState.
type NPCState struct {
	ID      uint16
	PosX    float32
	PosY    float32
	PosZ    float32
	RotY    float32
	State   uint8
	DefName string
}

// Player finds a player by peer ID, or returns nil.
func (ws *WorldState) Player(peerID uint16) *PlayerState {
	for i := range ws.Players {
		if ws.Players[i].PeerID == peerID {
			return &ws.Players[i]
		}
	}
	return nil
}

// WaitWorldState waits for the next OpWorldState and decodes it.
func (c *Client) WaitWorldState(timeout time.Duration) *WorldState {
	c.t.Helper()
	msg := c.WaitFor(message.OpWorldState, timeout)
	ws, err := decodeWorldState(msg.Payload)
	if err != nil {
		c.t.Fatalf("decode WorldState: %v", err)
	}
	return ws
}

// decodeWorldState parses an OpWorldState payload into a WorldState.
func decodeWorldState(data []byte) (*WorldState, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("world state too short: %d bytes", len(data))
	}

	ws := &WorldState{}
	off := 0

	ws.Tick = leU32(data[off:])
	off += 4

	playerCount := int(data[off])
	off++

	for i := 0; i < playerCount; i++ {
		if off+2 > len(data) {
			return nil, fmt.Errorf("truncated player %d header", i)
		}
		p := PlayerState{}
		p.PeerID = leU16(data[off:])
		off += 2
		p.PosX = leF32(data[off:])
		off += 4
		p.PosY = leF32(data[off:])
		off += 4
		p.PosZ = leF32(data[off:])
		off += 4
		p.RotY = leF32(data[off:])
		off += 4
		p.Health = leF32(data[off:])
		off += 4
		off += 4 // max_health
		p.State = data[off]
		off++

		// class name (str8)
		classLen := int(data[off])
		off++
		p.ClassName = string(data[off : off+classLen])
		off += classLen

		// username (str8)
		nameLen := int(data[off])
		off++
		p.Username = string(data[off : off+nameLen])
		off += nameLen

		// visual_state (u8)
		p.VisualState = data[off]
		off++
		off += 4 // aim pitch
		off++    // buff flags
		off++    // config
		p.Stamina = leF32(data[off:])
		off += 4 // stamina
		off += 4 // BD shield HP
		off += 4 // munitions
		off += 4 // resonance

		ws.Players = append(ws.Players, p)
	}

	// Enemies
	if off >= len(data) {
		return ws, nil
	}
	enemyCount := int(data[off])
	off++

	for i := 0; i < enemyCount; i++ {
		e := EnemyState{}
		e.Alive = data[off] == 1
		off++
		e.ID = leU16(data[off:])
		off += 2
		e.PosX = leF32(data[off:])
		off += 4
		e.PosY = leF32(data[off:])
		off += 4
		e.PosZ = leF32(data[off:])
		off += 4
		e.RotY = leF32(data[off:])
		off += 4
		e.Health = leF32(data[off:])
		off += 4
		e.State = data[off]
		off++
		e.Phase = data[off]
		off++
		off += 4 // max health
		defLen := int(data[off])
		off++
		e.DefName = string(data[off : off+defLen])
		off += defLen
		off += 4 * 6 // ranged target pos(3) + charge dir(3)
		off += 4     // melee cone angle
		off += 4     // melee range

		ws.Enemies = append(ws.Enemies, e)
	}

	// Projectiles
	if off >= len(data) {
		return ws, nil
	}
	projCount := int(data[off])
	off++
	off += projCount * (4 + 4*6) // id + pos(3) + dir(3)

	// NPCs
	if off >= len(data) {
		return ws, nil
	}
	npcCount := int(data[off])
	off++

	for i := 0; i < npcCount; i++ {
		n := NPCState{}
		n.ID = leU16(data[off:])
		off += 2
		n.PosX = leF32(data[off:])
		off += 4
		n.PosY = leF32(data[off:])
		off += 4
		n.PosZ = leF32(data[off:])
		off += 4
		n.RotY = leF32(data[off:])
		off += 4
		n.State = data[off]
		off++
		defLen := int(data[off])
		off++
		n.DefName = string(data[off : off+defLen])
		off += defLen

		ws.NPCs = append(ws.NPCs, n)
	}

	return ws, nil
}

// ---------------------------------------------------------------------------
// GameFlowEvent parsing
// ---------------------------------------------------------------------------

// GameFlowEvent is a decoded OpGameFlowEvent.
type GameFlowEvent struct {
	FlowType uint8
	Text     string
}

// WaitGameFlow waits for an OpGameFlowEvent and decodes it.
func (c *Client) WaitGameFlow(timeout time.Duration) GameFlowEvent {
	c.t.Helper()
	msg := c.WaitFor(message.OpGameFlowEvent, timeout)
	ev := GameFlowEvent{FlowType: msg.Payload[0]}
	if len(msg.Payload) > 2 {
		textLen := int(msg.Payload[1])
		if len(msg.Payload) >= 2+textLen {
			ev.Text = string(msg.Payload[2 : 2+textLen])
		}
	}
	return ev
}

// ---------------------------------------------------------------------------
// CharacterState parsing (sent on connect, before JoinZone)
// ---------------------------------------------------------------------------

// CharacterState is a decoded OpCharacterState message.
type CharacterState struct {
	CharID    uint32
	ClassName string
	Name      string
	PosX      float32
	PosY      float32
	PosZ      float32
	RotY      float32
}

// WaitCharacterState waits for an OpCharacterState and decodes it.
func (c *Client) WaitCharacterState(timeout time.Duration) CharacterState {
	c.t.Helper()
	msg := c.WaitFor(message.OpCharacterState, timeout)
	return decodeCharacterState(msg.Payload)
}

// decodeCharacterState parses an OpCharacterState payload.
// Format: [charID:u32 LE][classLen:u8][class:...][nameLen:u8][name:...][pos x/y/z/rotY:f32 LE]
func decodeCharacterState(data []byte) CharacterState {
	cs := CharacterState{}
	if len(data) < 5 {
		return cs
	}
	off := 0
	cs.CharID = leU32(data[off:])
	off += 4
	classLen := int(data[off])
	off++
	if classLen > 0 {
		cs.ClassName = string(data[off : off+classLen])
		off += classLen
	}
	nameLen := int(data[off])
	off++
	if nameLen > 0 {
		cs.Name = string(data[off : off+nameLen])
		off += nameLen
	}
	if off+16 <= len(data) {
		cs.PosX = leF32(data[off:])
		off += 4
		cs.PosY = leF32(data[off:])
		off += 4
		cs.PosZ = leF32(data[off:])
		off += 4
		cs.RotY = leF32(data[off:])
	}
	return cs
}

// ---------------------------------------------------------------------------
// Wire helpers (little-endian)
// ---------------------------------------------------------------------------

func leU16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
func leU32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
func leF32(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}

func appendF32(buf []byte, v float32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, math.Float32bits(v))
	return append(buf, b...)
}

func appendU32(buf []byte, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(buf, b...)
}

func encodePlayerInput(posX, posY, posZ, rotY float32, tick uint32, visualState uint8, aimPitch float32) []byte {
	buf := make([]byte, 0, 25)
	buf = appendF32(buf, posX)
	buf = appendF32(buf, posY)
	buf = appendF32(buf, posZ)
	buf = appendF32(buf, rotY)
	buf = appendU32(buf, tick)
	buf = append(buf, visualState)
	buf = appendF32(buf, aimPitch)
	return buf
}

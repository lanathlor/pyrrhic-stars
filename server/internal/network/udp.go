package network

import (
	"crypto/rand"
	"encoding/binary"
	"log/slog"
	"net"
	"sync"
)

// UDPServer manages a single shared UDP socket for all zones.
// The read loop handles token-based association and dispatches incoming
// player input to the appropriate zone. Zone tick goroutines write
// outbound packets directly via Client.SendUDP (no channel, no copy).
type UDPServer struct {
	conn *net.UDPConn

	mu     sync.Mutex
	tokens map[[16]byte]*pendingAssoc // token -> session info
	routes map[string]*routeEntry     // addr string -> associated session
}

type pendingAssoc struct {
	Client *Client
	SessID uint32
}

type routeEntry struct {
	Client *Client
	SessID uint32
	Addr   *net.UDPAddr
}

// NewUDPServer binds a UDP socket on the given address and returns a server
// ready to accept associations. Call ReadLoop in a goroutine to start processing.
func NewUDPServer(addr string) (*UDPServer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return &UDPServer{
		conn:   conn,
		tokens: make(map[[16]byte]*pendingAssoc),
		routes: make(map[string]*routeEntry),
	}, nil
}

// Port returns the local port the UDP server is listening on.
func (s *UDPServer) Port() int {
	addr, ok := s.conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return 0
	}
	return addr.Port
}

// Conn returns the underlying UDP connection for direct writes from zone ticks.
func (s *UDPServer) Conn() *net.UDPConn {
	return s.conn
}

// GenerateToken creates a crypto-random 16-byte token for UDP association.
// The token is stored internally and validated when the client sends it
// back over UDP as OpUDPAssociateAck.
func (s *UDPServer) GenerateToken(client *Client, sessID uint32) [16]byte {
	var token [16]byte
	_, _ = rand.Read(token[:])
	s.mu.Lock()
	s.tokens[token] = &pendingAssoc{Client: client, SessID: sessID}
	s.mu.Unlock()
	return token
}

// ReadLoop reads incoming UDP packets and either completes token association
// or dispatches player input to zones. Blocks until the connection is closed.
//
// dispatch is called for every validated packet from an associated client.
// It receives the session ID, source peer ID (from the message header),
// opcode, and payload. The callback should route OpPlayerInput to the zone.
func (s *UDPServer) ReadLoop(dispatch func(sessID uint32, peerID, opcode uint16, payload []byte)) {
	buf := make([]byte, 2048) // max UDP payload
	slog.Info("udp read loop started", "local_addr", s.conn.LocalAddr().String())
	for {
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			slog.Info("udp read loop exiting", "error", err)
			return // connection closed
		}
		slog.Info("udp packet received", "from", addr.String(), "bytes", n)
		if n < 4 { // minimum: 4-byte header
			continue
		}

		opcode := binary.BigEndian.Uint16(buf[0:2])
		senderID := binary.BigEndian.Uint16(buf[2:4])

		// Association handshake: client sends OpUDPAssociateAck with [token:16].
		if opcode == opUDPAssociateAck {
			s.handleAssociation(buf[4:n], addr)
			continue
		}

		// Validate source address against associated route.
		addrKey := addr.String()
		s.mu.Lock()
		route, ok := s.routes[addrKey]
		s.mu.Unlock()
		if !ok {
			continue // unknown source, drop
		}

		// Copy payload out of the shared read buffer before dispatching.
		// The dispatch callback stores the slice (via QueueInput) and it
		// must survive until the zone tick processes it.
		payload := make([]byte, n-4)
		copy(payload, buf[4:n])
		dispatch(route.SessID, senderID, opcode, payload)
	}
}

// opUDPAssociateAck is the opcode for the client's UDP association response.
// Defined here to avoid a circular import with the message package.
const opUDPAssociateAck uint16 = 0xFF11

func (s *UDPServer) handleAssociation(payload []byte, addr *net.UDPAddr) {
	if len(payload) < 16 {
		return
	}
	var token [16]byte
	copy(token[:], payload[:16])

	s.mu.Lock()
	pending, ok := s.tokens[token]
	if !ok {
		s.mu.Unlock()
		return
	}
	delete(s.tokens, token)

	addrKey := addr.String()
	s.routes[addrKey] = &routeEntry{
		Client: pending.Client,
		SessID: pending.SessID,
		Addr:   addr,
	}
	s.mu.Unlock()

	// Set the UDP address on the network client (atomic, lock-free for readers).
	pending.Client.AssociateUDP(s.conn, addr)
	slog.Info("udp associated", "sess_id", pending.SessID, "addr", addrKey)
}

// RemoveClient removes the UDP route for a disconnecting client.
// Safe to call even if the client never associated.
func (s *UDPServer) RemoveClient(client *Client) {
	addr := client.UDPAddrString()
	if addr == "" {
		return
	}
	s.mu.Lock()
	delete(s.routes, addr)
	s.mu.Unlock()
}

// Close shuts down the UDP listener.
func (s *UDPServer) Close() error {
	return s.conn.Close()
}

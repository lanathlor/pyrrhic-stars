package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"net"
	"strings"
	"sync"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/abilitycatalog"
	"codex-online/server/internal/auth"
	"codex-online/server/internal/character"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/container"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/friend"
	"codex-online/server/internal/group"
	"codex-online/server/internal/inventory"
	"codex-online/server/internal/item"
	"codex-online/server/internal/level"
	"codex-online/server/internal/merchant"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/overflux"
	"codex-online/server/internal/progression"
	"codex-online/server/internal/session"
	"codex-online/server/internal/user"
	"codex-online/server/internal/zone"
)

// defaultOpenWorldZone is the zone ID for the main open-world zone.
const defaultOpenWorldZone = "hub"

// gateway manages zones, player sessions, and groups.
type gateway struct {
	container   *container.Container
	zones       map[string]*zoneInstance
	levels      map[string]*level.Level // cached level data by zone name
	sessions    *session.Registry
	groups      *group.Manager
	friends     *friend.Service
	users       *user.Service
	characters  *character.Service
	inventory   *inventory.Service
	merchant    *merchant.Service
	progression *progression.Service
	catalog     *abilitycatalog.Catalog
	abilityEng  *ability.Engine      // for stat lookups when building catalog
	udpServer   *network.UDPServer   // shared UDP socket for game state transport
	verifier    auth.SessionVerifier // resolves Kratos session tokens to identities
	mu          sync.Mutex           // protects zones and levels
	devMode     bool                 // CODEX_DEV=1 enables debug features

	// udpPublicHost is the host/IP clients dial to reach the UDP socket. When UDP
	// is exposed separately from the WebSocket (e.g. a dedicated UDP LoadBalancer),
	// set GATEWAY_UDP_PUBLIC_HOST so clients dial the right host. Empty means
	// clients reuse the WebSocket host (correct when WS and UDP share an IP).
	udpPublicHost string

	// udpPublicPort is the port clients dial to reach the UDP socket, when it
	// differs from the listen port (e.g. a Kubernetes NodePort fronting the
	// gateway, since Scaleway Kapsule cannot expose UDP via LoadBalancer or
	// hostPort). Set GATEWAY_UDP_PUBLIC_PORT. Zero means advertise the listen port.
	udpPublicPort uint16

	// Connection limiting
	connMu    sync.Mutex
	connPerIP map[string]int // IP -> active connection count
}

const (
	maxConnsPerIP = 5
	maxConnsTotal = 200
)

// acquireConn checks connection limits for the given remote address.
// Returns true if the connection is allowed, false if it should be rejected.
func (g *gateway) acquireConn(remoteAddr string) bool {
	ip, _, _ := net.SplitHostPort(remoteAddr)
	if ip == "" {
		ip = remoteAddr
	}
	g.connMu.Lock()
	defer g.connMu.Unlock()
	total := 0
	for _, n := range g.connPerIP {
		total += n
	}
	if total >= maxConnsTotal {
		return false
	}
	if g.connPerIP[ip] >= maxConnsPerIP {
		return false
	}
	g.connPerIP[ip]++
	return true
}

// releaseConn decrements the connection count for the given remote address.
func (g *gateway) releaseConn(remoteAddr string) {
	ip, _, _ := net.SplitHostPort(remoteAddr)
	if ip == "" {
		ip = remoteAddr
	}
	g.connMu.Lock()
	defer g.connMu.Unlock()
	g.connPerIP[ip]--
	if g.connPerIP[ip] <= 0 {
		delete(g.connPerIP, ip)
	}
}

type zoneInstance struct {
	zone   *zone.Zone
	cancel context.CancelFunc
	nextID uint16
	mu     sync.Mutex
}

func newGateway(ctr *container.Container) *gateway {
	return &gateway{
		container:   ctr,
		zones:       make(map[string]*zoneInstance),
		levels:      make(map[string]*level.Level),
		sessions:    session.NewRegistry(),
		groups:      group.NewManager(),
		friends:     friend.NewService(ctr.Repo),
		users:       user.NewService(ctr.Repo),
		characters:  character.NewService(ctr.Repo),
		inventory:   inventory.NewService(ctr.Repo),
		merchant:    merchant.NewService(ctr.Repo),
		progression: progression.NewService(ctr.Repo),
		connPerIP:   make(map[string]int),
	}
}

// loadLevel returns a cached level or loads it from shared/levels/.
func (g *gateway) loadLevel(zoneName string) (*level.Level, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if l, ok := g.levels[zoneName]; ok {
		return l, nil
	}
	l, err := level.Load(zoneName)
	if err != nil {
		return nil, err
	}
	g.levels[zoneName] = l
	return l, nil
}

// getOrCreateZone returns the zone for the given ID, creating it if needed.
// The level defines all zone properties including type. Pass oflx for instanced
// zones with overflux conditions (nil for open-world zones or joining existing).
func (g *gateway) getOrCreateZone(zoneID string, lvl *level.Level, groupSize int, oflx *overflux.State) *zoneInstance {
	g.mu.Lock()
	defer g.mu.Unlock()
	zi, ok := g.zones[zoneID]
	if !ok {
		z := zone.New(zoneID, lvl, oflx)
		z.CombatLogSink = g.container.CombatLogSink
		ctx, cancel := context.WithCancel(context.Background())
		zi = &zoneInstance{zone: z, cancel: cancel, nextID: 1}
		g.zones[zoneID] = zi
		go z.Run(ctx)
		slog.Info("zone created", "zone_id", zoneID, "type", lvl.ZoneType, "group_size", groupSize)
	}
	return zi
}

// getZone returns an existing zone or nil.
func (g *gateway) getZone(zoneID string) *zoneInstance {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.zones[zoneID]
}

func (g *gateway) removeZone(zoneID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if zi, ok := g.zones[zoneID]; ok {
		zi.cancel()
		delete(g.zones, zoneID)
		slog.Info("zone removed", "zone_id", zoneID)
	}
}

// joinResponse controls the message sent to the player after joining a zone.
type joinResponse uint8

const (
	joinResponseZoneJoined   joinResponse = iota // OpZoneJoined  [peerID:2][0:1]
	joinResponseZoneTransfer                     // OpZoneTransfer [type:1][peerID:2]
)

// leaveZone removes a player from their current zone, broadcasts their
// departure, and cleans up empty arena zones.
func (g *gateway) leaveZone(sess *session.Session) {
	if sess.ZoneID == "" {
		return
	}
	zi := g.getZone(sess.ZoneID)
	if zi == nil {
		return
	}
	zi.zone.RemoveClient(sess.PeerID)
	disconnMsg := message.Encode(message.OpPeerDisconnected, 0, codec.EncodePeerID(sess.PeerID))
	zi.zone.Broadcast(disconnMsg, sess.PeerID)
	if zi.zone.Type == zone.ZoneTypeInstanced && zi.zone.ClientCount() == 0 {
		g.removeZone(sess.ZoneID)
	}
}

// joinZone adds a player to a zone, notifies peers, and sends the appropriate
// response. It handles peer ID allocation, display name resolution, position
// restore (hub zones), and class selection queuing. When returnFromZone is set
// (the player is leaving that dungeon for the hub), they spawn at the dungeon's
// hub entrance instead of their last-saved position.
func (g *gateway) joinZone(sess *session.Session, zi *zoneInstance, resp joinResponse, returnFromZone string) {
	// Allocate peer ID.
	zi.mu.Lock()
	peerID := zi.nextID
	zi.nextID++
	zi.mu.Unlock()

	sess.Mu.Lock()
	sess.PeerID = peerID
	sess.ZoneID = zi.zone.ID
	sess.ZoneType = uint8(zi.zone.Type)
	sess.Mu.Unlock()
	g.sessions.IndexZonePeer(sess.ID, zi.zone.ID, peerID)

	displayName := resolveDisplayName(sess)

	zi.zone.AddClient(&zone.Client{
		PeerID:   peerID,
		Username: displayName,
		Send:     sess.Conn.Send,
		SendUDP:  sess.Conn.SendUDP,
		HasUDP:   sess.Conn.HasUDP,
	})

	// Queue class selection for non-gunner characters.
	if sess.Class != "" && sess.Class != entity.ClassGunner {
		zi.zone.QueueInput(peerID, message.OpInteractInput,
			codec.EncodeInteractInput(message.InteractClassSelect, sess.Class))
	}

	// Queue spec selection if non-default.
	if sess.Spec != "" {
		zi.zone.QueueInput(peerID, message.OpInteractInput,
			codec.EncodeInteractInput(message.InteractSpecSelect, sess.Spec))
	}

	if entrancePos, ok := g.dungeonReturnEntrance(zi, returnFromZone); ok {
		zi.zone.SetPlayerPosition(peerID, entrancePos, zi.zone.SpawnYaw())
	} else {
		g.restoreSavedPosition(zi, peerID, sess)
	}

	notifyExistingPeers(sess, zi, peerID)
	sendJoinResponse(sess, zi, peerID, resp)

	// Send UDP association token if the client hasn't associated yet.
	// UDP survives zone transfers (same gateway socket), so only send once.
	if !sess.Conn.HasUDP() {
		g.sendUDPAssociate(sess)
	}

	// Broadcast new peer to existing peers.
	peerMsg := message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(peerID))
	zi.zone.Broadcast(peerMsg, peerID)

	// Load inventory and apply gear stats.
	if sess.CharID != 0 {
		g.loadAndApplyGear(sess, zi)
		g.loadAndSendLoadout(sess, zi)
	}

	// Send overflux state to players entering instanced zones.
	if oflx := zi.zone.OverfluxState(); oflx != nil && len(oflx.Conditions) > 0 {
		sess.Conn.Send(message.Encode(message.OpOverfluxState, 0, overflux.EncodeState(oflx)))
	}

	slog.Info("peer joined zone", "zone_id", zi.zone.ID, "peer_id", peerID, "username", displayName)
}

// sendUDPAssociate generates a token and sends OpUDPAssociate to the client.
func notifyExistingPeers(sess *session.Session, zi *zoneInstance, peerID uint16) {
	for _, existingID := range zi.zone.GetPeerIDs() {
		if existingID == peerID {
			continue
		}
		sess.Conn.Send(message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(existingID)))
	}
}

func sendJoinResponse(sess *session.Session, zi *zoneInstance, peerID uint16, resp joinResponse) {
	// Trailing [spawn_yaw][spawn_x][spawn_y][spawn_z], all f32 BE, lets the client
	// place the local player at the server-authoritative spawn without guessing.
	var spawn entity.Vec3
	if p := zi.zone.GetPlayer(peerID); p != nil {
		spawn = p.Position
	}
	appendSpawn := func(buf []byte) []byte {
		buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(zi.zone.SpawnYaw()))
		buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(spawn.X))
		buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(spawn.Y))
		buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(spawn.Z))
		return buf
	}
	switch resp {
	case joinResponseZoneJoined:
		buf := make([]byte, 3, 19)
		binary.BigEndian.PutUint16(buf[0:2], peerID)
		buf[2] = 0
		sess.Conn.Send(message.Encode(message.OpZoneJoined, 0, appendSpawn(buf)))
	case joinResponseZoneTransfer:
		buf := make([]byte, 3, 19)
		buf[0] = byte(zi.zone.Type)
		binary.BigEndian.PutUint16(buf[1:3], peerID)
		sess.Conn.Send(message.Encode(message.OpZoneTransfer, 0, appendSpawn(buf)))
	}
}

// No-op if the UDP server is not running.
func (g *gateway) sendUDPAssociate(sess *session.Session) {
	if g.udpServer == nil {
		return
	}
	token := g.udpServer.GenerateToken(sess.Conn, sess.ID)
	// [token:16][port:2 BE][hostLen:2 BE][host:hostLen]. The host tells the client
	// which address to dial for UDP. Empty (hostLen 0) means reuse the WS host,
	// correct when WS and UDP share an IP. A non-empty host is required when UDP is
	// exposed on a separate endpoint (e.g. a dedicated UDP LoadBalancer).
	host := []byte(g.udpPublicHost)
	port := uint16(g.udpServer.Port())
	if g.udpPublicPort != 0 {
		port = g.udpPublicPort
	}
	payload := make([]byte, 20+len(host))
	copy(payload[0:16], token[:])
	binary.BigEndian.PutUint16(payload[16:18], port)
	binary.BigEndian.PutUint16(payload[18:20], uint16(len(host)))
	copy(payload[20:], host)
	sess.Conn.Send(message.Encode(message.OpUDPAssociate, 0, payload))
}

// resolveDisplayName returns the best available display name for a session.
// It prefers CharName, then Username, and synthesises a name as a last resort,
// updating sess.Username so the synthetic name is stable for the connection.
func resolveDisplayName(sess *session.Session) string {
	if sess.CharName != "" {
		return sess.CharName
	}
	if sess.Username != "" {
		return sess.Username
	}
	name := fmt.Sprintf("Player_%d", sess.ID)
	sess.Username = name
	return name
}

// restoreSavedPosition restores a character's last saved position for hub
// (open-world) zones. It is a no-op for instanced zones or when the character
// has no saved position.
func (g *gateway) restoreSavedPosition(zi *zoneInstance, peerID uint16, sess *session.Session) {
	if zi.zone.Type != zone.ZoneTypeOpenWorld || sess.CharID == 0 {
		return
	}
	ch, _ := g.container.Repo.GetCharacterByID(sess.CharID)
	if ch == nil || (ch.PosX == 0 && ch.PosY == 0 && ch.PosZ == 0) {
		return
	}
	zi.zone.SetPlayerPosition(peerID, entity.Vec3{
		X: float32(ch.PosX),
		Y: float32(ch.PosY),
		Z: float32(ch.PosZ),
	}, float32(ch.RotY))
}

// loadAndApplyGear loads a character's inventory, computes gear stats, applies
// them to the zone player, and sends the inventory snapshot to the client.
func (g *gateway) loadAndApplyGear(sess *session.Session, zi *zoneInstance) {
	equipped, bag, err := g.inventory.LoadInventory(sess.CharID)
	if err != nil {
		slog.Error("load inventory", "char_id", sess.CharID, "error", err)
		return
	}

	stats := item.ComputeStats(equipped)
	zi.zone.SetPlayerGear(sess.PeerID, entity.GearStats(stats))

	// Current mercenary scrip balance (0 if it can't be loaded).
	scrip := 0
	if ms, mErr := g.progression.GetState(sess.CharID); mErr == nil && ms != nil {
		scrip = ms.ScripBalance
	}

	// Build and send OpInventoryState.
	sess.Conn.Send(message.Encode(message.OpInventoryState, 0,
		codec.EncodeInventoryState(
			buildInventoryInfos(equipped[:]),
			buildBagInfos(bag),
			stats,
			scrip,
		),
	))
}

// loadAndSendLoadout loads a character's ability loadout, sends the ability catalog
// and loadout state to the client, and applies the loadout in the zone.
// Only relevant for Arcanotechnicien characters.
func (g *gateway) loadAndSendLoadout(sess *session.Session, zi *zoneInstance) {
	// Only for Arcanotechnicien.
	if sess.Class != entity.ClassArcanotechnicien {
		return
	}

	// Load persisted loadout (or default).
	var slots [6]string
	if sess.CharID != 0 {
		loadout, err := g.container.Repo.GetLoadout(sess.CharID)
		if err != nil {
			slog.Error("load loadout", "char_id", sess.CharID, "error", err)
		}
		if loadout != nil {
			slots = [6]string{loadout.Slot0, loadout.Slot1, loadout.Slot2, loadout.Slot3, loadout.Slot4, loadout.Slot5}
		}
	}

	// If no persisted loadout, use default.
	hasLoadout := false
	for _, s := range slots {
		if s != "" {
			hasLoadout = true
			break
		}
	}
	if !hasLoadout {
		slots = entity.HarmonistDefaultLoadoutSlots()
	}

	// Send catalog with affinity info for the spec.
	if g.catalog != nil {
		specID := sess.Spec
		if specID == "" {
			specID = "harmonist"
		}
		entries := g.buildAbilityCatalogEntries(specID)
		slog.Info("sending ability catalog", "peer_id", sess.PeerID, "spec", specID, "count", len(entries))
		sess.Conn.Send(message.Encode(message.OpAbilityCatalog, 0, codec.EncodeAbilityCatalog(entries)))
	} else {
		slog.Warn("no ability catalog loaded, skipping catalog send", "peer_id", sess.PeerID)
	}

	// Send loadout state.
	sess.Conn.Send(message.Encode(message.OpLoadoutState, 0, codec.EncodeLoadoutState(slots)))

	// Send flux commitment state (load from DB or use default for Harmonist).
	if sess.Spec == "harmonist" || sess.Spec == "" {
		commitEntries := g.loadFluxCommitment(sess.CharID)
		sess.Conn.Send(message.Encode(message.OpFluxCommitState, 0, codec.EncodeFluxCommitState(commitEntries)))
		// Apply commitment to zone player.
		commitPayload := codec.EncodeFluxCommitState(commitEntries)
		zi.zone.QueueInput(sess.PeerID, message.OpSetFluxCommitment, commitPayload)
	}

	// Apply loadout to zone player.
	zi.zone.QueueInput(sess.PeerID, message.OpSetLoadout, codec.EncodeLoadoutState(slots))

	// Send loadout presets.
	g.sendPresetList(sess)
}

// buildAbilityCatalogEntries builds the codec entries for all abilities in the
// given spec, enriching each entry with runtime stats from the ability engine
// and applying flux-cost fallbacks when no AbilityDef is registered.
func (g *gateway) buildAbilityCatalogEntries(specID string) []codec.AbilityCatalogEntry {
	abilities := g.catalog.AbilitiesForSpec(specID)
	var entries []codec.AbilityCatalogEntry
	for _, sw := range abilities {
		entry := codec.AbilityCatalogEntry{
			ID:          sw.ID,
			Name:        sw.Name,
			School:      sw.School,
			AbilityType: sw.AbilityType,
			Delivery:    sw.Delivery,
			FluxCost:    sw.FluxCost,
			Description: sw.Description,
			Cooldown:    sw.Cooldown,
			CommitTime:  sw.CommitTime,
			Implemented: sw.Implemented,
			Affinity:    sw.Affinity,
		}
		if g.abilityEng != nil {
			if def := g.abilityEng.GetAbility(sw.ID); def != nil {
				for _, cost := range def.Costs {
					if cost.Resource == "flux" {
						entry.FluxAmount = cost.Amount
					}
				}
				entry.BaseHeal = def.BaseHeal
				entry.BaseDamage = def.BaseDamage
				entry.Range = def.Hit.Range
				entry.GCD = def.GCD
				entry.CommitTime = def.CommitTime
				entry.ZoneRadius = def.ZoneRadius
				entry.ZoneDuration = def.ZoneDuration
				entry.ZoneHealTick = def.ZoneHealTick
				entry.Sustain = def.Sustain
			}
		}
		// Fallback: estimate flux from categorical label when no AbilityDef exists.
		if entry.FluxAmount == 0 && entry.FluxCost != "" {
			switch entry.FluxCost {
			case "low":
				entry.FluxAmount = 5
			case "medium":
				entry.FluxAmount = 15
			case "high":
				entry.FluxAmount = 40
			case "extreme":
				entry.FluxAmount = 80
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

// loadFluxCommitment loads the persisted flux commitment for a character,
// falling back to the default Harmonist allocation when none is stored.
func (g *gateway) loadFluxCommitment(charID uint) []codec.FluxCommitEntry {
	var commitEntries []codec.FluxCommitEntry
	if charID != 0 {
		repoEntries, err := g.container.Repo.GetFluxCommitment(charID)
		if err != nil {
			slog.Error("load flux commitment", "char_id", charID, "error", err)
		}
		for _, e := range repoEntries {
			commitEntries = append(commitEntries, codec.FluxCommitEntry{School: e.School, Percentage: e.Percentage})
		}
	}
	if len(commitEntries) == 0 {
		commitEntries = []codec.FluxCommitEntry{
			{School: "bioarcanotechnic", Percentage: 60},
			{School: "biometabolic", Percentage: 40},
		}
	}
	return commitEntries
}

// buildInventoryInfos converts equipped items to codec-compatible structs.
func buildInventoryInfos(equipped []*item.Item) []codec.InventoryItemInfo {
	var out []codec.InventoryItemInfo
	for slot := range item.SlotCount {
		it := equipped[slot]
		if it == nil {
			continue
		}
		out = append(out, itemToCodec(it))
	}
	return out
}

// buildBagInfos converts bag items to codec-compatible structs.
func buildBagInfos(bag []*item.Item) []codec.InventoryItemInfo {
	out := make([]codec.InventoryItemInfo, len(bag))
	for i, it := range bag {
		out[i] = itemToCodec(it)
	}
	return out
}

// itemToCodec converts an item.Item to a codec.InventoryItemInfo.
func itemToCodec(it *item.Item) codec.InventoryItemInfo {
	def := item.DefRegistry[it.DefID]
	name := it.DefID
	if def != nil {
		name = def.Name
	}

	statLines := item.ComputeStatsForItem(it)
	codecLines := make([]codec.InventoryStatLine, len(statLines))
	for i, sl := range statLines {
		codecLines[i] = codec.InventoryStatLine{Stat: uint8(sl.Stat), Value: sl.Value}
	}

	return codec.InventoryItemInfo{
		ItemID:    uint32(it.ID),
		SlotID:    uint8(it.Slot),
		DefID:     it.DefID,
		Name:      name,
		ILvl:      uint16(it.ILvl),
		StatLines: codecLines,
	}
}

// transferPlayer moves a player from their current zone to a new zone.
// groupSize is used for instance scaling. Pass oflx for new instanced zones
// with overflux conditions (nil for open-world or joining existing). When the
// player is leaving a dungeon for the hub, returnFromZone is the source instance
// ID so joinZone can place them at that dungeon's hub entrance ("" otherwise).
func (g *gateway) transferPlayer(sess *session.Session, targetZoneID string, lvl *level.Level, groupSize int, oflx *overflux.State, returnFromZone string) {
	g.leaveZone(sess)

	zi := g.getOrCreateZone(targetZoneID, lvl, groupSize, oflx)
	if zi.zone.Type == zone.ZoneTypeInstanced {
		zi.zone.SetOnPlayerReturnToOpenWorld(func(peerID uint16) {
			g.handlePlayerReturnToOpenWorld(targetZoneID, peerID)
		})
		zi.zone.SetOnBossDefeated(func(peerIDs []uint16, overfluxScore int, overTime bool) {
			g.handleBossDefeated(targetZoneID, peerIDs, overfluxScore, overTime)
		})
	}

	g.joinZone(sess, zi, joinResponseZoneTransfer, returnFromZone)
}

// handleBossDefeated awards mercenary scrip to all players in the instance when the boss dies.
// An over-time finish (boss killed after the instance timer expired) pays reduced
// scrip and grants no watermark / tier-unlock progress.
func (g *gateway) handleBossDefeated(zoneID string, peerIDs []uint16, overfluxScore int, overTime bool) {
	for _, peerID := range peerIDs {
		globalID := g.sessions.ResolveZonePeer(zoneID, peerID)
		if globalID == 0 {
			continue
		}
		sess := g.sessions.GetByID(globalID)
		if sess == nil || sess.CharID == 0 {
			continue
		}
		amount, err := g.progression.AwardScrip(sess.CharID, overfluxScore, overTime)
		if err != nil {
			slog.Error("award scrip failed", "char_id", sess.CharID, "error", err)
			continue
		}
		slog.Info("scrip awarded", "char_id", sess.CharID, "amount", amount, "overflux", overfluxScore, "over_time", overTime)
		// Send award notification to the player.
		bal, _ := g.progression.GetState(sess.CharID)
		newBalance := 0
		if bal != nil {
			newBalance = bal.ScripBalance
		}
		sess.Conn.Send(message.Encode(message.OpScripAward, 0,
			codec.EncodeScripAward(amount, newBalance)))
	}
}

// handlePlayerReturnToOpenWorld transfers a single dead player back to the open-world zone.
func (g *gateway) handlePlayerReturnToOpenWorld(zoneID string, peerID uint16) {
	globalID := g.sessions.ResolveZonePeer(zoneID, peerID)
	if globalID == 0 {
		return
	}
	sess := g.sessions.GetByID(globalID)
	if sess == nil {
		return
	}
	slog.Info("player returning to open world", "player_id", sess.ID, "from_zone", zoneID)
	lvl, err := g.loadLevel(defaultOpenWorldZone)
	if err != nil {
		slog.Error("open-world level not found for return", "error", err)
		return
	}
	// A player leaving a dungeon appears at that dungeon's hub entrance rather
	// than their last-saved position; joinZone resolves this from the source zone.
	g.transferPlayer(sess, defaultOpenWorldZone, lvl, 0, nil, zoneID)

	grp := g.groups.GetGroup(sess.ID)
	if grp != nil {
		g.broadcastGroupState(grp)
	}
}

// dungeonReturnEntrance returns the hub-entrance spawn for a player leaving the
// dungeon returnFromZone, or ok=false when this is not a dungeon return into an
// open-world zone (the caller then restores the last-saved position instead).
func (g *gateway) dungeonReturnEntrance(zi *zoneInstance, returnFromZone string) (entity.Vec3, bool) {
	if returnFromZone == "" || zi.zone.Type != zone.ZoneTypeOpenWorld {
		return entity.Vec3{}, false
	}
	return hubEntrance(zi.zone.Portals(), returnFromZone)
}

// instanceBaseZone strips the instance suffix from an instanced zone ID, yielding
// the base zone name that matches a hub portal's TargetZone. Instance IDs are
// formed as "<base>_g<groupID>", "<base>_s<sessionID>", or "<base>_dev"
// (see handleEnterPortal). Base zone names contain no underscores.
func instanceBaseZone(zoneID string) string {
	for _, sep := range []string{"_g", "_s", "_dev"} {
		if i := strings.Index(zoneID, sep); i > 0 {
			return zoneID[:i]
		}
	}
	return zoneID
}

// hubEntrance returns the spawn position at the hub portal leading to
// dungeonZoneID. The player spawns directly on the portal pad (which the level
// author places on solid ground); re-entry needs an explicit interact, so the
// pad is safe to stand on. ok is false when no hub portal targets that dungeon.
func hubEntrance(portals []level.PortalDef, dungeonZoneID string) (entity.Vec3, bool) {
	base := instanceBaseZone(dungeonZoneID)
	for _, portal := range portals {
		if portal.TargetZone == base {
			return portal.Position, true
		}
	}
	return entity.Vec3{}, false
}

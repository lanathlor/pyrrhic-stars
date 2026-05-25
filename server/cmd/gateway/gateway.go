package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/abilitycatalog"
	"codex-online/server/internal/character"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/container"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/group"
	"codex-online/server/internal/inventory"
	"codex-online/server/internal/item"
	"codex-online/server/internal/message"
	"codex-online/server/internal/session"
	"codex-online/server/internal/user"
	"codex-online/server/internal/zone"
)

// gateway manages zones, player sessions, and groups.
type gateway struct {
	container  *container.Container
	zones      map[string]*zoneInstance
	sessions   *session.Registry
	groups     *group.Manager
	users      *user.Service
	characters *character.Service
	inventory  *inventory.Service
	catalog    *abilitycatalog.Catalog
	abilityEng *ability.Engine // for stat lookups when building catalog
	mu         sync.Mutex      // protects zones
	devMode    bool            // CODEX_DEV=1 enables debug features
}

type zoneInstance struct {
	zone     *zone.Zone
	zoneType zone.ZoneType
	cancel   context.CancelFunc
	nextID   uint16
	mu       sync.Mutex
}

func newGateway(ctr *container.Container) *gateway {
	return &gateway{
		container:  ctr,
		zones:      make(map[string]*zoneInstance),
		sessions:   session.NewRegistry(),
		groups:     group.NewManager(),
		users:      user.NewService(ctr.Repo),
		characters: character.NewService(ctr.Repo),
		inventory:  inventory.NewService(ctr.Repo),
	}
}

// getOrCreateZone returns the zone for the given ID, creating it if needed.
// Instance scaling is handled dynamically as players join/leave via AddClient/RemoveClient.
func (g *gateway) getOrCreateZone(zoneID string, zoneType zone.ZoneType, groupSize int) *zoneInstance {
	g.mu.Lock()
	defer g.mu.Unlock()
	zi, ok := g.zones[zoneID]
	if !ok {
		z := zone.New(zoneID, zoneType)
		z.CombatLogSink = g.container.CombatLogSink
		ctx, cancel := context.WithCancel(context.Background())
		zi = &zoneInstance{zone: z, zoneType: zoneType, cancel: cancel, nextID: 1}
		g.zones[zoneID] = zi
		go z.Run(ctx)
		slog.Info("zone created", "zone_id", zoneID, "type", zoneType, "group_size", groupSize)
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
	if zi.zoneType == zone.ZoneTypeInstanced && zi.zone.ClientCount() == 0 {
		g.removeZone(sess.ZoneID)
	}
}

// joinZone adds a player to a zone, notifies peers, and sends the appropriate
// response. It handles peer ID allocation, display name resolution, position
// restore (hub zones), and class selection queuing.
func (g *gateway) joinZone(sess *session.Session, zi *zoneInstance, resp joinResponse) {
	// Allocate peer ID.
	zi.mu.Lock()
	peerID := zi.nextID
	zi.nextID++
	zi.mu.Unlock()

	sess.PeerID = peerID
	sess.ZoneID = zi.zone.ID

	displayName := resolveDisplayName(sess)

	zi.zone.AddClient(&zone.Client{
		PeerID:   peerID,
		Username: displayName,
		Send:     sess.Conn.Send,
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

	g.restoreSavedPosition(zi, peerID, sess)

	// Notify new peer about existing peers.
	for _, existingID := range zi.zone.GetPeerIDs() {
		if existingID == peerID {
			continue
		}
		sess.Conn.Send(message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(existingID)))
	}

	// Send response to the joining client.
	switch resp {
	case joinResponseZoneJoined:
		buf := make([]byte, 3)
		binary.BigEndian.PutUint16(buf[0:2], peerID)
		buf[2] = 0
		sess.Conn.Send(message.Encode(message.OpZoneJoined, 0, buf))
	case joinResponseZoneTransfer:
		buf := make([]byte, 3)
		buf[0] = byte(zi.zoneType)
		binary.BigEndian.PutUint16(buf[1:3], peerID)
		sess.Conn.Send(message.Encode(message.OpZoneTransfer, 0, buf))
	}

	// Broadcast new peer to existing peers.
	peerMsg := message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(peerID))
	zi.zone.Broadcast(peerMsg, peerID)

	// Load inventory and apply gear stats.
	if sess.CharID != 0 {
		g.loadAndApplyGear(sess, zi)
		g.loadAndSendLoadout(sess, zi)
	}

	slog.Info("peer joined zone", "zone_id", zi.zone.ID, "peer_id", peerID, "username", displayName)
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
	if zi.zoneType != zone.ZoneTypeOpenWorld || sess.CharID == 0 {
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

	// Build and send OpInventoryState.
	sess.Conn.Send(message.Encode(message.OpInventoryState, 0,
		codec.EncodeInventoryState(
			buildInventoryInfos(equipped[:]),
			buildBagInfos(bag),
			stats,
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
		slots = [6]string{"mending_surge", "mending_beam", "vital_bloom", "restoration_matrix", "life_swap", "transfusion"}
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
			{School: "bioarcanotechnic", Percentage: 50},
			{School: "biometabolic", Percentage: 30},
			{School: "frost", Percentage: 10},
			{School: "aerokinetic", Percentage: 10},
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
// groupSize is used for instance scaling when creating a new arena.
func (g *gateway) transferPlayer(sess *session.Session, targetZoneID string, targetType zone.ZoneType, groupSize int) {
	g.leaveZone(sess)

	zi := g.getOrCreateZone(targetZoneID, targetType, groupSize)
	if targetType == zone.ZoneTypeInstanced {
		zi.zone.OnPlayerRespawnHub = func(peerID uint16) {
			g.handlePlayerRespawnHub(targetZoneID, peerID)
		}
	}

	g.joinZone(sess, zi, joinResponseZoneTransfer)
}

// handlePlayerRespawnHub transfers a single dead player back to the hub.
func (g *gateway) handlePlayerRespawnHub(zoneID string, peerID uint16) {
	globalID := g.sessions.ResolveZonePeer(zoneID, peerID)
	if globalID == 0 {
		return
	}
	sess := g.sessions.GetByID(globalID)
	if sess == nil {
		return
	}
	slog.Info("player respawning to hub", "player_id", sess.ID, "from_zone", zoneID)
	g.transferPlayer(sess, zone.ZoneHub, zone.ZoneTypeOpenWorld, 0)

	grp := g.groups.GetGroup(sess.ID)
	if grp != nil {
		g.broadcastGroupState(grp)
	}
}

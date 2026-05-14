package inventory

import (
	"fmt"

	"codex-online/server/internal/item"
	"codex-online/server/internal/persistence"
)

// Service handles inventory and equipment operations for characters.
type Service struct {
	repo persistence.Repository
}

// NewService creates an inventory service.
func NewService(repo persistence.Repository) *Service {
	return &Service{repo: repo}
}

// LoadInventory returns equipped items (by slot) and bag items for a character.
func (s *Service) LoadInventory(charID uint) (equipped [item.SlotCount]*item.Item, bag []*item.Item, err error) {
	allItems, err := s.repo.GetItemsByCharacterID(charID)
	if err != nil {
		return equipped, nil, fmt.Errorf("load items: %w", err)
	}

	eqRows, err := s.repo.GetEquipment(charID)
	if err != nil {
		return equipped, nil, fmt.Errorf("load equipment: %w", err)
	}

	// Build a set of equipped item IDs.
	equippedIDs := make(map[uint]uint8, len(eqRows)) // itemID → slotID
	for _, eq := range eqRows {
		equippedIDs[eq.ItemID] = eq.SlotID
	}

	// Build item map for lookup.
	itemMap := make(map[uint]*persistence.CharacterItem, len(allItems))
	for _, ci := range allItems {
		itemMap[ci.ID] = ci
	}

	// Fill equipped slots.
	for _, eq := range eqRows {
		ci, ok := itemMap[eq.ItemID]
		if !ok {
			continue
		}
		slot := item.SlotID(eq.SlotID)
		if slot < item.SlotCount {
			equipped[slot] = &item.Item{
				ID:    ci.ID,
				DefID: ci.DefID,
				ILvl:  ci.ILvl,
				Slot:  item.SlotID(ci.Slot),
			}
		}
	}

	// Everything not equipped goes into the bag.
	for _, ci := range allItems {
		if _, isEquipped := equippedIDs[ci.ID]; isEquipped {
			continue
		}
		bag = append(bag, &item.Item{
			ID:    ci.ID,
			DefID: ci.DefID,
			ILvl:  ci.ILvl,
			Slot:  item.SlotID(ci.Slot),
		})
	}

	return equipped, bag, nil
}

// Equip moves an item into an equipment slot. If the slot is occupied,
// the existing item is returned to the bag.
func (s *Service) Equip(charID uint, itemID uint, slotID item.SlotID) error {
	if slotID >= item.SlotCount {
		return fmt.Errorf("equip: invalid slot %d", slotID)
	}

	// Verify item exists and belongs to this character.
	allItems, err := s.repo.GetItemsByCharacterID(charID)
	if err != nil {
		return fmt.Errorf("equip: load items: %w", err)
	}
	var target *persistence.CharacterItem
	for _, ci := range allItems {
		if ci.ID == itemID {
			target = ci
			break
		}
	}
	if target == nil {
		return fmt.Errorf("equip: item %d not found for character %d", itemID, charID)
	}

	// Verify item fits this slot.
	if item.SlotID(target.Slot) != slotID {
		return fmt.Errorf("equip: item %q (slot %d) cannot go in slot %d", target.DefID, target.Slot, slotID)
	}

	// Upsert equipment row (replaces any existing item in that slot).
	if err := s.repo.SetEquipment(charID, uint8(slotID), itemID); err != nil {
		return fmt.Errorf("equip: set equipment: %w", err)
	}

	return nil
}

// Unequip removes the item from an equipment slot (returns it to bag).
func (s *Service) Unequip(charID uint, slotID item.SlotID) error {
	if slotID >= item.SlotCount {
		return fmt.Errorf("unequip: invalid slot %d", slotID)
	}
	if err := s.repo.ClearEquipment(charID, uint8(slotID)); err != nil {
		return fmt.Errorf("unequip: clear equipment: %w", err)
	}
	return nil
}

// starterDefs defines which item def + ilvl goes in each slot at character creation.
var starterEquipped = []struct {
	DefID string
	ILvl  int
	Slot  item.SlotID
}{
	{"frame_basic", 1, item.SlotFrame},
	{"core_basic", 1, item.SlotPowerCore},
	{"weapon_basic", 1, item.SlotPrimaryWeapon},
	{"tool_basic", 1, item.SlotSecondaryTool},
	{"augment_basic", 1, item.SlotAugment},
	{"module_basic", 1, item.SlotModule},
}

// starterBag defines extra items placed in the bag at character creation.
var starterBag = []struct {
	DefID string
	ILvl  int
	Slot  item.SlotID
}{
	{"frame_reinforced", 2, item.SlotFrame},
	{"weapon_sharpened", 2, item.SlotPrimaryWeapon},
	{"module_combat", 3, item.SlotModule},
}

// SpawnStarterGear creates starter equipment and bag items for a new character.
func (s *Service) SpawnStarterGear(charID uint) error {
	// Create and equip starter items.
	for _, def := range starterEquipped {
		ci := &persistence.CharacterItem{
			CharacterID: charID,
			DefID:       def.DefID,
			ILvl:        def.ILvl,
			Slot:        uint8(def.Slot),
		}
		if err := s.repo.CreateItem(ci); err != nil {
			return fmt.Errorf("spawn starter: create item %q: %w", def.DefID, err)
		}
		if err := s.repo.SetEquipment(charID, uint8(def.Slot), ci.ID); err != nil {
			return fmt.Errorf("spawn starter: equip %q: %w", def.DefID, err)
		}
	}

	// Create bag items (not equipped).
	for _, def := range starterBag {
		ci := &persistence.CharacterItem{
			CharacterID: charID,
			DefID:       def.DefID,
			ILvl:        def.ILvl,
			Slot:        uint8(def.Slot),
		}
		if err := s.repo.CreateItem(ci); err != nil {
			return fmt.Errorf("spawn starter: create bag item %q: %w", def.DefID, err)
		}
	}

	return nil
}

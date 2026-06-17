package merchant

import (
	"fmt"
	"slices"

	"codex-online/server/internal/item"
	"codex-online/server/internal/overflux"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/progression"
)

// Service handles merchant purchases: spending scrip and gating the shop by the
// player's tier unlocks. Run-completion rewards live in the progression package.
type Service struct {
	repo persistence.Repository
}

// NewService creates a merchant service.
func NewService(repo persistence.Repository) *Service {
	return &Service{repo: repo}
}

// BuyItem purchases an item from a merchant tier. Validates tier unlock,
// sufficient scrip, and valid item def. Creates the item in the character's bag.
// Returns the new item ID and remaining scrip balance.
func (s *Service) BuyItem(charID uint, tier int, defID string) (itemID uint, newBalance int, err error) {
	// Validate tier exists.
	if tier < 0 || tier >= len(Tiers) {
		return 0, 0, fmt.Errorf("invalid tier %d", tier)
	}
	td := Tiers[tier]

	// Validate item is sold by merchants.
	if !slices.Contains(MerchantItems, defID) {
		return 0, 0, fmt.Errorf("item %q not sold by merchants", defID)
	}

	// Validate item def exists in registry.
	def := item.DefRegistry[defID]
	if def == nil {
		return 0, 0, fmt.Errorf("item def %q not found", defID)
	}

	// Check tier unlock.
	maxScore := overflux.MaxScore()
	best, err := s.repo.GetWatermark(charID, progression.CurrentSeason)
	if err != nil {
		return 0, 0, fmt.Errorf("get watermark: %w", err)
	}
	if !IsTierUnlocked(tier, best, maxScore) {
		return 0, 0, fmt.Errorf("tier %d locked (need %d%% of %d, have %d)", tier, td.UnlockPercent, maxScore, best)
	}

	// Check the player can afford it, with a clean user-facing message before
	// touching the scrip ledger (DeductScrip's errors are not display-ready).
	balance, err := s.repo.GetScrip(charID, progression.CurrentSeason)
	if err != nil {
		return 0, 0, fmt.Errorf("get scrip: %w", err)
	}
	if balance < td.Price {
		return 0, 0, fmt.Errorf("not enough scrip: need %d, have %d", td.Price, balance)
	}

	// Deduct scrip.
	if err := s.repo.DeductScrip(charID, progression.CurrentSeason, td.Price); err != nil {
		return 0, 0, err
	}

	// Create the item at the tier's ilvl.
	ci := &persistence.CharacterItem{
		CharacterID: charID,
		DefID:       defID,
		ILvl:        td.ILvl,
		Slot:        uint8(def.Slot),
	}
	if err := s.repo.CreateItem(ci); err != nil {
		// Attempt to refund scrip on failure.
		_ = s.repo.AddScrip(charID, progression.CurrentSeason, td.Price)
		return 0, 0, fmt.Errorf("create item: %w", err)
	}

	// Get updated balance.
	bal, _ := s.repo.GetScrip(charID, progression.CurrentSeason)
	return ci.ID, bal, nil
}

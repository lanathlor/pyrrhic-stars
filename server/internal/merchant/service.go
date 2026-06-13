package merchant

import (
	"fmt"
	"slices"

	"codex-online/server/internal/item"
	"codex-online/server/internal/overflux"
	"codex-online/server/internal/persistence"
)

// Service handles mercenary scrip rewards and merchant purchases.
type Service struct {
	repo persistence.Repository
}

// NewService creates a merchant service.
func NewService(repo persistence.Repository) *Service {
	return &Service{repo: repo}
}

// PlayerState holds the scrip/unlock state for a character in the current season.
type PlayerState struct {
	ScripBalance int
	BestScore    int
	Season       uint16
	MaxScore     int // theoretical max overflux score
}

// GetState returns the player's current scrip balance and watermark for the current season.
func (s *Service) GetState(charID uint) (*PlayerState, error) {
	balance, err := s.repo.GetScrip(charID, CurrentSeason)
	if err != nil {
		return nil, fmt.Errorf("get scrip: %w", err)
	}
	best, err := s.repo.GetWatermark(charID, CurrentSeason)
	if err != nil {
		return nil, fmt.Errorf("get watermark: %w", err)
	}
	return &PlayerState{
		ScripBalance: balance,
		BestScore:    best,
		Season:       CurrentSeason,
		MaxScore:     overflux.MaxScore(),
	}, nil
}

// AwardScrip grants scrip to a character based on the overflux score of a completed run.
// On a timed clear (overTime=false) it also updates the watermark if this score
// is a new personal best. An over-time finish is not a "clear": scrip is cut to
// 1/OverTimePenaltyDivisor and the watermark is left untouched, so it grants no
// tier-unlock progress.
// Returns the amount of scrip awarded.
//
//nolint:revive // overTime reflects the run's outcome (a clear vs an over-time finish), not a control toggle the caller flips
func (s *Service) AwardScrip(charID uint, overfluxScore int, overTime bool) (int, error) {
	maxScore := overflux.MaxScore()
	amount := ScripReward(overfluxScore, maxScore)
	if overTime {
		amount /= OverTimePenaltyDivisor
	}

	if err := s.repo.AddScrip(charID, CurrentSeason, amount); err != nil {
		return 0, fmt.Errorf("add scrip: %w", err)
	}
	if !overTime {
		if err := s.repo.UpdateWatermark(charID, CurrentSeason, overfluxScore); err != nil {
			return 0, fmt.Errorf("update watermark: %w", err)
		}
	}
	return amount, nil
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
	best, err := s.repo.GetWatermark(charID, CurrentSeason)
	if err != nil {
		return 0, 0, fmt.Errorf("get watermark: %w", err)
	}
	if !IsTierUnlocked(tier, best, maxScore) {
		return 0, 0, fmt.Errorf("tier %d locked (need %d%% of %d, have %d)", tier, td.UnlockPercent, maxScore, best)
	}

	// Check the player can afford it, with a clean user-facing message before
	// touching the scrip ledger (DeductScrip's errors are not display-ready).
	balance, err := s.repo.GetScrip(charID, CurrentSeason)
	if err != nil {
		return 0, 0, fmt.Errorf("get scrip: %w", err)
	}
	if balance < td.Price {
		return 0, 0, fmt.Errorf("not enough scrip: need %d, have %d", td.Price, balance)
	}

	// Deduct scrip.
	if err := s.repo.DeductScrip(charID, CurrentSeason, td.Price); err != nil {
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
		_ = s.repo.AddScrip(charID, CurrentSeason, td.Price)
		return 0, 0, fmt.Errorf("create item: %w", err)
	}

	// Get updated balance.
	bal, _ := s.repo.GetScrip(charID, CurrentSeason)
	return ci.ID, bal, nil
}

package progression

import (
	"fmt"

	"codex-online/server/internal/overflux"
	"codex-online/server/internal/persistence"
)

// Service grants run-completion rewards and reads the player's season economy
// snapshot. It shares the persistence-backed scrip/watermark ledger with the
// merchant package but owns the award logic.
type Service struct {
	repo persistence.Repository
}

// NewService creates a progression service.
func NewService(repo persistence.Repository) *Service {
	return &Service{repo: repo}
}

// PlayerState holds the scrip/watermark state for a character in the current season.
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

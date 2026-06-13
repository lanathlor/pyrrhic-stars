package merchant

// CurrentSeason is the active season identifier used for watermark scoping.
const CurrentSeason uint16 = 1

// TierDef describes one merchant tier: its item level, scrip cost, and the
// minimum overflux score percentage required to unlock it.
type TierDef struct {
	Tier          int
	ILvl          int
	Price         int
	UnlockPercent int // 0-100, percentage of MaxScore required
}

// Tiers lists all four merchant tiers in ascending order.
var Tiers = []TierDef{
	{Tier: 0, ILvl: 20, Price: 50, UnlockPercent: 0},
	{Tier: 1, ILvl: 30, Price: 100, UnlockPercent: 20},
	{Tier: 2, ILvl: 40, Price: 200, UnlockPercent: 45},
	{Tier: 3, ILvl: 50, Price: 400, UnlockPercent: 80},
}

// MerchantItems are the def IDs sold at every tier (same base items, different ilvl).
var MerchantItems = []string{
	"frame_basic",
	"core_basic",
	"weapon_basic",
	"tool_basic",
	"augment_basic",
	"module_basic",
}

// OverTimePenaltyDivisor is applied to the scrip reward when the boss is
// defeated after the instance time limit expired (Mythic+-style: an over-time
// finish is not a "clear"). The player keeps 1/OverTimePenaltyDivisor of the
// scrip and earns no watermark / tier-unlock progress.
const OverTimePenaltyDivisor = 10

// ScripReward returns the integer scrip earned for a single run.
// Base reward is 100 scrip; a bonus of up to 300 scales linearly with
// score/maxScore. Result is clamped to [100, 400].
// If maxScore is 0 the base reward of 100 is returned.
func ScripReward(score, maxScore int) int {
	if maxScore <= 0 {
		return 100
	}
	reward := 100 + int(300*float64(score)/float64(maxScore))
	if reward < 100 {
		return 100
	}
	if reward > 400 {
		return 400
	}
	return reward
}

// IsTierUnlocked reports whether the player's best overflux watermark meets
// the threshold required for the given tier index.
// Tier 0 always returns true.
func IsTierUnlocked(tier int, bestScore, maxScore int) bool {
	if tier < 0 || tier >= len(Tiers) {
		return false
	}
	t := Tiers[tier]
	if t.UnlockPercent == 0 {
		return true
	}
	return bestScore >= maxScore*t.UnlockPercent/100
}

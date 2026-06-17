// Package progression owns run-completion rewards and the per-character season
// economy snapshot (scrip balance + overflux watermark). It answers "what does
// finishing a run grant", as distinct from the merchant package which spends
// scrip and gates the shop.
package progression

// CurrentSeason is the active season identifier used for watermark scoping.
const CurrentSeason uint16 = 1

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

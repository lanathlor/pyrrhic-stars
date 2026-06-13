package merchant

import (
	"fmt"
	"strings"
	"testing"

	"codex-online/server/internal/item"
	"codex-online/server/internal/overflux"
	"codex-online/server/internal/persistence"
)

// testRepo is an in-memory stub that satisfies persistence.Repository.
// Only the methods called by Service are implemented; all others are no-ops.
type testRepo struct {
	scrip     map[uint]map[uint16]int // charID -> season -> balance
	watermark map[uint]map[uint16]int // charID -> season -> bestScore
	items     []*persistence.CharacterItem
	nextID    uint
}

func newTestRepo() *testRepo {
	return &testRepo{
		scrip:     make(map[uint]map[uint16]int),
		watermark: make(map[uint]map[uint16]int),
		nextID:    1,
	}
}

func (r *testRepo) GetScrip(charID uint, season uint16) (int, error) {
	if s, ok := r.scrip[charID]; ok {
		return s[season], nil
	}
	return 0, nil
}

func (r *testRepo) AddScrip(charID uint, season uint16, amount int) error {
	if r.scrip[charID] == nil {
		r.scrip[charID] = make(map[uint16]int)
	}
	r.scrip[charID][season] += amount
	return nil
}

func (r *testRepo) DeductScrip(charID uint, season uint16, amount int) error {
	if r.scrip[charID] == nil {
		return fmt.Errorf("deduct scrip: no scrip record for character %d season %d", charID, season)
	}
	bal := r.scrip[charID][season]
	if bal < amount {
		return fmt.Errorf("deduct scrip: insufficient balance (%d < %d)", bal, amount)
	}
	r.scrip[charID][season] = bal - amount
	return nil
}

func (r *testRepo) GetWatermark(charID uint, season uint16) (int, error) {
	if w, ok := r.watermark[charID]; ok {
		return w[season], nil
	}
	return 0, nil
}

func (r *testRepo) UpdateWatermark(charID uint, season uint16, score int) error {
	if r.watermark[charID] == nil {
		r.watermark[charID] = make(map[uint16]int)
	}
	if score > r.watermark[charID][season] {
		r.watermark[charID][season] = score
	}
	return nil
}

func (r *testRepo) CreateItem(ci *persistence.CharacterItem) error {
	ci.ID = r.nextID
	r.nextID++
	dup := *ci
	r.items = append(r.items, &dup)
	return nil
}

// Unimplemented stubs - return zero values and nil errors.
func (r *testRepo) UpsertUser(_, _ string) error                   { return nil }
func (r *testRepo) GetUser(_ string) (*persistence.User, error)    { return nil, nil }
func (r *testRepo) CreateCharacter(_ *persistence.Character) error { return nil }
func (r *testRepo) UpdateCharacterPosition(_ uint, _, _, _, _ float64) error {
	return nil
}
func (r *testRepo) UpdateCharacterSpec(_ uint, _ string) error               { return nil }
func (r *testRepo) GetCharacterByID(_ uint) (*persistence.Character, error)  { return nil, nil }
func (r *testRepo) GetCharacters(_ string) ([]*persistence.Character, error) { return nil, nil }
func (r *testRepo) IsCharacterNameTaken(_ string) (bool, error)              { return false, nil }
func (r *testRepo) CountCharacters(_ string) (int64, error)                  { return 0, nil }
func (r *testRepo) DeleteItem(_ uint) error                                  { return nil }
func (r *testRepo) GetItemsByCharacterID(_ uint) ([]*persistence.CharacterItem, error) {
	return nil, nil
}
func (r *testRepo) SetEquipment(_ uint, _ uint8, _ uint) error { return nil }
func (r *testRepo) ClearEquipment(_ uint, _ uint8) error       { return nil }
func (r *testRepo) GetEquipment(_ uint) ([]*persistence.CharacterEquipment, error) {
	return nil, nil
}
func (r *testRepo) UpsertLoadout(_ uint, _ [6]string) error { return nil }
func (r *testRepo) GetLoadout(_ uint) (*persistence.CharacterLoadout, error) {
	return nil, nil
}
func (r *testRepo) UpsertFluxCommitment(_ uint, _ []persistence.FluxCommitmentEntry) error {
	return nil
}
func (r *testRepo) GetFluxCommitment(_ uint) ([]persistence.FluxCommitmentEntry, error) {
	return nil, nil
}
func (r *testRepo) SaveLoadoutPreset(_ uint, _ string, _ [6]string, _ string) error {
	return nil
}
func (r *testRepo) DeleteLoadoutPreset(_ uint, _ uint) error { return nil }
func (r *testRepo) GetLoadoutPresets(_ uint) ([]*persistence.CharacterLoadoutPreset, error) {
	return nil, nil
}

// seedDefRegistry populates item.DefRegistry with the six merchant items so
// tests do not depend on YAML files being present at a relative path.
func seedDefRegistry() {
	item.DefRegistry["frame_basic"] = &item.ItemDef{ID: "frame_basic", Name: "Standard Frame", Slot: item.SlotFrame}
	item.DefRegistry["core_basic"] = &item.ItemDef{ID: "core_basic", Name: "Standard Power Core", Slot: item.SlotPowerCore}
	item.DefRegistry["weapon_basic"] = &item.ItemDef{ID: "weapon_basic", Name: "Standard Weapon", Slot: item.SlotPrimaryWeapon}
	item.DefRegistry["tool_basic"] = &item.ItemDef{ID: "tool_basic", Name: "Standard Sidearm", Slot: item.SlotSecondaryTool}
	item.DefRegistry["augment_basic"] = &item.ItemDef{ID: "augment_basic", Name: "Standard Augment", Slot: item.SlotAugment}
	item.DefRegistry["module_basic"] = &item.ItemDef{ID: "module_basic", Name: "Standard Module", Slot: item.SlotModule}
}

func TestMain(m *testing.M) {
	seedDefRegistry()
	m.Run()
}

// --- GetState ---

func TestGetState_FreshCharacter(t *testing.T) {
	svc := NewService(newTestRepo())
	state, err := svc.GetState(42)
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if state.ScripBalance != 0 {
		t.Errorf("ScripBalance = %d, want 0", state.ScripBalance)
	}
	if state.BestScore != 0 {
		t.Errorf("BestScore = %d, want 0", state.BestScore)
	}
	if state.Season != CurrentSeason {
		t.Errorf("Season = %d, want %d", state.Season, CurrentSeason)
	}
	if state.MaxScore != overflux.MaxScore() {
		t.Errorf("MaxScore = %d, want %d", state.MaxScore, overflux.MaxScore())
	}
}

func TestGetState_AfterAward(t *testing.T) {
	repo := newTestRepo()
	svc := NewService(repo)
	score := overflux.MaxScore() / 2 // half the max score

	awarded, err := svc.AwardScrip(7, score, false)
	if err != nil {
		t.Fatalf("AwardScrip error: %v", err)
	}

	state, err := svc.GetState(7)
	if err != nil {
		t.Fatalf("GetState error: %v", err)
	}
	if state.ScripBalance != awarded {
		t.Errorf("ScripBalance = %d, want %d", state.ScripBalance, awarded)
	}
	if state.BestScore != score {
		t.Errorf("BestScore = %d, want %d", state.BestScore, score)
	}
}

// --- AwardScrip ---

func TestAwardScrip_CorrectAmount(t *testing.T) {
	maxScore := overflux.MaxScore()
	tests := []struct {
		name  string
		score int
	}{
		{"zero score", 0},
		{"half score", maxScore / 2},
		{"full score", maxScore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(newTestRepo())
			got, err := svc.AwardScrip(1, tt.score, false)
			if err != nil {
				t.Fatalf("AwardScrip error: %v", err)
			}
			want := ScripReward(tt.score, maxScore)
			if got != want {
				t.Errorf("AwardScrip(%d) = %d, want %d", tt.score, got, want)
			}
		})
	}
}

func TestAwardScrip_UpdatesWatermark(t *testing.T) {
	repo := newTestRepo()
	svc := NewService(repo)
	score := 60

	if _, err := svc.AwardScrip(3, score, false); err != nil {
		t.Fatalf("AwardScrip error: %v", err)
	}

	wm, _ := repo.GetWatermark(3, CurrentSeason)
	if wm != score {
		t.Errorf("watermark = %d, want %d", wm, score)
	}
}

func TestAwardScrip_AccumulatesBalance(t *testing.T) {
	svc := NewService(newTestRepo())
	maxScore := overflux.MaxScore()

	first, _ := svc.AwardScrip(5, 0, false)         // base reward
	second, _ := svc.AwardScrip(5, maxScore, false) // max reward

	state, err := svc.GetState(5)
	if err != nil {
		t.Fatalf("GetState error: %v", err)
	}
	want := first + second
	if state.ScripBalance != want {
		t.Errorf("balance = %d, want %d (first=%d + second=%d)", state.ScripBalance, want, first, second)
	}
}

func TestAwardScrip_WatermarkOnlyRaisesHigher(t *testing.T) {
	repo := newTestRepo()
	svc := NewService(repo)

	if _, err := svc.AwardScrip(9, 80, false); err != nil {
		t.Fatalf("first AwardScrip error: %v", err)
	}
	if _, err := svc.AwardScrip(9, 40, false); err != nil {
		t.Fatalf("second AwardScrip error: %v", err)
	}

	wm, _ := repo.GetWatermark(9, CurrentSeason)
	if wm != 80 {
		t.Errorf("watermark = %d, want 80 (lower second award must not overwrite)", wm)
	}
}

func TestAwardScrip_OverTimePenalty(t *testing.T) {
	repo := newTestRepo()
	svc := NewService(repo)
	maxScore := overflux.MaxScore()
	score := maxScore // full reward = 400 before penalty

	got, err := svc.AwardScrip(1, score, true)
	if err != nil {
		t.Fatalf("AwardScrip error: %v", err)
	}
	want := ScripReward(score, maxScore) / OverTimePenaltyDivisor
	if got != want {
		t.Errorf("over-time AwardScrip = %d, want %d (1/%d of full)", got, want, OverTimePenaltyDivisor)
	}
}

func TestAwardScrip_OverTimeSkipsWatermark(t *testing.T) {
	repo := newTestRepo()
	svc := NewService(repo)

	if _, err := svc.AwardScrip(2, 75, true); err != nil {
		t.Fatalf("AwardScrip error: %v", err)
	}

	wm, _ := repo.GetWatermark(2, CurrentSeason)
	if wm != 0 {
		t.Errorf("watermark = %d, want 0 (over-time finish is not a clear, must not improve watermark)", wm)
	}
}

// --- BuyItem ---

// buySetup seeds a character with enough scrip and watermark to unlock the
// requested tier, then returns a ready service backed by that repo.
func buySetup(t *testing.T, charID uint, tier int) (*Service, *testRepo) {
	t.Helper()
	repo := newTestRepo()

	// Grant sufficient watermark to unlock the tier.
	maxScore := overflux.MaxScore()
	if tier < len(Tiers) {
		needed := maxScore * Tiers[tier].UnlockPercent / 100
		if needed > 0 {
			// Bump one point above the threshold to ensure unlock.
			repo.watermark[charID] = map[uint16]int{CurrentSeason: needed + 1}
		}
	}

	// Grant enough scrip for the tier price.
	if tier < len(Tiers) {
		repo.scrip[charID] = map[uint16]int{CurrentSeason: Tiers[tier].Price * 2}
	}

	return NewService(repo), repo
}

func TestBuyItem_Success(t *testing.T) {
	const charID uint = 10
	const tier = 0
	const defID = "weapon_basic"
	svc, repo := buySetup(t, charID, tier)

	startBal, _ := repo.GetScrip(charID, CurrentSeason)
	itemID, newBal, err := svc.BuyItem(charID, tier, defID)
	if err != nil {
		t.Fatalf("BuyItem error: %v", err)
	}
	if itemID == 0 {
		t.Error("itemID should be non-zero")
	}
	wantBal := startBal - Tiers[tier].Price
	if newBal != wantBal {
		t.Errorf("newBalance = %d, want %d", newBal, wantBal)
	}

	// Verify the created item has the correct ilvl and slot.
	if len(repo.items) != 1 {
		t.Fatalf("expected 1 item in repo, got %d", len(repo.items))
	}
	ci := repo.items[0]
	if ci.ILvl != Tiers[tier].ILvl {
		t.Errorf("item ILvl = %d, want %d", ci.ILvl, Tiers[tier].ILvl)
	}
	if ci.Slot != uint8(item.SlotPrimaryWeapon) {
		t.Errorf("item Slot = %d, want %d (PrimaryWeapon)", ci.Slot, item.SlotPrimaryWeapon)
	}
	if ci.DefID != defID {
		t.Errorf("item DefID = %q, want %q", ci.DefID, defID)
	}
}

func TestBuyItem_BalanceDecreasedByTierPrice(t *testing.T) {
	const charID uint = 11
	for tier := 0; tier < len(Tiers); tier++ {
		t.Run(fmt.Sprintf("tier%d", tier), func(t *testing.T) {
			svc, repo := buySetup(t, charID, tier)
			startBal, _ := repo.GetScrip(charID, CurrentSeason)
			_, newBal, err := svc.BuyItem(charID, tier, "frame_basic")
			if err != nil {
				t.Fatalf("BuyItem tier %d error: %v", tier, err)
			}
			want := startBal - Tiers[tier].Price
			if newBal != want {
				t.Errorf("tier %d: newBalance = %d, want %d", tier, newBal, want)
			}
		})
	}
}

func TestBuyItem_ErrorInvalidTier(t *testing.T) {
	svc := NewService(newTestRepo())
	for _, bad := range []int{-1, len(Tiers), 99} {
		_, _, err := svc.BuyItem(1, bad, "frame_basic")
		if err == nil {
			t.Errorf("tier %d: expected error, got nil", bad)
		}
	}
}

func TestBuyItem_ErrorTierLocked(t *testing.T) {
	repo := newTestRepo()
	svc := NewService(repo)
	const charID uint = 20
	// No watermark set - character is fresh, tier 1 requires 20% of MaxScore.
	_, _, err := svc.BuyItem(charID, 1, "frame_basic")
	if err == nil {
		t.Fatal("expected error for locked tier, got nil")
	}
}

func TestBuyItem_ErrorInsufficientScrip(t *testing.T) {
	const charID uint = 30
	const tier = 0
	repo := newTestRepo()
	svc := NewService(repo)
	// Tier 0 is always unlocked (UnlockPercent = 0), but we set scrip below its price.
	repo.scrip[charID] = map[uint16]int{CurrentSeason: Tiers[tier].Price - 1}
	_, _, err := svc.BuyItem(charID, tier, "frame_basic")
	if err == nil {
		t.Fatal("expected insufficient scrip error, got nil")
	}
	// The message must be display-ready, not a leaked DB-layer string.
	if !strings.Contains(err.Error(), "not enough scrip") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not enough scrip")
	}
}

func TestBuyItem_FreshCharacterGetsCleanError(t *testing.T) {
	const charID uint = 31
	const tier = 0
	repo := newTestRepo()
	svc := NewService(repo)
	// No scrip record at all (fresh character). Previously this leaked the raw
	// "deduct scrip: no scrip record..." error to the player.
	_, _, err := svc.BuyItem(charID, tier, "frame_basic")
	if err == nil {
		t.Fatal("expected error buying with no scrip, got nil")
	}
	if strings.Contains(err.Error(), "no scrip record") {
		t.Errorf("error leaked DB-layer message: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "not enough scrip") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not enough scrip")
	}
}

func TestBuyItem_ErrorInvalidDefID(t *testing.T) {
	const charID uint = 40
	const tier = 0
	repo := newTestRepo()
	svc := NewService(repo)
	repo.scrip[charID] = map[uint16]int{CurrentSeason: 9999}

	// "widget_unknown" is not in MerchantItems, so the service should reject it
	// before even checking the def registry.
	_, _, err := svc.BuyItem(charID, tier, "widget_unknown")
	if err == nil {
		t.Fatal("expected error for unknown item def, got nil")
	}
}

func TestBuyItem_ErrorDefNotInRegistry(t *testing.T) {
	const charID uint = 50
	const tier = 0

	// Temporarily add the defID to MerchantItems so validation passes that
	// gate, then remove it to confirm the registry check fires.
	ghostID := "ghost_item"
	MerchantItems = append(MerchantItems, ghostID)
	defer func() { MerchantItems = MerchantItems[:len(MerchantItems)-1] }()

	repo := newTestRepo()
	svc := NewService(repo)
	repo.scrip[charID] = map[uint16]int{CurrentSeason: 9999}

	_, _, err := svc.BuyItem(charID, tier, ghostID)
	if err == nil {
		t.Fatal("expected error for def not in registry, got nil")
	}
}

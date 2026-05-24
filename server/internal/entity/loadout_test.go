package entity

import "testing"

const siphonPulseAbility = "siphon_pulse"

func TestHarmonistDefaultLoadout(t *testing.T) {
	p := NewPlayer(1, ClassArcanotechnicien)

	tests := []struct {
		action uint8
		want   string
	}{
		{50, siphonPulseAbility},
		{51, "mending_beam"},
		{52, "mending_surge"},
		{53, "restoration_matrix"},
		{54, "life_swap"},
		{55, "vital_drain"},
	}
	for _, tc := range tests {
		got, ok := p.ActionMap[tc.action]
		if !ok {
			t.Errorf("ActionMap[%d] missing, want %q", tc.action, tc.want)
			continue
		}
		if got != tc.want {
			t.Errorf("ActionMap[%d] = %q, want %q", tc.action, got, tc.want)
		}
	}
}

func TestApplyLoadoutUpdatesActionMap(t *testing.T) {
	p := NewPlayer(1, ClassArcanotechnicien)

	// Verify the initial state from default loadout.
	if p.ActionMap[50] != siphonPulseAbility {
		t.Fatalf("precondition: ActionMap[50] = %q, want siphon_pulse", p.ActionMap[50])
	}

	// Overwrite the loadout and re-apply.
	p.Loadout.Slots[0] = "new_heal_spell"
	p.ApplyLoadout()

	if p.ActionMap[50] != "new_heal_spell" {
		t.Errorf("ActionMap[50] = %q, want new_heal_spell after re-apply", p.ActionMap[50])
	}

	// Other slots should remain unchanged.
	if p.ActionMap[51] != "mending_beam" {
		t.Errorf("ActionMap[51] = %q, want mending_beam (unchanged)", p.ActionMap[51])
	}
}

func TestLoadoutSlotChangeUpdatesActionMap(t *testing.T) {
	tests := []struct {
		name    string
		slot    int
		ability string
		action  uint8
	}{
		{"slot 0", 0, "custom_heal_a", 50},
		{"slot 1", 1, "custom_heal_b", 51},
		{"slot 2", 2, "custom_heal_c", 52},
		{"slot 3", 3, "custom_heal_d", 53},
		{"slot 4", 4, "custom_heal_e", 54},
		{"slot 5", 5, "custom_heal_f", 55},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPlayer(1, ClassArcanotechnicien)
			p.Loadout.Slots[tc.slot] = tc.ability
			p.ApplyLoadout()
			if p.ActionMap[tc.action] != tc.ability {
				t.Errorf("ActionMap[%d] = %q, want %q", tc.action, p.ActionMap[tc.action], tc.ability)
			}
		})
	}
}

func TestDodgeStillWorksAfterLoadout(t *testing.T) {
	// Gunner has dodge mapped in ActionMap (arcanotechnicien handles dodge
	// as a special case in InputSystem, not through ActionMap).
	p := NewPlayer(1, ClassGunner)

	got, ok := p.ActionMap[3]
	if !ok {
		t.Fatal("ActionMap[3] (dodge) missing after loadout application")
	}
	if got != "dodge" {
		t.Errorf("ActionMap[3] = %q, want dodge", got)
	}
}

func TestApplyLoadoutNilIsNoop(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	// Gunner has no loadout; calling ApplyLoadout should not panic.
	p.ApplyLoadout()

	// Verify ActionMap is unchanged (gunner action 0 = fire_shot).
	if p.ActionMap[0] != "fire_shot" {
		t.Errorf("ActionMap[0] = %q, want fire_shot (unchanged)", p.ActionMap[0])
	}
}

func TestLoadoutEmptySlotsSkipped(t *testing.T) {
	p := NewPlayer(1, ClassArcanotechnicien)

	// Clear a slot and re-apply.
	p.Loadout.Slots[2] = ""
	// First, delete the key so we can verify it doesn't get re-added.
	delete(p.ActionMap, 52)
	p.ApplyLoadout()

	if _, ok := p.ActionMap[52]; ok {
		t.Error("ActionMap[52] should not be set for empty loadout slot")
	}

	// Non-empty slots should still be present.
	if p.ActionMap[50] != siphonPulseAbility {
		t.Errorf("ActionMap[50] = %q, want siphon_pulse", p.ActionMap[50])
	}
}

func TestLoadoutIsolationBetweenPlayers(t *testing.T) {
	p1 := NewPlayer(1, ClassArcanotechnicien)
	p2 := NewPlayer(2, ClassArcanotechnicien)

	// Mutate p1's loadout.
	p1.Loadout.Slots[0] = "custom_spell"
	p1.ApplyLoadout()

	// p2 should still have the default.
	if p2.ActionMap[50] != siphonPulseAbility {
		t.Errorf("p2 ActionMap[50] = %q, want siphon_pulse (p1 mutation leaked)", p2.ActionMap[50])
	}
	if p2.Loadout.Slots[0] != siphonPulseAbility {
		t.Errorf("p2 Loadout.Slots[0] = %q, want siphon_pulse (p1 mutation leaked)", p2.Loadout.Slots[0])
	}
}

func TestNonArcanotechnicienHasNoLoadout(t *testing.T) {
	tests := []struct {
		name  string
		class string
	}{
		{"gunner", ClassGunner},
		{"vanguard", ClassVanguard},
		{"blade_dancer", ClassBladeDancer},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPlayer(1, tc.class)
			if p.Loadout != nil {
				t.Errorf("%s should have nil Loadout", tc.class)
			}
		})
	}
}

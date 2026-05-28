package abilitycatalog

import (
	"path/filepath"
	"runtime"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/entity"
)

const testFireball = "fireball"

// yamlPath returns the absolute path to the real YAML catalog.
func yamlPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file location")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "data", "abilities", "arcanotechnicien.yaml")
}

func TestLoad(t *testing.T) {
	cat, err := Load(yamlPath(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cat.Abilities) == 0 {
		t.Fatal("expected at least one ability")
	}
	if len(cat.Affinities) == 0 {
		t.Fatal("expected at least one spec in affinities")
	}
}

func TestLoadBadPath(t *testing.T) {
	_, err := Load("/nonexistent/path/nope.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestGetAbility(t *testing.T) {
	cat, err := Load(yamlPath(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantNil bool
		wantID  string
	}{
		{name: "known ability mending_surge", id: ability.IDMendingSurge, wantNil: false, wantID: ability.IDMendingSurge},
		{name: "known ability fireball", id: testFireball, wantNil: false, wantID: testFireball},
		{name: "known ability frost_ward", id: ability.IDFrostWard, wantNil: false, wantID: ability.IDFrostWard},
		{name: "unknown ability", id: "nonexistent_spell_xyz", wantNil: true},
		{name: "empty string", id: "", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cat.GetAbility(tt.id)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetAbility(%q) = %+v, want nil", tt.id, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("GetAbility(%q) = nil, want non-nil", tt.id)
			}
			if got.ID != tt.wantID {
				t.Errorf("GetAbility(%q).ID = %q, want %q", tt.id, got.ID, tt.wantID)
			}
		})
	}
}

func TestGetAffinityTier(t *testing.T) {
	cat, err := Load(yamlPath(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name   string
		spec   string
		school string
		want   string
	}{
		// Harmonist affinities: primary=[bioarcanotechnic, biometabolic, frost], secondary=[aerokinetic, hydrodynamic, pure]
		{name: "harmonist/bioarcanotechnic is primary", spec: entity.SpecHarmonist, school: entity.SchoolBioarcanotechnic, want: AffinityPrimary},
		{name: "harmonist/biometabolic is primary", spec: entity.SpecHarmonist, school: entity.SchoolBiometabolic, want: AffinityPrimary},
		{name: "harmonist/frost is primary", spec: entity.SpecHarmonist, school: entity.SchoolFrost, want: AffinityPrimary},
		{name: "harmonist/aerokinetic is secondary", spec: entity.SpecHarmonist, school: entity.SchoolAerokinetic, want: AffinitySecondary},
		{name: "harmonist/hydrodynamic is secondary", spec: entity.SpecHarmonist, school: entity.SchoolHydrodynamic, want: AffinitySecondary},
		{name: "harmonist/pure is secondary", spec: entity.SpecHarmonist, school: entity.SchoolPure, want: AffinitySecondary},
		{name: "harmonist/fire is off", spec: entity.SpecHarmonist, school: entity.SchoolFire, want: AffinityOff},
		{name: "harmonist/shadow is off", spec: entity.SpecHarmonist, school: entity.SchoolShadow, want: AffinityOff},

		// Destroyer affinities: primary=[fire, frost, electricity], secondary=[gravitonic, aerokinetic, pure]
		{name: "destroyer/fire is primary", spec: entity.SpecDestroyer, school: entity.SchoolFire, want: AffinityPrimary},
		{name: "destroyer/gravitonic is secondary", spec: entity.SpecDestroyer, school: entity.SchoolGravitonic, want: AffinitySecondary},
		{name: "destroyer/bioarcanotechnic is off", spec: entity.SpecDestroyer, school: entity.SchoolBioarcanotechnic, want: AffinityOff},

		// Battlemage affinities: primary=[electricity, fire, martial], secondary=[shadow, aerokinetic, pure]
		{name: "battlemage/martial is primary", spec: entity.SpecBattlemage, school: entity.SchoolMartial, want: AffinityPrimary},
		{name: "battlemage/shadow is secondary", spec: entity.SpecBattlemage, school: entity.SchoolShadow, want: AffinitySecondary},

		// Unknown spec
		{name: "unknown spec returns off", spec: "nonexistent_spec", school: entity.SchoolFire, want: AffinityOff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cat.GetAffinityTier(tt.spec, tt.school)
			if got != tt.want {
				t.Errorf("GetAffinityTier(%q, %q) = %q, want %q", tt.spec, tt.school, got, tt.want)
			}
		})
	}
}

func TestValidateLoadout(t *testing.T) {
	cat, err := Load(yamlPath(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name  string
		slots [6]string
		want  bool
	}{
		{
			name:  "all valid implemented abilities",
			slots: [6]string{ability.IDMendingSurge, ability.IDMendingBeam, ability.IDFrostWard, "gust_step", ability.IDRestorationMatrix, ability.IDLifeSwap},
			want:  true,
		},
		{
			name:  "all empty slots",
			slots: [6]string{"", "", "", "", "", ""},
			want:  true,
		},
		{
			name:  "mix of valid and empty",
			slots: [6]string{ability.IDMendingSurge, "", ability.IDFrostWard, "", "", ""},
			want:  true,
		},
		{
			name:  "one unimplemented ability",
			slots: [6]string{ability.IDMendingSurge, testFireball, "", "", "", ""},
			want:  false,
		},
		{
			name:  "unknown ability id",
			slots: [6]string{"totally_fake_spell", "", "", "", "", ""},
			want:  false,
		},
		{
			name:  "all unimplemented",
			slots: [6]string{testFireball, "ignition", "burn", "flame_wall", "frost_nova", "ice_javelin"},
			want:  false,
		},
		{
			name:  "valid abilities plus one unknown at end",
			slots: [6]string{ability.IDMendingSurge, ability.IDFrostWard, "gust_step", ability.IDLifeSwap, ability.IDTransfusion, "does_not_exist"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cat.ValidateLoadout(tt.slots)
			if got != tt.want {
				t.Errorf("ValidateLoadout(%v) = %v, want %v", tt.slots, got, tt.want)
			}
		})
	}
}

func TestAllAbilities(t *testing.T) {
	cat, err := Load(yamlPath(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	all := cat.AllAbilities()
	if len(all) != len(cat.Abilities) {
		t.Fatalf("AllAbilities() returned %d abilities, catalog has %d", len(all), len(cat.Abilities))
	}

	// Verify it is a copy, not the same backing array.
	if len(all) > 0 {
		all[0].Name = "MUTATED"
		if cat.Abilities[0].Name == "MUTATED" {
			t.Error("AllAbilities() returned a slice sharing the catalog backing array")
		}
	}
}

func TestAbilitiesForSpec(t *testing.T) {
	cat, err := Load(yamlPath(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	results := cat.AbilitiesForSpec(entity.SpecHarmonist)
	if len(results) != len(cat.Abilities) {
		t.Fatalf("AbilitiesForSpec returned %d entries, want %d", len(results), len(cat.Abilities))
	}

	// Build a lookup for spot-checking.
	byID := make(map[string]AbilityWithAffinity, len(results))
	for _, r := range results {
		byID[r.ID] = r
	}

	tests := []struct {
		id           string
		wantAffinity string
	}{
		// Harmonist primary schools: bioarcanotechnic, biometabolic, frost
		{id: ability.IDMendingSurge, wantAffinity: AffinityPrimary}, // bioarcanotechnic
		{id: ability.IDLifeSwap, wantAffinity: AffinityPrimary},     // biometabolic
		{id: ability.IDFrostWard, wantAffinity: AffinityPrimary},    // frost
		// Harmonist secondary schools: aerokinetic, hydrodynamic, pure
		{id: "gust_step", wantAffinity: AffinitySecondary},     // aerokinetic
		{id: "torrent", wantAffinity: AffinitySecondary},       // hydrodynamic
		{id: "flux_negation", wantAffinity: AffinitySecondary}, // pure
		// Off-affinity
		{id: testFireball, wantAffinity: AffinityOff},          // fire
		{id: "chain_lightning", wantAffinity: AffinityOff},     // electricity
		{id: "shadow_step", wantAffinity: AffinityOff},         // shadow
		{id: "gravitonic_collapse", wantAffinity: AffinityOff}, // gravitonic
		{id: "adrenaline", wantAffinity: AffinityOff},          // martial
		{id: "mirage", wantAffinity: AffinityOff},              // illusion
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			entry, ok := byID[tt.id]
			if !ok {
				t.Fatalf("ability %q not found in AbilitiesForSpec results", tt.id)
			}
			if entry.Affinity != tt.wantAffinity {
				t.Errorf("AbilitiesForSpec(%q).Affinity = %q, want %q", tt.id, entry.Affinity, tt.wantAffinity)
			}
		})
	}
}

func TestAbilitiesForSpecUnknown(t *testing.T) {
	cat, err := Load(yamlPath(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	results := cat.AbilitiesForSpec("nonexistent_spec")
	for _, r := range results {
		if r.Affinity != AffinityOff {
			t.Errorf("ability %q has affinity %q for unknown spec, want %q", r.ID, r.Affinity, AffinityOff)
		}
	}
}

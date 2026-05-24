package abilitycatalog

import (
	"path/filepath"
	"runtime"
	"testing"
)

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
		{name: "known ability mending_surge", id: "mending_surge", wantNil: false, wantID: "mending_surge"},
		{name: "known ability fireball", id: "fireball", wantNil: false, wantID: "fireball"},
		{name: "known ability frost_ward", id: "frost_ward", wantNil: false, wantID: "frost_ward"},
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
		{name: "harmonist/bioarcanotechnic is primary", spec: "harmonist", school: "bioarcanotechnic", want: "primary"},
		{name: "harmonist/biometabolic is primary", spec: "harmonist", school: "biometabolic", want: "primary"},
		{name: "harmonist/frost is primary", spec: "harmonist", school: "frost", want: "primary"},
		{name: "harmonist/aerokinetic is secondary", spec: "harmonist", school: "aerokinetic", want: "secondary"},
		{name: "harmonist/hydrodynamic is secondary", spec: "harmonist", school: "hydrodynamic", want: "secondary"},
		{name: "harmonist/pure is secondary", spec: "harmonist", school: "pure", want: "secondary"},
		{name: "harmonist/fire is off", spec: "harmonist", school: "fire", want: "off"},
		{name: "harmonist/shadow is off", spec: "harmonist", school: "shadow", want: "off"},

		// Destroyer affinities: primary=[fire, frost, electricity], secondary=[gravitonic, aerokinetic, pure]
		{name: "destroyer/fire is primary", spec: "destroyer", school: "fire", want: "primary"},
		{name: "destroyer/gravitonic is secondary", spec: "destroyer", school: "gravitonic", want: "secondary"},
		{name: "destroyer/bioarcanotechnic is off", spec: "destroyer", school: "bioarcanotechnic", want: "off"},

		// Battlemage affinities: primary=[electricity, fire, martial], secondary=[shadow, aerokinetic, pure]
		{name: "battlemage/martial is primary", spec: "battlemage", school: "martial", want: "primary"},
		{name: "battlemage/shadow is secondary", spec: "battlemage", school: "shadow", want: "secondary"},

		// Unknown spec
		{name: "unknown spec returns off", spec: "nonexistent_spec", school: "fire", want: "off"},
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
			slots: [6]string{"mending_surge", "mending_beam", "frost_ward", "gust_step", "restoration_matrix", "life_swap"},
			want:  true,
		},
		{
			name:  "all empty slots",
			slots: [6]string{"", "", "", "", "", ""},
			want:  true,
		},
		{
			name:  "mix of valid and empty",
			slots: [6]string{"mending_surge", "", "frost_ward", "", "", ""},
			want:  true,
		},
		{
			name:  "one unimplemented ability",
			slots: [6]string{"mending_surge", "fireball", "", "", "", ""},
			want:  false,
		},
		{
			name:  "unknown ability id",
			slots: [6]string{"totally_fake_spell", "", "", "", "", ""},
			want:  false,
		},
		{
			name:  "all unimplemented",
			slots: [6]string{"fireball", "ignition", "burn", "flame_wall", "frost_nova", "ice_javelin"},
			want:  false,
		},
		{
			name:  "valid abilities plus one unknown at end",
			slots: [6]string{"mending_surge", "frost_ward", "gust_step", "life_swap", "transfusion", "does_not_exist"},
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

	results := cat.AbilitiesForSpec("harmonist")
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
		{id: "mending_surge", wantAffinity: "primary"}, // bioarcanotechnic
		{id: "life_swap", wantAffinity: "primary"},     // biometabolic
		{id: "frost_ward", wantAffinity: "primary"},    // frost
		// Harmonist secondary schools: aerokinetic, hydrodynamic, pure
		{id: "gust_step", wantAffinity: "secondary"},     // aerokinetic
		{id: "torrent", wantAffinity: "secondary"},       // hydrodynamic
		{id: "flux_negation", wantAffinity: "secondary"}, // pure
		// Off-affinity
		{id: "fireball", wantAffinity: "off"},            // fire
		{id: "chain_lightning", wantAffinity: "off"},     // electricity
		{id: "shadow_step", wantAffinity: "off"},         // shadow
		{id: "gravitonic_collapse", wantAffinity: "off"}, // gravitonic
		{id: "adrenaline", wantAffinity: "off"},          // martial
		{id: "mirage", wantAffinity: "off"},              // illusion
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
		if r.Affinity != "off" {
			t.Errorf("ability %q has affinity %q for unknown spec, want \"off\"", r.ID, r.Affinity)
		}
	}
}

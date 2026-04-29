package validation

import "testing"

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid lowercase", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid uppercase", "550E8400-E29B-41D4-A716-446655440000", true},
		{"valid mixed case", "550e8400-E29B-41d4-a716-446655440000", true},
		{"all zeros", "00000000-0000-0000-0000-000000000000", true},
		{"all f", "ffffffff-ffff-ffff-ffff-ffffffffffff", true},
		{"empty string", "", false},
		{"too short", "550e8400-e29b-41d4-a716", false},
		{"too long", "550e8400-e29b-41d4-a716-4466554400001", false},
		{"missing dash at 8", "550e84001e29b-41d4-a716-446655440000", false},
		{"missing dash at 13", "550e8400-e29b141d4-a716-446655440000", false},
		{"missing dash at 18", "550e8400-e29b-41d41a716-446655440000", false},
		{"missing dash at 23", "550e8400-e29b-41d4-a7161446655440000", false},
		{"invalid hex char g", "g50e8400-e29b-41d4-a716-446655440000", false},
		{"invalid hex char z", "550e8400-e29b-41d4-a716-44665544000z", false},
		{"spaces", "550e8400 e29b 41d4 a716 446655440000", false},
		{"no dashes", "550e8400e29b41d4a716446655440000", false},
		{"extra dash", "550e8400-e29b-41d4-a716-44665544-000", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValidUUID(tc.input)
			if got != tc.want {
				t.Errorf("IsValidUUID(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

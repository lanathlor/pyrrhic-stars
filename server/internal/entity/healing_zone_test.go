package entity

import "testing"

func TestHealingZone_ContainsPoint(t *testing.T) {
	tests := []struct {
		name   string
		center Vec3
		radius float32
		point  Vec3
		want   bool
	}{
		{
			name:   "origin inside",
			center: Vec3{},
			radius: 5,
			point:  Vec3{X: 3, Z: 3},
			want:   true,
		},
		{
			name:   "exactly on edge",
			center: Vec3{},
			radius: 5,
			point:  Vec3{X: 5, Z: 0},
			want:   true,
		},
		{
			name:   "outside radius",
			center: Vec3{},
			radius: 5,
			point:  Vec3{X: 4, Z: 4},
			want:   false,
		},
		{
			name:   "Y ignored",
			center: Vec3{Y: 0},
			radius: 5,
			point:  Vec3{X: 1, Y: 100, Z: 1},
			want:   true,
		},
		{
			name:   "offset center inside",
			center: Vec3{X: 10, Z: 10},
			radius: 3,
			point:  Vec3{X: 11, Z: 11},
			want:   true,
		},
		{
			name:   "offset center outside",
			center: Vec3{X: 10, Z: 10},
			radius: 1,
			point:  Vec3{X: 12, Z: 12},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &HealingZone{
				Position: tt.center,
				Radius:   tt.radius,
			}
			got := z.ContainsPoint(tt.point)
			if got != tt.want {
				t.Errorf("ContainsPoint(%v) = %v, want %v", tt.point, got, tt.want)
			}
		})
	}
}

func TestHealingZone_Tick(t *testing.T) {
	tests := []struct {
		name     string
		duration float32
		dt       float32
		expired  bool
		wantDur  float32
	}{
		{
			name:     "still alive",
			duration: 5.0,
			dt:       1.0,
			expired:  false,
			wantDur:  4.0,
		},
		{
			name:     "exactly expired",
			duration: 1.0,
			dt:       1.0,
			expired:  true,
			wantDur:  0.0,
		},
		{
			name:     "over expired",
			duration: 0.5,
			dt:       1.0,
			expired:  true,
			wantDur:  -0.5,
		},
		{
			name:     "tiny tick still alive",
			duration: 0.05,
			dt:       0.01,
			expired:  false,
			wantDur:  0.04,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &HealingZone{Duration: tt.duration}
			got := z.Tick(tt.dt)
			if got != tt.expired {
				t.Errorf("Tick(%v) expired = %v, want %v", tt.dt, got, tt.expired)
			}
			diff := z.Duration - tt.wantDur
			if diff > 0.001 || diff < -0.001 {
				t.Errorf("Duration after Tick = %v, want %v", z.Duration, tt.wantDur)
			}
		})
	}
}

func TestHealingZone_ShouldTick(t *testing.T) {
	tests := []struct {
		name      string
		tickTimer float32
		interval  float32
		dt        float32
		want      bool
	}{
		{
			name:      "fires on first interval",
			tickTimer: 0.5,
			interval:  0.5,
			dt:        0.5,
			want:      true,
		},
		{
			name:      "not yet",
			tickTimer: 0.5,
			interval:  0.5,
			dt:        0.3,
			want:      false,
		},
		{
			name:      "fires past interval",
			tickTimer: 0.5,
			interval:  0.5,
			dt:        0.7,
			want:      true,
		},
		{
			name:      "timer resets after fire",
			tickTimer: 0.5,
			interval:  1.0,
			dt:        0.6,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &HealingZone{
				TickTimer: tt.tickTimer,
				Interval:  tt.interval,
			}
			got := z.ShouldTick(tt.dt)
			if got != tt.want {
				t.Errorf("ShouldTick(%v) = %v, want %v", tt.dt, got, tt.want)
			}
		})
	}
}

func TestHealingZone_ShouldTick_MultipleIntervals(t *testing.T) {
	z := &HealingZone{
		TickTimer: 1.0,
		Interval:  1.0,
	}

	// First 0.5s: no fire
	if z.ShouldTick(0.5) {
		t.Error("should not fire at 0.5s")
	}

	// Next 0.5s: fires at 1.0s boundary
	if !z.ShouldTick(0.5) {
		t.Error("should fire at 1.0s boundary")
	}

	// Timer should have reset to ~1.0
	if z.TickTimer < 0.9 || z.TickTimer > 1.1 {
		t.Errorf("TickTimer after fire = %v, want ~1.0", z.TickTimer)
	}

	// Another 0.5s: no fire
	if z.ShouldTick(0.5) {
		t.Error("should not fire 0.5s after last pulse")
	}
}

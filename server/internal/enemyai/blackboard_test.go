package enemyai

import "testing"

func TestBlackboardFlags(t *testing.T) {
	bb := NewBlackboard()

	if bb.GetFlag("x") {
		t.Fatal("unset flag should be false")
	}
	bb.SetFlag("x")
	if !bb.GetFlag("x") {
		t.Fatal("set flag should be true")
	}
	bb.ClearFlag("x")
	if bb.GetFlag("x") {
		t.Fatal("cleared flag should be false")
	}
}

func TestBlackboardCounters(t *testing.T) {
	bb := NewBlackboard()

	if bb.GetCounter("c") != 0 {
		t.Fatal("unset counter should be 0")
	}
	bb.SetCounter("c", 5)
	if bb.GetCounter("c") != 5 {
		t.Fatalf("want 5, got %d", bb.GetCounter("c"))
	}
	bb.IncrementCounter("c")
	if bb.GetCounter("c") != 6 {
		t.Fatalf("want 6, got %d", bb.GetCounter("c"))
	}
}

func TestBlackboardTimers(t *testing.T) {
	bb := NewBlackboard()

	// Not started = expired
	if !bb.TimerExpired("t") {
		t.Fatal("absent timer should be expired")
	}

	bb.StartTimer("t", 1.0)
	if bb.TimerExpired("t") {
		t.Fatal("fresh timer should not be expired")
	}
	if r := bb.TimerRemaining("t"); r != 1.0 {
		t.Fatalf("want 1.0, got %f", r)
	}

	// Tick partially
	bb.TickTimers(0.4)
	if bb.TimerExpired("t") {
		t.Fatal("timer should still be active after 0.4s")
	}
	if r := bb.TimerRemaining("t"); r < 0.59 || r > 0.61 {
		t.Fatalf("want ~0.6, got %f", r)
	}

	// Tick past expiry
	bb.TickTimers(0.7)
	if !bb.TimerExpired("t") {
		t.Fatal("timer should be expired after 1.1s total")
	}
	if r := bb.TimerRemaining("t"); r != 0 {
		t.Fatalf("expired timer remaining should be 0, got %f", r)
	}
}

func TestBlackboardValues(t *testing.T) {
	bb := NewBlackboard()

	bb.Set("str", "hello")
	bb.Set("num", float32(3.14))
	bb.Set("idx", 42)

	if bb.GetString("str") != "hello" {
		t.Fatalf("want hello, got %s", bb.GetString("str"))
	}
	if bb.GetFloat32("num") != 3.14 {
		t.Fatalf("want 3.14, got %f", bb.GetFloat32("num"))
	}
	if bb.GetInt("idx") != 42 {
		t.Fatalf("want 42, got %d", bb.GetInt("idx"))
	}

	// Wrong type returns zero
	if bb.GetInt("str") != 0 {
		t.Fatal("wrong type should return zero")
	}
	if bb.GetString("missing") != "" {
		t.Fatal("missing key should return zero")
	}

	bb.Delete("str")
	if bb.Get("str") != nil {
		t.Fatal("deleted key should be nil")
	}
}

func TestBlackboardReset(t *testing.T) {
	bb := NewBlackboard()
	bb.SetFlag("f")
	bb.SetCounter("c", 10)
	bb.StartTimer("t", 5.0)
	bb.Set("v", "x")

	bb.Reset()

	if bb.GetFlag("f") {
		t.Fatal("flag should be cleared after reset")
	}
	if bb.GetCounter("c") != 0 {
		t.Fatal("counter should be 0 after reset")
	}
	if !bb.TimerExpired("t") {
		t.Fatal("timer should be expired after reset")
	}
	if bb.Get("v") != nil {
		t.Fatal("value should be nil after reset")
	}
}

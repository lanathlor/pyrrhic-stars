package bosstest_test

import (
	"testing"

	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combatlog"
)

func TestReplayFramesContainProjectiles(t *testing.T) {
	sink := combatlog.NewInMemorySink()

	result := bosstest.RunSimulation(bosstest.SimConfig{
		Boss: "guard_captain",
		Party: []bosstest.PuppetConfig{
			{Class: "gunner", Profile: bosstest.ProfileAverage},
			{Class: "vanguard", Profile: bosstest.ProfileAverage},
		},
		Seed:    42,
		Sink:    sink,
		GroupID: "test_proj",
		RunID:   "test_proj_42",
	})

	// instanceID = GroupID + "_" + Seed
	frames := sink.ReplayFrames("test_proj_42")
	if len(frames) == 0 {
		t.Fatal("no replay frames recorded")
	}

	var framesWithProj int
	var totalProjs int
	for i, frame := range frames {
		ws, ok := codec.DecodeWorldState(frame)
		if !ok {
			t.Fatalf("frame %d: DecodeWorldState failed", i)
		}
		if len(ws.Projectiles) > 0 {
			framesWithProj++
			totalProjs += len(ws.Projectiles)
		}
	}

	t.Logf("outcome=%s ticks=%d frames=%d framesWithProjectiles=%d totalProjectileInstances=%d",
		result.Outcome, result.TotalTicks, len(frames), framesWithProj, totalProjs)

	if framesWithProj == 0 {
		t.Error("no replay frames contain projectiles — simulation should produce ranged attacks")
	}
}

func TestReplayFramesContainPlayerHealth(t *testing.T) {
	sink := combatlog.NewInMemorySink()

	bosstest.RunSimulation(bosstest.SimConfig{
		Boss: "guard_captain",
		Party: []bosstest.PuppetConfig{
			{Class: "vanguard", Profile: bosstest.ProfileBad},
		},
		Seed:    99,
		Sink:    sink,
		GroupID: "test_hp",
		RunID:   "test_hp_99",
	})

	// instanceID = GroupID + "_" + Seed
	frames := sink.ReplayFrames("test_hp_99")
	if len(frames) == 0 {
		t.Fatal("no replay frames recorded")
	}

	var framesWithDamage int
	for _, frame := range frames {
		ws, ok := codec.DecodeWorldState(frame)
		if !ok {
			continue
		}
		for _, p := range ws.Players {
			if p.Health < 200.0 { // vanguard max HP
				framesWithDamage++
				break
			}
		}
	}

	if framesWithDamage == 0 {
		t.Error("no replay frames show player health below max — boss should deal damage")
	}
}

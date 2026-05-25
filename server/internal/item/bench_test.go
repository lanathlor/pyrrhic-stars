package item

import "testing"

// --- scalingFactor ---

func BenchmarkScalingFactor_Hull(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		scalingFactor(StatHull, 35)
	}
}

func BenchmarkScalingFactor_AllStats(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for s := range StatCount {
			scalingFactor(s, 35)
		}
	}
}

// --- ScaleStatLine ---

func BenchmarkScaleStatLine(b *testing.B) {
	line := StatLine{Stat: StatHull, Value: 90}
	b.ReportAllocs()
	for b.Loop() {
		ScaleStatLine(line, 35)
	}
}

// --- ComputeStats (full 6-slot kit) ---

func benchEquipped() [SlotCount]*Item {
	// Register test defs so ComputeStats can look them up.
	DefRegistry["bench_frame"] = &ItemDef{
		ID: "bench_frame", Slot: SlotFrame,
		StatLines: []StatLine{
			{Stat: StatHull, Value: 90},
			{Stat: StatPlating, Value: 12},
			{Stat: StatMastery, Value: 5},
		},
	}
	DefRegistry["bench_core"] = &ItemDef{
		ID: "bench_core", Slot: SlotPowerCore,
		StatLines: []StatLine{
			{Stat: StatHull, Value: 20},
			{Stat: StatOutput, Value: 22},
			{Stat: StatTempo, Value: 8},
		},
	}
	DefRegistry["bench_weapon"] = &ItemDef{
		ID: "bench_weapon", Slot: SlotPrimaryWeapon,
		StatLines: []StatLine{
			{Stat: StatOutput, Value: 25},
			{Stat: StatIdentity, Value: 10},
			{Stat: StatMastery, Value: 5},
		},
	}
	DefRegistry["bench_tool"] = &ItemDef{
		ID: "bench_tool", Slot: SlotSecondaryTool,
		StatLines: []StatLine{
			{Stat: StatOutput, Value: 8},
			{Stat: StatTempo, Value: 8},
			{Stat: StatPlating, Value: 8},
		},
	}
	DefRegistry["bench_aug"] = &ItemDef{
		ID: "bench_aug", Slot: SlotAugment,
		StatLines: []StatLine{
			{Stat: StatOutput, Value: 12},
			{Stat: StatIdentity, Value: 8},
			{Stat: StatMastery, Value: 6},
		},
	}
	DefRegistry["bench_mod"] = &ItemDef{
		ID: "bench_mod", Slot: SlotModule,
		StatLines: []StatLine{
			{Stat: StatHull, Value: 40},
			{Stat: StatOutput, Value: 8},
			{Stat: StatTempo, Value: 6},
		},
	}

	return [SlotCount]*Item{
		{DefID: "bench_frame", ILvl: 35, Slot: SlotFrame},
		{DefID: "bench_core", ILvl: 35, Slot: SlotPowerCore},
		{DefID: "bench_weapon", ILvl: 35, Slot: SlotPrimaryWeapon},
		{DefID: "bench_tool", ILvl: 30, Slot: SlotSecondaryTool},
		{DefID: "bench_aug", ILvl: 32, Slot: SlotAugment},
		{DefID: "bench_mod", ILvl: 28, Slot: SlotModule},
	}
}

func BenchmarkComputeStats_FullKit(b *testing.B) {
	equipped := benchEquipped()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		ComputeStats(equipped)
	}
}

func BenchmarkComputeStats_3Slots(b *testing.B) {
	equipped := benchEquipped()
	// Only frame, core, weapon equipped
	equipped[3] = nil
	equipped[4] = nil
	equipped[5] = nil
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		ComputeStats(equipped)
	}
}

func BenchmarkComputeStats_Empty(b *testing.B) {
	var equipped [SlotCount]*Item
	b.ReportAllocs()
	for b.Loop() {
		ComputeStats(equipped)
	}
}

// --- ComputeStatsForItem ---

func BenchmarkComputeStatsForItem(b *testing.B) {
	equipped := benchEquipped()
	it := equipped[0]
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		ComputeStatsForItem(it)
	}
}

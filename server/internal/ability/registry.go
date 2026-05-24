package ability

import "codex-online/server/internal/entity"

func registerAbilities(eng *Engine) {
	// Shared
	eng.Register(&dodgeDef)

	// Gunner
	eng.Register(&fireShotDef)
	eng.Register(&overclockDef)
	eng.Register(&rechamberDef)
	eng.Register(&rechamberConfirmDef)
	eng.Register(&reloadDef)
	eng.Register(&loadEnhancedDef)
	eng.Register(&magDumpDef)

	// Vanguard — Blade
	eng.Register(&meleeLightDef)  // Cleave
	eng.Register(&meleeHeavyDef)  // Upheaval
	eng.Register(&vgBlockDef)     // Blade Parry
	eng.Register(&vgBlockStopDef)
	eng.Register(&bladeSwirlDef)  // Vortex
	eng.Register(&groundSlamDef)  // Execution

	// Vanguard — Shield
	eng.Register(&vgShieldBlockDef)
	eng.Register(&vgShieldBlockStopDef)
	eng.Register(&shieldBashDef)
	eng.Register(&bullRushDef)
	eng.Register(&braceDef)
	eng.Register(&retaliateDef)

	// Blade Dancer
	eng.Register(&bdGuardDef)
	for _, def := range bdTransitionSpells() {
		eng.Register(def)
	}

	// Arcanotechnicien — Harmonist
	eng.Register(&siphonPulseDef)
	eng.Register(&mendingSurgeDef)
	eng.Register(&mendingBeamDef)
	eng.Register(&vitalBloomDef)
	eng.Register(&restorationMatrixDef)
	eng.Register(&lifeSwapDef)
	eng.Register(&transfusionDef)
	eng.Register(&vitalDrainDef)
	eng.Register(&overclockATDef)
	eng.Register(&neuralFortificationDef)
	eng.Register(&regenProtocolDef)
	eng.Register(&vitalCircuitDef)
	eng.Register(&metabolicBurstDef)
	eng.Register(&lastBreathDef)
	eng.Register(&gustStepDef)
	eng.Register(&frostWardDef)
}

func registerHandlers(eng *Engine) {
	eng.RegisterHandler("rechamber", rechamberHandler)
	eng.RegisterHandler("rechamber_confirm", rechamberConfirmHandler)
	eng.RegisterHandler("overclock", overclockHandler)
	eng.RegisterHandler("fire_shot_assault", fireShotAssaultHandler)
	eng.RegisterHandler("reload_assault", reloadAssaultHandler)
	eng.RegisterHandler("load_enhanced_assault", loadEnhancedHandler)
	eng.RegisterHandler("mag_dump_assault", magDumpHandler)
	eng.RegisterHandler("vortex", vortexHandler)
	eng.RegisterHandler("vg_block", vgBlockHandler)
	eng.RegisterHandler("vg_block_stop", vgBlockStopHandler)
	eng.RegisterHandler("vg_shield_block", vgShieldBlockHandler)
	eng.RegisterHandler("vg_shield_block_stop", vgShieldBlockStopHandler)
	eng.RegisterHandler("shield_bash", shieldBashHandler)
	eng.RegisterHandler("bull_rush", bullRushHandler)
	eng.RegisterHandler("brace", braceHandler)
	eng.RegisterHandler("retaliate", retaliateHandler)
	eng.RegisterHandler("cleave_vg", cleaveHandler)
	eng.RegisterHandler("upheaval_vg", upheavalHandler)
	eng.RegisterHandler("execution_vg", executionVGHandler)

	eng.RegisterHandler("siphon_pulse", siphonPulseHandler)
	eng.RegisterHandler("mending_surge", mendingSurgeHandler)
	eng.RegisterHandler("mending_beam", mendingBeamHandler)
	eng.RegisterHandler("vital_bloom", vitalBloomHandler)
	eng.RegisterHandler("restoration_matrix", restorationMatrixHandler)
	eng.RegisterHandler("life_swap", lifeSwapHandler)
	eng.RegisterHandler("transfusion", transfusionHandler)
	eng.RegisterHandler("vital_drain", vitalDrainHandler)
	eng.RegisterHandler("overclock_at", overclockATHandler)
	eng.RegisterHandler("neural_fortification", neuralFortificationHandler)
	eng.RegisterHandler("regen_protocol", regenProtocolHandler)
	eng.RegisterHandler("vital_circuit", vitalCircuitHandler)
	eng.RegisterHandler("metabolic_burst", metabolicBurstHandler)
	eng.RegisterHandler("last_breath", lastBreathHandler)
	eng.RegisterHandler("gust_step", gustStepHandler)
	eng.RegisterHandler("frost_ward", frostWardHandler)

	eng.RegisterTickHandler("rechamber", rechamberTick)
	eng.RegisterTickHandler("vortex", vortexTick)
	eng.RegisterTickHandler("vg_block", vgBlockTick)
	eng.RegisterTickHandler("vg_shield_block", vgShieldBlockTick)
	eng.RegisterTickHandler("gunner_assault", gunnerAssaultTick)
	eng.RegisterTickHandler("bd_flow", bdFlowTick)
	eng.RegisterTickHandler("frost_ward", frostWardTick)
}

// ApplyThreat adds threat for all damage results to any Threateable target.
func ApplyThreat(results []DamageResult, peerID uint16) {
	for _, r := range results {
		if th, ok := r.Target.(entity.Threateable); ok {
			th.AddThreat(peerID, r.Amount)
		}
	}
}

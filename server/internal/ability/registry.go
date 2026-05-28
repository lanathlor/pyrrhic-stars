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
	eng.Register(&meleeLightDef) // Cleave
	eng.Register(&meleeHeavyDef) // Upheaval
	eng.Register(&vgBlockDef)    // Blade Parry
	eng.Register(&vgBlockStopDef)
	eng.Register(&bladeSwirlDef) // Vortex
	eng.Register(&groundSlamDef) // Execution

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
	registerGunnerHandlers(eng)
	registerVanguardHandlers(eng)
	registerSupportHandlers(eng)
	registerTickHandlers(eng)
}

func registerGunnerHandlers(eng *Engine) {
	eng.RegisterHandler(IDRechamber, rechamberHandler)
	eng.RegisterHandler(IDRechamberConfirm, rechamberConfirmHandler)
	eng.RegisterHandler(IDOverclock, overclockHandler)
	eng.RegisterHandler("fire_shot_assault", fireShotAssaultHandler)
	eng.RegisterHandler("reload_assault", reloadAssaultHandler)
	eng.RegisterHandler("load_enhanced_assault", loadEnhancedHandler)
	eng.RegisterHandler("mag_dump_assault", magDumpHandler)
}

func registerVanguardHandlers(eng *Engine) {
	eng.RegisterHandler(IDVortex, vortexHandler)
	eng.RegisterHandler(IDVgBlock, vgBlockHandler)
	eng.RegisterHandler(IDVgBlockStop, vgBlockStopHandler)
	eng.RegisterHandler(IDVgShieldBlock, vgShieldBlockHandler)
	eng.RegisterHandler(IDVgShieldBlockStop, vgShieldBlockStopHandler)
	eng.RegisterHandler("shield_bash", shieldBashHandler)
	eng.RegisterHandler("bull_rush", bullRushHandler)
	eng.RegisterHandler(IDBrace, braceHandler)
	eng.RegisterHandler("retaliate", retaliateHandler)
	eng.RegisterHandler("cleave_vg", cleaveHandler)
	eng.RegisterHandler("upheaval_vg", upheavalHandler)
	eng.RegisterHandler("execution_vg", executionVGHandler)
}

func registerSupportHandlers(eng *Engine) {
	eng.RegisterHandler(IDSiphonPulse, siphonPulseHandler)
	eng.RegisterHandler(IDMendingSurge, mendingSurgeHandler)
	eng.RegisterHandler(IDMendingBeam, mendingBeamHandler)
	eng.RegisterHandler(IDVitalBloom, vitalBloomHandler)
	eng.RegisterHandler(IDRestorationMatrix, restorationMatrixHandler)
	eng.RegisterHandler(IDLifeSwap, lifeSwapHandler)
	eng.RegisterHandler(IDTransfusion, transfusionHandler)
	eng.RegisterHandler(IDVitalDrain, vitalDrainHandler)
	eng.RegisterHandler(IDOverclockAT, overclockATHandler)
	eng.RegisterHandler("neural_fortification", neuralFortificationHandler)
	eng.RegisterHandler(IDRegenProtocol, regenProtocolHandler)
	eng.RegisterHandler("vital_circuit", vitalCircuitHandler)
	eng.RegisterHandler("metabolic_burst", metabolicBurstHandler)
	eng.RegisterHandler(IDLastBreath, lastBreathHandler)
	eng.RegisterHandler("gust_step", gustStepHandler)
	eng.RegisterHandler(IDFrostWard, frostWardHandler)
}

func registerTickHandlers(eng *Engine) {
	eng.RegisterTickHandler(IDRechamber, rechamberTick)
	eng.RegisterTickHandler(IDVortex, vortexTick)
	eng.RegisterTickHandler(IDVgBlock, vgBlockTick)
	eng.RegisterTickHandler(IDVgShieldBlock, vgShieldBlockTick)
	eng.RegisterTickHandler("gunner_assault", gunnerAssaultTick)
	eng.RegisterTickHandler("bd_flow", bdFlowTick)
	eng.RegisterTickHandler(IDFrostWard, frostWardTick)
}

// ApplyThreat adds threat for all damage results to any Threateable target.
func ApplyThreat(results []DamageResult, peerID uint16) {
	for _, r := range results {
		if th, ok := r.Target.(entity.Threateable); ok {
			th.AddThreat(peerID, r.Amount)
		}
	}
}

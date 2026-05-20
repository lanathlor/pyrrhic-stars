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

	// Vanguard
	eng.Register(&meleeLightDef)  // Cleave
	eng.Register(&meleeHeavyDef)  // Upheaval
	eng.Register(&vgBlockDef)     // Blade Parry
	eng.Register(&vgBlockStopDef)
	eng.Register(&bladeSwirlDef)  // Vortex
	eng.Register(&groundSlamDef)  // Execution

	// Blade Dancer
	eng.Register(&bdGuardDef)
	for _, def := range bdTransitionSpells() {
		eng.Register(def)
	}
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
	eng.RegisterHandler("cleave_vg", cleaveHandler)
	eng.RegisterHandler("upheaval_vg", upheavalHandler)
	eng.RegisterHandler("execution_vg", executionVGHandler)

	eng.RegisterTickHandler("rechamber", rechamberTick)
	eng.RegisterTickHandler("vortex", vortexTick)
	eng.RegisterTickHandler("vg_block", vgBlockTick)
	eng.RegisterTickHandler("gunner_assault", gunnerAssaultTick)
	eng.RegisterTickHandler("bd_flow", bdFlowTick)
}

// ApplyThreat adds threat for all damage results to any Threateable target.
func ApplyThreat(results []DamageResult, peerID uint16) {
	for _, r := range results {
		if th, ok := r.Target.(entity.Threateable); ok {
			th.AddThreat(peerID, r.Amount)
		}
	}
}

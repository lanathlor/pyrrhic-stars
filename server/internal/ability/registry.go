package ability

func registerAbilities(eng *Engine) {
	// Shared
	eng.Register(&dodgeDef)

	// Gunner
	eng.Register(&fireShotDef)
	eng.Register(&overclockDef)
	eng.Register(&rechamberDef)
	eng.Register(&rechamberConfirmDef)

	// Vanguard
	eng.Register(&meleeLightDef)
	eng.Register(&meleeHeavyDef)
	eng.Register(&vgBlockDef)
	eng.Register(&bladeSwirlDef)
	eng.Register(&groundSlamDef)

	// Blade Dancer
	eng.Register(&bdMeleeDef)
	eng.Register(&bdHeavyDef)
	eng.Register(&bdGuardDef)
	for _, def := range bdTransitionSpells() {
		eng.Register(def)
	}
}

func registerHandlers(eng *Engine) {
	eng.RegisterHandler("rechamber", rechamberHandler)
	eng.RegisterHandler("rechamber_confirm", rechamberConfirmHandler)
	eng.RegisterHandler("overclock", overclockHandler)
	eng.RegisterHandler("blade_swirl", bladeSwirlHandler)
	eng.RegisterHandler("vg_block", vgBlockHandler)
	eng.RegisterHandler("melee_light_vg", meleeLightVGHandler)
	eng.RegisterHandler("melee_heavy_vg", meleeHeavyVGHandler)

	eng.RegisterTickHandler("rechamber", rechamberTick)
	eng.RegisterTickHandler("blade_swirl", bladeSwirlTick)
}

// ApplyThreat adds threat for all damage results to the relevant enemies.
func ApplyThreat(results []DamageResult, peerID uint16) {
	for _, r := range results {
		if r.Enemy != nil {
			r.Enemy.AddThreat(peerID, r.Amount)
		}
	}
}

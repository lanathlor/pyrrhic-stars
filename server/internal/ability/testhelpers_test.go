package ability

// Test case name constants.
const (
	tcGrantsConfluence              = "grants confluence stack"
	tcRejectsInsufficientFlux       = "rejects on insufficient flux"
	tcRejectsGCD                    = "rejects on GCD"
	tcRejectsCooldown               = "rejects on cooldown"
	tcInsufficientFluxBeforeHandler = "insufficient flux rejects before handler"
	tcTestDmg                       = "test_dmg"

	tcTestSustain = "test_sustain"

	// Dynamic "insufficient [school] flux" reason strings used in tests.
	tcInsufficientBioarcanotechnicFlux = "insufficient bioarcanotechnic flux"
	tcInsufficientBiometabolicFlux     = "insufficient biometabolic flux"
)

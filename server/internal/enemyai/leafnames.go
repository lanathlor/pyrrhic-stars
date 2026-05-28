package enemyai

// BT leaf name constants.
const (
	LeafIsDead             = "is_dead"
	LeafPhaseTransitioning = "phase_transitioning"
	LeafHasLoS             = "has_los"
	LeafStop               = "stop"
	LeafAggroOrPatrol      = "aggro_or_patrol"
	LeafLeashReset         = "leash_reset"
	LeafChase              = "chase"
	LeafWaitAbility        = "wait_ability"
	LeafWaitTransition     = "wait_transition"
)

// BT composite node type constants used in YAML tree definitions.
const (
	NodeSequence         = "sequence"
	NodeSelector         = "selector"
	NodeReactiveSelector = "reactive_selector"
)

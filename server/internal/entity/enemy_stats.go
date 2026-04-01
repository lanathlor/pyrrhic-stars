package entity

// Phase-scaled stat getters for the enemy boss.
// Values transcribed from client/scenes/enemies/basic_enemy/basic_enemy.gd.

func (e *Enemy) GetMoveSpeed() float32 {
	switch e.Phase {
	case 2:
		return 5.0
	case 3:
		return 6.0
	default:
		return 4.0
	}
}

func (e *Enemy) GetMeleeDamage() float32 {
	if e.Phase == 3 {
		return 35.0
	}
	return 30.0
}

func (e *Enemy) getMeleeTelegraphTime() float32 {
	switch e.Phase {
	case 2:
		return 0.9
	case 3:
		return 0.7
	default:
		return 1.2
	}
}

func (e *Enemy) getRangedTelegraphTime() float32 {
	switch e.Phase {
	case 2:
		return 0.8
	case 3:
		return 0.6
	default:
		return 1.0
	}
}

func (e *Enemy) GetRangedPerProjectileDamage() float32 {
	switch e.Phase {
	case 2:
		return 15.0
	case 3:
		return 12.0
	default:
		return 20.0
	}
}

func (e *Enemy) GetRangedBurstCount() int {
	switch e.Phase {
	case 2:
		return 2
	case 3:
		return 3
	default:
		return 1
	}
}

func (e *Enemy) GetAoEDamage() float32 {
	if e.Phase == 3 {
		return 45.0
	}
	return 40.0
}

func (e *Enemy) GetAoERadius() float32 {
	switch e.Phase {
	case 2:
		return 6.0
	case 3:
		return 7.0
	default:
		return 5.0
	}
}

func (e *Enemy) getAoETelegraphTime() float32 {
	switch e.Phase {
	case 2:
		return 1.2
	case 3:
		return 1.0
	default:
		return 1.5
	}
}

func (e *Enemy) GetChargeDamage() float32 {
	if e.Phase == 3 {
		return 40.0
	}
	return 35.0
}

func (e *Enemy) GetChargeSpeed() float32 {
	switch e.Phase {
	case 2:
		return 14.0
	case 3:
		return 16.0
	default:
		return 12.0
	}
}

func (e *Enemy) getChargeTelegraphTime() float32 {
	switch e.Phase {
	case 2:
		return 0.8
	case 3:
		return 0.6
	default:
		return 1.0
	}
}

func (e *Enemy) GetChargeMaxDistance() float32 {
	switch e.Phase {
	case 2:
		return 18.0
	case 3:
		return 20.0
	default:
		return 15.0
	}
}

func (e *Enemy) getCooldownTime() float32 {
	switch e.Phase {
	case 2:
		return 1.2
	case 3:
		return 0.9
	default:
		return 1.5
	}
}

// PhaseWeights returns attack weights [melee, ranged, aoe, charge] for the current phase.
func (e *Enemy) PhaseWeights() [4]int {
	switch e.Phase {
	case 2:
		return [4]int{25, 25, 25, 25}
	case 3:
		return [4]int{20, 20, 25, 35}
	default:
		return [4]int{30, 30, 20, 20}
	}
}

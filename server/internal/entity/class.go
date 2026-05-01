package entity

import "codex-online/server/internal/entity/class"

// Class name constants (re-exported from class subpackage).
const (
	ClassGunner      = class.Gunner
	ClassVanguard    = class.Vanguard
	ClassBladeDancer = class.BladeDancer
)

// Type aliases for backward compatibility.
type ClassMovement = class.Movement
type ResourceTemplate = class.ResourceTemplate
type ClassDef = class.Def

// Classes is the global class registry.
var Classes = class.Registry

class_name ItemData

## Static item/stat definitions. Matches server/internal/item/item.go.

# Stat IDs — must match server StatID enum
enum Stat { HULL = 0, OUTPUT = 1, PLATING = 2, TEMPO = 3, IDENTITY = 4, MASTERY = 5 }

# Equipment slot IDs — must match server SlotID enum
enum Slot {
	FRAME = 0, POWER_CORE = 1, PRIMARY_WEAPON = 2, SECONDARY_TOOL = 3, AUGMENT = 4, MODULE = 5
}

const SLOT_COUNT := 6

const STAT_NAMES := {
	Stat.HULL: "Hull",
	Stat.OUTPUT: "Output",
	Stat.PLATING: "Plating",
	Stat.TEMPO: "Tempo",
	Stat.IDENTITY: "Identity",
	Stat.MASTERY: "Mastery",
}

const SLOT_NAMES := {
	Slot.FRAME: "Frame",
	Slot.POWER_CORE: "Power Core",
	Slot.PRIMARY_WEAPON: "Primary Weapon",
	Slot.SECONDARY_TOOL: "Secondary Tool",
	Slot.AUGMENT: "Augment",
	Slot.MODULE: "Module",
}

const STAT_COLORS := {
	Stat.HULL: Color(0.3, 0.8, 0.3),  # green
	Stat.OUTPUT: Color(0.9, 0.3, 0.2),  # red
	Stat.PLATING: Color(0.5, 0.6, 0.8),  # steel blue
	Stat.TEMPO: Color(0.9, 0.8, 0.2),  # gold
	Stat.IDENTITY: Color(0.6, 0.3, 0.9),  # purple
	Stat.MASTERY: Color(0.2, 0.7, 0.9),  # cyan
}

# Class base HP — matches server ClassDef values
const CLASS_BASE_HP := {
	"gunner": 150.0,
	"vanguard": 200.0,
	"blade_dancer": 150.0,
}

# Class-specific stat names and short descriptions (Tempo, Identity, Mastery)
const CLASS_STAT_INFO := {
	"gunner":
	{
		"tempo": ["Action", "fire rate, ADS speed"],
		"identity": ["Munitions", "enhanced round reserve, regen"],
		"mastery": ["Pressure", "dmg per consecutive hit"],
	},
	"vanguard":
	{
		"tempo": ["Recovery", "parry, dodge, combo windows"],
		"identity": ["Tenacity", "stamina pool, efficiency, regen"],
		"mastery": ["Onslaught", "dmg on hit streaks"],
	},
	"blade_dancer":
	{
		"tempo": ["Transition", "config change speed"],
		"identity": ["Resonance", "charge capacity, gain, retention"],
		"mastery": ["Flow", "transition chain bonus"],
	},
}


## Get display name for a stat, using class-specific names for secondary stats.
static func class_stat_name(stat_id: int, cls: String) -> String:
	if stat_id < 3:
		return STAT_NAMES.get(stat_id, "?")
	var key: String = ["tempo", "identity", "mastery"][stat_id - 3]
	var info: Dictionary = CLASS_STAT_INFO.get(cls, {})
	var entry: Array = info.get(key, [])
	return entry[0] if entry.size() > 0 else STAT_NAMES.get(stat_id, "?")


## Get the gameplay effect description for a stat value.
static func stat_effect_desc(stat_id: int, value: float, cls: String) -> String:
	match stat_id:
		Stat.HULL:
			return "+%.0f HP" % value
		Stat.OUTPUT:
			return "+%.0f%% damage" % value
		Stat.PLATING:
			return "-%.0f per hit" % value
		Stat.TEMPO:
			var pct := 100.0 * value / (100.0 + value)
			return "-%.0f%% cooldowns" % pct
		_:
			var key: String = ["tempo", "identity", "mastery"][stat_id - 3]
			var info: Dictionary = CLASS_STAT_INFO.get(cls, {})
			var entry: Array = info.get(key, [])
			return entry[1] if entry.size() > 1 else ""


## Merge duplicate stat lines and split into [primary, secondary].
## Primary = Hull/Output/Plating, Secondary = Tempo/Identity/Mastery.
static func merge_and_split(stat_lines: Array) -> Array:
	var merged := {}
	for sl in stat_lines:
		var sid: int = sl.get("stat", 0)
		merged[sid] = merged.get(sid, 0.0) + sl.get("value", 0.0)
	var primary := []
	var secondary := []
	for sid in [Stat.HULL, Stat.OUTPUT, Stat.PLATING]:
		if merged.has(sid):
			primary.append({"stat": sid, "value": merged[sid]})
	for sid in [Stat.TEMPO, Stat.IDENTITY, Stat.MASTERY]:
		if merged.has(sid):
			secondary.append({"stat": sid, "value": merged[sid]})
	return [primary, secondary]


## Returns a color based on item level.
static func ilvl_color(ilvl: int) -> Color:
	if ilvl >= 5:
		return Color(0.9, 0.5, 0.1)  # orange
	if ilvl >= 3:
		return Color(0.3, 0.5, 0.9)  # blue
	if ilvl >= 2:
		return Color(0.2, 0.8, 0.2)  # green
	return Color(0.7, 0.7, 0.7)  # grey

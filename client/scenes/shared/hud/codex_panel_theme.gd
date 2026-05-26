class_name CodexPanelTheme
extends RefCounted

## Static theme constants for the Codex Panel: colors, layout metrics, label maps.

# -- Colors --
const BG_OVERLAY := Color(0.0, 0.0, 0.02, 0.7)
const PANEL_BG := Color(0.02, 0.025, 0.035, 0.92)
const PANEL_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const TEXT_PRIMARY := Color(0.92, 0.94, 0.97, 0.95)
const TEXT_MUTED := Color(0.66, 0.7, 0.77, 0.9)
const TEXT_DIM := Color(0.45, 0.48, 0.55, 0.7)
const ACCENT := Color(0.3, 0.65, 0.85)
const TAB_ACTIVE_BG := Color(0.12, 0.15, 0.22, 0.9)
const TAB_HOVER_BG := Color(0.08, 0.1, 0.15, 0.7)
const CLOSE_COLOR := Color(0.8, 0.3, 0.3)
const APPLY_ACTIVE := Color(0.25, 0.7, 0.4)
const APPLY_INACTIVE := Color(0.3, 0.35, 0.4, 0.5)
const UNIMPL_OVERLAY := Color(0.0, 0.0, 0.0, 0.55)
const SLOT_EMPTY_BG := Color(0.04, 0.05, 0.07, 0.6)
const SLOT_FILLED_BG := Color(0.06, 0.08, 0.12, 0.8)
const PRESET_BG := Color(0.06, 0.08, 0.12, 0.7)
const PRESET_HOVER_BG := Color(0.1, 0.13, 0.18, 0.8)
const PRESET_SAVE_COLOR := Color(0.25, 0.7, 0.4)
const PRESET_DELETE_COLOR := Color(0.7, 0.3, 0.3)

const SCHOOL_COLORS := {
	"bioarcanotechnic": Color(0.2, 0.7, 0.65),
	"biometabolic": Color(0.3, 0.75, 0.35),
	"frost": Color(0.4, 0.75, 0.95),
	"fire": Color(0.9, 0.4, 0.2),
	"electricity": Color(0.95, 0.85, 0.3),
	"aerokinetic": Color(0.7, 0.9, 0.95),
	"hydrodynamic": Color(0.3, 0.5, 0.9),
	"pure": Color(0.85, 0.85, 0.9),
	"shadow": Color(0.55, 0.3, 0.7),
	"gravitonic": Color(0.45, 0.25, 0.6),
	"illusion": Color(0.8, 0.45, 0.7),
	"martial": Color(0.7, 0.55, 0.35),
}

const SCHOOL_LABELS := {
	"bioarcanotechnic": "Bioarc.",
	"biometabolic": "Biometa.",
	"frost": "Frost",
	"fire": "Fire",
	"electricity": "Electric",
	"aerokinetic": "Aero.",
	"hydrodynamic": "Hydro.",
	"pure": "Pure",
	"shadow": "Shadow",
	"gravitonic": "Gravit.",
	"illusion": "Illusion",
	"martial": "Martial",
}

const TYPE_LETTERS := {
	"destruction": "D",
	"protection": "P",
	"enhancement": "E",
	"affliction": "A",
	"displacement": "X",
}

const FLUX_DOT_COUNT := {
	"low": 1,
	"medium": 2,
	"high": 3,
	"extreme": 4,
}

const SLOT_KEYBINDS: Array[String] = ["1", "2", "R", "T", "F", "C"]

# -- Layout metrics --
const MARGIN_H := 80.0
const MARGIN_TOP := 50.0
const MARGIN_BOTTOM := 60.0
const TITLE_H := 42.0
const TAB_ROW_H := 28.0
const TAB_GAP := 4.0
const CARD_W := 150.0
const CARD_H := 95.0
const CARD_GAP := 8.0
const LOADOUT_H := 80.0
const LOADOUT_SLOT_W := 130.0
const LOADOUT_SLOT_H := 55.0
const LOADOUT_GAP := 8.0
const APPLY_W := 80.0
const APPLY_H := 32.0
const COMMIT_H := 64.0
const COMMIT_INPUT_W := 48.0
const COMMIT_INPUT_H := 28.0
const PRESET_ROW_H := 28.0
const PRESET_BTN_H := 22.0
const PRESET_GAP := 6.0
const PRESET_SAVE_W := 50.0
const PRESET_NAME_INPUT_W := 120.0
const PRESET_DELETE_SIZE := 14.0

# -- Tooltip font sizes --
const TIP_NAME_SIZE := 18
const TIP_INFO_SIZE := 13
const TIP_STAT_SIZE := 13
const TIP_DETAIL_SIZE := 12
const TIP_AFF_SIZE := 11
const TIP_DESC_SIZE := 13
const TIP_WARN_SIZE := 12
const TIP_LINE_H := 18.0
const TIP_DESC_LINE_H := 17.0


static func get_school_color(school: String) -> Color:
	return SCHOOL_COLORS.get(school, Color(0.5, 0.5, 0.55))


static func wrap_text(font: Font, text: String, max_width: float, font_size: int) -> Array[String]:
	if text == "":
		return []
	var lines: Array[String] = []
	var words := text.split(" ")
	var current_line := ""
	for word in words:
		var test_line := (current_line + " " + word).strip_edges() if current_line != "" else word
		var w := font.get_string_size(test_line, HORIZONTAL_ALIGNMENT_LEFT, -1, font_size).x
		if w > max_width and current_line != "":
			lines.append(current_line)
			current_line = word
		else:
			current_line = test_line
	if current_line != "":
		lines.append(current_line)
	return lines


static func get_slot_for_ability(ability_id: String, loadout: Array[String]) -> int:
	for i in 6:
		if i < loadout.size() and loadout[i] == ability_id:
			return i
	return -1

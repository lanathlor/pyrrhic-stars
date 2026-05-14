class_name TestItemData
extends GdUnitTestSuite

## Tests for ItemData class — enums, name lookups, colors, ilvl_color.

# =============================================================================
# Enum values — must match server/internal/item/item.go
# =============================================================================


func test_stat_enum_hull() -> void:
	assert_int(ItemData.Stat.HULL).is_equal(0)


func test_stat_enum_output() -> void:
	assert_int(ItemData.Stat.OUTPUT).is_equal(1)


func test_stat_enum_plating() -> void:
	assert_int(ItemData.Stat.PLATING).is_equal(2)


func test_stat_enum_tempo() -> void:
	assert_int(ItemData.Stat.TEMPO).is_equal(3)


func test_stat_enum_identity() -> void:
	assert_int(ItemData.Stat.IDENTITY).is_equal(4)


func test_stat_enum_mastery() -> void:
	assert_int(ItemData.Stat.MASTERY).is_equal(5)


func test_slot_enum_frame() -> void:
	assert_int(ItemData.Slot.FRAME).is_equal(0)


func test_slot_enum_power_core() -> void:
	assert_int(ItemData.Slot.POWER_CORE).is_equal(1)


func test_slot_enum_primary_weapon() -> void:
	assert_int(ItemData.Slot.PRIMARY_WEAPON).is_equal(2)


func test_slot_enum_secondary_tool() -> void:
	assert_int(ItemData.Slot.SECONDARY_TOOL).is_equal(3)


func test_slot_enum_augment() -> void:
	assert_int(ItemData.Slot.AUGMENT).is_equal(4)


func test_slot_enum_module() -> void:
	assert_int(ItemData.Slot.MODULE).is_equal(5)


func test_slot_count() -> void:
	assert_int(ItemData.SLOT_COUNT).is_equal(6)


# =============================================================================
# Name lookups
# =============================================================================


func test_stat_names_has_all_six() -> void:
	assert_int(ItemData.STAT_NAMES.size()).is_equal(6)


func test_stat_names_hull() -> void:
	assert_str(ItemData.STAT_NAMES[ItemData.Stat.HULL]).is_equal("Hull")


func test_stat_names_mastery() -> void:
	assert_str(ItemData.STAT_NAMES[ItemData.Stat.MASTERY]).is_equal("Mastery")


func test_slot_names_has_all_six() -> void:
	assert_int(ItemData.SLOT_NAMES.size()).is_equal(6)


func test_slot_names_frame() -> void:
	assert_str(ItemData.SLOT_NAMES[ItemData.Slot.FRAME]).is_equal("Frame")


func test_slot_names_module() -> void:
	assert_str(ItemData.SLOT_NAMES[ItemData.Slot.MODULE]).is_equal("Module")


# =============================================================================
# Stat colors
# =============================================================================


func test_stat_colors_has_all_six() -> void:
	assert_int(ItemData.STAT_COLORS.size()).is_equal(6)


func test_stat_colors_hull_is_green() -> void:
	var c: Color = ItemData.STAT_COLORS[ItemData.Stat.HULL]
	assert_float(c.g).is_greater(c.r)
	assert_float(c.g).is_greater(c.b)


func test_stat_colors_output_is_red() -> void:
	var c: Color = ItemData.STAT_COLORS[ItemData.Stat.OUTPUT]
	assert_float(c.r).is_greater(c.g)
	assert_float(c.r).is_greater(c.b)


# =============================================================================
# ilvl_color
# =============================================================================


func test_ilvl_color_grey_for_level_1() -> void:
	var c := ItemData.ilvl_color(1)
	assert_float(c.r).is_equal_approx(0.7, 0.01)
	assert_float(c.g).is_equal_approx(0.7, 0.01)
	assert_float(c.b).is_equal_approx(0.7, 0.01)


func test_ilvl_color_green_for_level_2() -> void:
	var c := ItemData.ilvl_color(2)
	assert_float(c.g).is_greater(c.r)


func test_ilvl_color_blue_for_level_3() -> void:
	var c := ItemData.ilvl_color(3)
	assert_float(c.b).is_greater(c.r)
	assert_float(c.b).is_greater(c.g)


func test_ilvl_color_blue_for_level_4() -> void:
	var c := ItemData.ilvl_color(4)
	assert_float(c.b).is_greater(c.r)


func test_ilvl_color_orange_for_level_5() -> void:
	var c := ItemData.ilvl_color(5)
	assert_float(c.r).is_greater(c.b)
	assert_float(c.r).is_greater(c.g)


func test_ilvl_color_orange_for_level_10() -> void:
	var c := ItemData.ilvl_color(10)
	assert_float(c.r).is_equal_approx(0.9, 0.01)

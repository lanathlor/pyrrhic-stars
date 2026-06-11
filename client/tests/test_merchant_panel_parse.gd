class_name TestMerchantPanelParse
extends GdUnitTestSuite

## Red test: merchant_panel.gd must compile without parse errors.
## Bug: "var locked := not tier_data.get(...)" fails type inference
## because Dictionary.get() returns Variant, and "not Variant" has no type.

const MerchantScript := preload("res://scenes/ui/merchant_panel.gd")


func test_merchant_panel_compiles() -> void:
	assert_object(MerchantScript).is_not_null()
	# If we get here, the script parsed and compiled successfully.
	var instance: Control = MerchantScript.new()
	assert_object(instance).is_not_null()
	instance.free()

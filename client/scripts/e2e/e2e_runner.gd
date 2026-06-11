extends Node

## Loads and runs E2E scenarios sequentially, writes results, quits.

const E2EContextScript := preload("res://scripts/e2e/e2e_context.gd")
const SCENARIO_DIR := "res://scripts/e2e/scenarios/"
const OUTPUT_DIR := "res://test_output/"

var scenario_names: PackedStringArray = []

var _registry: Dictionary = {}  # name -> script path
var _results: Array[Dictionary] = []


func _ready() -> void:
	_discover_scenarios()
	_run_all.call_deferred()


func _discover_scenarios() -> void:
	var dir := DirAccess.open(SCENARIO_DIR)
	if not dir:
		push_error("[E2E Runner] Cannot open %s" % SCENARIO_DIR)
		return
	dir.list_dir_begin()
	var file := dir.get_next()
	while file != "":
		if file.ends_with(".gd") and not file.begins_with("_"):
			var name := file.get_basename()
			_registry[name] = SCENARIO_DIR + file
		file = dir.get_next()
	print(
		"[E2E Runner] discovered %d scenarios: %s" % [_registry.size(), ", ".join(_registry.keys())]
	)


func _run_all() -> void:
	# Validate requested scenarios exist
	for sname in scenario_names:
		if sname not in _registry:
			push_error("[E2E Runner] Unknown scenario: %s" % sname)
			_write_results()
			get_tree().quit(1)
			return

	var main: Node3D = get_tree().current_scene
	print("[E2E Runner] running %d scenarios" % scenario_names.size())

	for sname in scenario_names:
		print("\n========== E2E: %s ==========" % sname)
		var script: GDScript = load(_registry[sname]) as GDScript
		var scenario: RefCounted = script.new()
		var ctx: RefCounted = E2EContextScript.new(main)
		var start := Time.get_ticks_msec()

		await scenario.run(ctx)

		var duration := (Time.get_ticks_msec() - start) / 1000.0
		var result := {
			"name": sname,
			"passed": ctx.failures.is_empty(),
			"duration": snappedf(duration, 0.1),
			"failures": ctx.failures.duplicate(),
		}
		_results.append(result)

		var status := "PASS" if ctx.failures.is_empty() else "FAIL"
		print("========== %s: %s (%.1fs) ==========\n" % [sname, status, duration])

		# Disconnect from server between scenarios
		NetworkManager.disconnect_game()
		await get_tree().create_timer(0.5).timeout

	_write_results()
	var all_passed := _results.all(func(r: Dictionary) -> bool: return r["passed"])
	get_tree().quit(0 if all_passed else 1)


func _write_results() -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(OUTPUT_DIR))
	var passed := _results.filter(func(r: Dictionary) -> bool: return r["passed"]).size()
	var summary := {
		"passed": passed,
		"failed": _results.size() - passed,
		"total": _results.size(),
		"scenarios": _results,
	}
	var json := JSON.stringify(summary, "  ")
	var path := ProjectSettings.globalize_path(OUTPUT_DIR + "e2e_results.json")
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file:
		file.store_string(json)
	print("[E2E Runner] results written to %s" % path)
	print(json)

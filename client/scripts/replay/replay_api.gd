extends Node
## HTTP client for the combat log REST API.
## Instantiated by the replay browser — NOT an autoload.

signal instances_loaded(data: Array)
signal replay_loaded(replay: Variant)

const BASE_URL := "http://90.29.175.30:7777"
const ReplayDataScript := preload("res://scripts/replay/replay_data.gd")

var _instances_http: HTTPRequest
var _replay_http: HTTPRequest


func _ready() -> void:
	_instances_http = HTTPRequest.new()
	_instances_http.timeout = 15.0
	add_child(_instances_http)
	_instances_http.request_completed.connect(_on_instances_response)

	_replay_http = HTTPRequest.new()
	_replay_http.timeout = 30.0
	add_child(_replay_http)
	_replay_http.request_completed.connect(_on_replay_response)


func fetch_instances() -> void:
	_instances_http.request(BASE_URL + "/api/v1/logs/instances")


func fetch_replay(instance_id: String) -> void:
	_replay_http.request(BASE_URL + "/api/v1/logs/instances/" + instance_id + "/replay")


func _on_instances_response(
	result: int, code: int, _headers: PackedStringArray, body: PackedByteArray
) -> void:
	if result != HTTPRequest.RESULT_SUCCESS or code != 200:
		instances_loaded.emit([])
		return
	var json := JSON.new()
	if json.parse(body.get_string_from_utf8()) != OK:
		instances_loaded.emit([])
		return
	var data: Array = json.data if json.data is Array else []
	instances_loaded.emit(data)


func _on_replay_response(
	result: int, code: int, _headers: PackedStringArray, body: PackedByteArray
) -> void:
	if result != HTTPRequest.RESULT_SUCCESS or code != 200:
		replay_loaded.emit(null)
		return
	var json := JSON.new()
	if json.parse(body.get_string_from_utf8()) != OK:
		replay_loaded.emit(null)
		return
	var replay = ReplayDataScript.from_json(json.data)
	replay_loaded.emit(replay)

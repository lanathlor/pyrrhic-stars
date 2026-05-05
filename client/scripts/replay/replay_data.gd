extends RefCounted
## Parsed replay export from the server API.
## Holds decoded WorldState frames, combat events, and instance metadata.

var instance_id: String
var encounter_id: String
var zone_id: String
var tick_rate: int = 20
var frame_count: int = 0
var duration_ms: int = 0
var outcome: String
var participants: Array[Dictionary] = []
var frames: Array[PackedByteArray] = []
var events: Array[Dictionary] = []

# Derived lookups built in from_json().
var _events_by_tick: Dictionary = {}  # tick(int) -> Array[Dictionary]
var _participant_names: Dictionary = {}  # entity_id(String) -> name(String)


static func from_json(data: Dictionary):
	var script := load("res://scripts/replay/replay_data.gd")
	var rd = script.new()
	rd.instance_id = ReplayUtils.safe_str(data, "instance_id", "")
	rd.encounter_id = ReplayUtils.safe_str(data, "encounter_id", "")
	rd.zone_id = ReplayUtils.safe_str(data, "zone_id", "")
	rd.tick_rate = ReplayUtils.safe_int(data, "tick_rate", 20)
	rd.duration_ms = ReplayUtils.safe_int(data, "duration_ms", 0)
	rd.outcome = ReplayUtils.safe_str(data, "outcome", "")

	# Participants
	var raw_participants: Array = (
		data.get("participants") if data.get("participants") is Array else []
	)
	for p in raw_participants:
		rd.participants.append(p)
		var eid: String = p.get("entity_id", "")
		if eid != "":
			rd._participant_names[eid] = p.get("name", eid)

	# Decode base64 frames to PackedByteArray
	var raw_frames: Array = data.get("frames") if data.get("frames") is Array else []
	for b64 in raw_frames:
		rd.frames.append(Marshalls.base64_to_raw(b64))

	# Derive frame_count from actual decoded frames (don't trust JSON field)
	rd.frame_count = rd.frames.size()

	# Events — build tick index
	var raw_events: Array = data.get("events") if data.get("events") is Array else []
	for ev in raw_events:
		rd.events.append(ev)
		var tick: int = ReplayUtils.safe_int(ev, "tick", 0)
		if tick not in rd._events_by_tick:
			rd._events_by_tick[tick] = []
		rd._events_by_tick[tick].append(ev)

	return rd


func get_frame(index: int) -> PackedByteArray:
	if index >= 0 and index < frames.size():
		return frames[index]
	return PackedByteArray()


func get_events_at_tick(tick: int) -> Array:
	return _events_by_tick.get(tick, [])


func get_participant_name(entity_id: String) -> String:
	return _participant_names.get(entity_id, entity_id)


func tick_to_time_str(tick: int) -> String:
	var seconds: float = float(tick) / float(tick_rate)
	var mins: int = int(seconds) / 60
	var secs: int = int(seconds) % 60
	return "%d:%02d" % [mins, secs]


func total_time_str() -> String:
	var seconds: float = float(duration_ms) / 1000.0
	var mins: int = int(seconds) / 60
	var secs: int = int(seconds) % 60
	return "%d:%02d" % [mins, secs]

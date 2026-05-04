extends Node

## Manages persistent player identity via a locally stored UUID.
## The UUID is generated once and saved to user://player_id.txt.

const PLAYER_ID_PATH := "user://player_id.txt"

var _player_uuid: String = ""


func _ready() -> void:
	_player_uuid = _load_or_create_uuid()
	print("[Identity] Player UUID: %s" % _player_uuid)


func get_player_id() -> String:
	return _player_uuid


func _load_or_create_uuid() -> String:
	if FileAccess.file_exists(PLAYER_ID_PATH):
		var f := FileAccess.open(PLAYER_ID_PATH, FileAccess.READ)
		var uuid := f.get_as_text().strip_edges()
		f.close()
		if uuid.length() == 36:
			return uuid
	var uuid := _generate_uuid_v4()
	var f := FileAccess.open(PLAYER_ID_PATH, FileAccess.WRITE)
	f.store_string(uuid)
	f.close()
	return uuid


func _generate_uuid_v4() -> String:
	var rng := RandomNumberGenerator.new()
	rng.randomize()
	var bytes: PackedByteArray = PackedByteArray()
	bytes.resize(16)
	for i in range(16):
		bytes[i] = rng.randi_range(0, 255)
	# Set version 4 (0100) in byte 6 high nibble
	bytes[6] = (bytes[6] & 0x0F) | 0x40
	# Set variant 1 (10xx) in byte 8 high bits
	bytes[8] = (bytes[8] & 0x3F) | 0x80
	return (
		"%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x"
		% [
			bytes[0],
			bytes[1],
			bytes[2],
			bytes[3],
			bytes[4],
			bytes[5],
			bytes[6],
			bytes[7],
			bytes[8],
			bytes[9],
			bytes[10],
			bytes[11],
			bytes[12],
			bytes[13],
			bytes[14],
			bytes[15],
		]
	)

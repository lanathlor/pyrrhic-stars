class_name ReplayUtils
## Shared utility functions for the replay system.


static func safe_int(d: Dictionary, key: String, fallback: int) -> int:
	var v = d.get(key)
	if v is int:
		return v
	if v is float:
		return int(v)
	return fallback


static func safe_str(d: Dictionary, key: String, fallback: String) -> String:
	var v = d.get(key)
	if v is String:
		return v
	return fallback

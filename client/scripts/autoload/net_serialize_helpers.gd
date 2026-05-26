class_name NetSerializeHelpers
## Shared binary buffer helpers for net serializer codec files.
## All methods are static so codec files can call them without instantiation.


static func put_vec3(buf: StreamPeerBuffer, v: Vector3) -> void:
	buf.put_float(v.x)
	buf.put_float(v.y)
	buf.put_float(v.z)


static func get_vec3(buf: StreamPeerBuffer) -> Vector3:
	return Vector3(buf.get_float(), buf.get_float(), buf.get_float())


static func put_str8(buf: StreamPeerBuffer, s: String) -> void:
	var bytes := s.to_utf8_buffer()
	buf.put_u8(bytes.size())
	if bytes.size() > 0:
		buf.put_data(bytes)


static func get_str8(buf: StreamPeerBuffer) -> String:
	var length := buf.get_u8()
	if length == 0:
		return ""
	var bytes := buf.get_data(length)
	if bytes[0] != OK:
		return ""
	return (bytes[1] as PackedByteArray).get_string_from_utf8()

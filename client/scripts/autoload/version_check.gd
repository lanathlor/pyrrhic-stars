extends Node

## Pre-login client/server version gate.
##
## Release builds are stamped with the run's projected release tag (CI seds
## application/config/version, which carries the "dev" placeholder in-repo).
## The client must run the exact tag the stamped server reports: nearly every
## release changes the wire protocol, so there is no cross-version play.
## Unstamped builds (editor, PR exports) skip the gate; a stamped gateway
## rejects them at the WS handshake instead.

enum Status { OK, MISMATCH, UNREACHABLE }

const VERSION_SETTING := "application/config/version"
const BASE_SETTING := "pyrrhic/update/download_base"
const REQUEST_TIMEOUT := 5.0

## Release asset name per OS.get_name(); platforms without a released binary
## (Web, macOS) get no self-update offer.
const ASSETS := {
	"Linux": "pyrrhic-stars-linux.x86_64",
	"Windows": "pyrrhic-stars-windows.exe",
}


func _ready() -> void:
	# A Windows self-update renames the running exe aside (see update_overlay.gd);
	# the replaced binary can only be deleted on the next launch.
	if FileAccess.file_exists(OS.get_executable_path() + ".old"):
		DirAccess.remove_absolute(OS.get_executable_path() + ".old")


## The stamped release tag, or "" for an unstamped (editor/PR/dev) build.
func client_version() -> String:
	var v: String = str(ProjectSettings.get_setting(VERSION_SETTING, ""))
	return "" if v == "dev" else v


## Exact-match gate: only a stamped client talking to a stamped server of a
## different tag needs an update.
func is_update_required(client_v: String, server_v: String) -> bool:
	return client_v != "" and server_v != "" and client_v != server_v


## GitHub release asset URL for the given server tag, "" when this build has
## no stamped download base or the platform has no released binary.
func download_url(server_version: String, platform: String = OS.get_name()) -> String:
	var base: String = str(ProjectSettings.get_setting(BASE_SETTING, ""))
	if base == "" or not ASSETS.has(platform):
		return ""
	return "%s/%s/%s" % [base, server_version, ASSETS[platform]]


## Query-string suffix for the WS handshake; the stamped gateway rejects the
## upgrade (426) when the tag does not match its own.
func ws_version_param() -> String:
	var v := client_version()
	return "" if v == "" else "&v=%s" % v.uri_encode()


## Interprets a GET /version response body.
func parse_response(code: int, body: PackedByteArray) -> Dictionary:
	if code != 200:
		return {"ok": false, "version": ""}
	var json: Variant = JSON.parse_string(body.get_string_from_utf8())
	if typeof(json) != TYPE_DICTIONARY or typeof(json.get("version")) != TYPE_STRING:
		return {"ok": false, "version": ""}
	return {"ok": true, "version": json["version"]}


## Asks the gateway which release it runs. Returns {status: Status,
## server_version: String}. Unstamped clients skip the request entirely;
## an unreachable or unstamped server never blocks login here (the regular
## connection error paths handle that).
func check(host: String) -> Dictionary:
	if client_version() == "":
		return {"status": Status.OK, "server_version": ""}
	var url := "%s/version" % ServerConfig.gateway_http_base(host)
	var result := await _fetch(url)
	if not result.ok:
		return {"status": Status.UNREACHABLE, "server_version": ""}
	if is_update_required(client_version(), result.version):
		return {"status": Status.MISMATCH, "server_version": result.version}
	return {"status": Status.OK, "server_version": result.version}


func _fetch(url: String) -> Dictionary:
	var http := HTTPRequest.new()
	http.timeout = REQUEST_TIMEOUT
	add_child(http)
	var parsed := {"ok": false, "version": ""}
	if http.request(url) == OK:
		var res: Array = await http.request_completed
		# res: [result, response_code, headers, body]
		if res[0] == HTTPRequest.RESULT_SUCCESS:
			parsed = parse_response(res[1], res[3])
	http.queue_free()
	return parsed

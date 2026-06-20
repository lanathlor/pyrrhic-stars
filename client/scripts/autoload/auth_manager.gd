extends Node

## Talks to Ory Kratos over HTTP using the native "API" self-service flows and
## holds the resulting session token. The Godot WebSocketPeer cannot send
## headers/cookies, so the token is passed to the gateway as a ?token= query
## param on connect (see NetworkManager.connect_to_server).

signal auth_succeeded
signal auth_failed(message: String)

const TOKEN_SAVE_PATH := "user://session_token.txt"

var session_token: String = ""

var _http: HTTPRequest


func _ready() -> void:
	_http = HTTPRequest.new()
	add_child(_http)
	session_token = _load_token()


func get_token() -> String:
	return session_token


func has_token() -> bool:
	return session_token != ""


func clear_token() -> void:
	session_token = ""
	if FileAccess.file_exists(TOKEN_SAVE_PATH):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(TOKEN_SAVE_PATH))


## Logs in with email + password. Emits auth_succeeded or auth_failed.
func login(host: String, email: String, password: String) -> void:
	var base := _base_url(host)
	var flow: Dictionary = await _begin_flow(base, "login")
	if flow.is_empty():
		auth_failed.emit("Could not reach auth server")
		return
	var body := {
		"method": "password",
		"identifier": email,
		"password": password,
	}
	await _submit_flow(base, "login", flow.get("id", ""), body)


## Registers a new account, then is immediately logged in via the Kratos
## session hook. Emits auth_succeeded or auth_failed.
func register(host: String, email: String, password: String, username: String) -> void:
	var base := _base_url(host)
	var flow: Dictionary = await _begin_flow(base, "registration")
	if flow.is_empty():
		auth_failed.emit("Could not reach auth server")
		return
	var body := {
		"method": "password",
		"password": password,
		"traits": {"email": email, "username": username},
	}
	await _submit_flow(base, "registration", flow.get("id", ""), body)


# =============================================================================
# Internal HTTP plumbing
# =============================================================================


func _base_url(host: String) -> String:
	return ServerConfig.auth_base(host)


## GET /self-service/<kind>/api -> returns the flow JSON (with its id).
func _begin_flow(base: String, kind: String) -> Dictionary:
	var url := "%s/self-service/%s/api" % [base, kind]
	var result: Dictionary = await _do_request(url, HTTPClient.METHOD_GET, {})
	if result.get("code", 0) != 200:
		return {}
	return result.get("json", {})


## POST /self-service/<kind>?flow=<id> with the credential body, then parse the
## session token (or an error message).
func _submit_flow(base: String, kind: String, flow_id: String, body: Dictionary) -> void:
	if flow_id == "":
		auth_failed.emit("Auth server returned no flow")
		return
	var url := "%s/self-service/%s?flow=%s" % [base, kind, flow_id]
	var result: Dictionary = await _do_request(url, HTTPClient.METHOD_POST, body)
	var json: Dictionary = result.get("json", {})
	var code: int = result.get("code", 0)
	if code == 200 and json.has("session_token"):
		_set_token(String(json["session_token"]))
		auth_succeeded.emit()
		return
	auth_failed.emit(_extract_error(json, code))


## Performs one request and awaits completion. Returns {code, json}.
func _do_request(url: String, method: int, body: Dictionary) -> Dictionary:
	var headers := ["Content-Type: application/json", "Accept: application/json"]
	var payload := JSON.stringify(body) if not body.is_empty() else ""
	var err := _http.request(url, headers, method, payload)
	if err != OK:
		push_warning("[Auth] request error: %s" % error_string(err))
		return {"code": 0, "json": {}}
	var res: Array = await _http.request_completed
	# res = [result, response_code, headers, body]
	var code: int = res[1]
	var text := (res[3] as PackedByteArray).get_string_from_utf8()
	var parsed: Variant = JSON.parse_string(text)
	var json: Dictionary = parsed if parsed is Dictionary else {}
	return {"code": code, "json": json}


## Walks the Kratos UI error structure for the first human-readable message.
func _extract_error(json: Dictionary, code: int) -> String:
	var ui: Dictionary = json.get("ui", {})
	for msg in ui.get("messages", []):
		if msg is Dictionary and msg.get("type", "") == "error":
			return String(msg.get("text", ""))
	for node in ui.get("nodes", []):
		if node is Dictionary:
			for msg in node.get("messages", []):
				if msg is Dictionary and msg.get("type", "") == "error":
					return String(msg.get("text", ""))
	if json.has("error"):
		var e: Dictionary = json.get("error", {})
		return String(e.get("message", "Authentication failed"))
	return "Authentication failed (%d)" % code


# =============================================================================
# Token persistence
# =============================================================================


func _set_token(token: String) -> void:
	session_token = token
	var f := FileAccess.open(TOKEN_SAVE_PATH, FileAccess.WRITE)
	if f != null:
		f.store_string(token)
		f.close()


func _load_token() -> String:
	if not FileAccess.file_exists(TOKEN_SAVE_PATH):
		return ""
	var f := FileAccess.open(TOKEN_SAVE_PATH, FileAccess.READ)
	if f == null:
		return ""
	var token := f.get_as_text().strip_edges()
	f.close()
	return token

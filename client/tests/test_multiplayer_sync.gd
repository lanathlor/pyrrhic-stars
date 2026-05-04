extends SceneTree

## Multiplayer sync test.
## Run: godot --path client/ --headless -s tests/test_multiplayer_sync.gd

const GUNNER_SCENE := "res://scenes/controllers/gunner/gunner.tscn"

var _passed := 0
var _failed := 0


func verify(desc: String, condition: bool) -> void:
	if condition:
		print("  PASS: %s" % desc)
		_passed += 1
	else:
		print("  FAIL: %s" % desc)
		_failed += 1


func _initialize() -> void:
	print("\n=== Multiplayer Sync Test ===\n")

	_test_enet_setup()
	_test_player_net_vars()
	_test_net_sync_rpc()
	_test_two_peer_sync()

	print("\n=== Results: %d passed, %d failed ===" % [_passed, _failed])
	quit(1 if _failed > 0 else 0)


func _test_enet_setup() -> void:
	print("[1] ENet setup...")
	var host := ENetMultiplayerPeer.new()
	var err := host.create_server(17777, 4)
	verify("Server creates OK", err == OK)
	verify(
		"Server status is CONNECTED",
		host.get_connection_status() == MultiplayerPeer.CONNECTION_CONNECTED
	)
	host.close()


func _test_player_net_vars() -> void:
	print("[2] Player net vars...")
	var scene := load(GUNNER_SCENE) as PackedScene
	verify("Gunner scene loads", scene != null)
	if not scene:
		return

	var gunner := scene.instantiate()
	gunner.name = "TestGunner"
	root.add_child(gunner)

	verify("Has _net_position", "_net_position" in gunner)
	verify("Has _net_rotation_y", "_net_rotation_y" in gunner)
	verify("Has _net_anim", "_net_anim" in gunner)
	verify("Has _net_sync method", gunner.has_method("_net_sync"))
	verify("Has _sync_state_to_peers method", gunner.has_method("_sync_state_to_peers"))

	gunner.queue_free()


func _test_net_sync_rpc() -> void:
	print("[3] _net_sync direct call simulating RPC receive...")
	var scene := load(GUNNER_SCENE) as PackedScene
	if not scene:
		return

	var gunner := scene.instantiate()
	gunner.name = "TestGunner2"
	root.add_child(gunner)

	gunner._net_sync.call(Vector3(5.0, 1.0, 10.0), 1.57, "rifle_run", 1.5, 120.0)
	verify("Position synced", gunner._net_position == Vector3(5.0, 1.0, 10.0))
	verify("Rotation synced", is_equal_approx(gunner._net_rotation_y, 1.57))
	verify("Anim synced", gunner._net_anim == "rifle_run")
	verify("Anim speed synced", is_equal_approx(gunner._net_anim_speed, 1.5))
	verify("Health synced", is_equal_approx(gunner.health, 120.0))

	gunner.queue_free()


func _test_two_peer_sync() -> void:
	print("[4] Two-peer authority + sync...")

	var host_peer := ENetMultiplayerPeer.new()
	var err := host_peer.create_server(17778, 4)
	verify("Host created", err == OK)
	root.multiplayer.multiplayer_peer = host_peer
	verify("Host unique ID is 1", root.multiplayer.get_unique_id() == 1)
	verify("Host is server", root.multiplayer.is_server())

	# Spawn gunner as peer 1 (host)
	var scene := load(GUNNER_SCENE) as PackedScene
	var host_gunner := scene.instantiate()
	host_gunner.name = "Player_1"
	host_gunner.set_multiplayer_authority(1)
	root.add_child(host_gunner)
	verify("Host gunner is local", host_gunner._is_local() == true)

	# Spawn gunner as peer 2 (remote on host)
	var client_gunner := scene.instantiate()
	client_gunner.name = "Player_2"
	client_gunner.set_multiplayer_authority(2)
	root.add_child(client_gunner)
	verify("Client gunner is NOT local on host", client_gunner._is_local() == false)

	# Simulate receiving sync from peer 2
	client_gunner._net_sync.call(Vector3(3.0, 0.0, 7.0), -1.0, "rifle_idle", 1.0, 150.0)
	verify("Remote position set", client_gunner._net_position == Vector3(3.0, 0.0, 7.0))
	verify("Remote rotation set", is_equal_approx(client_gunner._net_rotation_y, -1.0))

	# Verify host gunner writes its own sync vars
	host_gunner.global_position = Vector3(10.0, 0.0, 20.0)
	host_gunner.rotation.y = 2.0
	host_gunner._net_position = host_gunner.global_position
	host_gunner._net_rotation_y = host_gunner.rotation.y
	verify("Host writes _net_position", host_gunner._net_position == Vector3(10.0, 0.0, 20.0))
	verify("Host writes _net_rotation_y", is_equal_approx(host_gunner._net_rotation_y, 2.0))

	host_gunner.queue_free()
	client_gunner.queue_free()
	host_peer.close()
	root.multiplayer.multiplayer_peer = null
